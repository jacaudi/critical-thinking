package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// bearerToken extracts the token from an Authorization header value. The scheme match is
// case-insensitive per RFC 6750/7235. The token must be non-empty and contain no whitespace.
func bearerToken(header string) (string, bool) {
	const scheme = "bearer "
	if len(header) < len(scheme) || !strings.EqualFold(header[:len(scheme)], scheme) {
		return "", false
	}
	tok := strings.TrimSpace(header[len(scheme):])
	if tok == "" || strings.ContainsAny(tok, " \t") {
		return "", false
	}
	return tok, true
}

// writeUnauthorized emits a 401 with an RFC 6750 challenge and the given publicMessage as the
// body. Callers MUST pass only a generic, non-sensitive message — never a raw bearer token or a
// verifier's internal error string — this function performs no sanitization of its own.
func writeUnauthorized(w http.ResponseWriter, publicMessage string) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	http.Error(w, publicMessage, http.StatusUnauthorized)
}

// newOIDCVerifier performs OIDC discovery against issuer and returns a verifier that enforces
// signature, issuer, audience (via the aud claim), and expiry. Discovery is a network call, run
// under a bounded context so a slow-loris / black-hole IdP cannot hang startup indefinitely (the
// signal context passed by runHTTP has no deadline). A returned error must abort startup. The
// Skip* options are deliberately never set — setting any of them would silently weaken the control.
func newOIDCVerifier(ctx context.Context, issuer, audience string) (*oidc.IDTokenVerifier, error) {
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(dctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery for issuer %q: %w", issuer, err)
	}
	return provider.Verifier(&oidc.Config{ClientID: audience}), nil
}

// requireAuth gates next behind a valid bearer token. Applied ONLY to /mcp. Any verification
// failure — bad signature, wrong aud/iss, expired, or JWKS unavailable for an uncached key —
// fails closed with 401. The internal error is logged at Debug only, never sent to the client.
func requireAuth(verifier *oidc.IDTokenVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeUnauthorized(w, "missing or malformed bearer token")
			return
		}
		if _, err := verifier.Verify(r.Context(), raw); err != nil {
			slog.Debug("bearer token rejected", "error", err)
			writeUnauthorized(w, "invalid token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
