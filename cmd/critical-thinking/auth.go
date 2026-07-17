package main

import (
	"net/http"
	"strings"
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
