package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		wantTok string
		wantOK  bool
	}{
		{"valid", "Bearer abc.def.ghi", "abc.def.ghi", true},
		{"lowercase scheme", "bearer abc.def.ghi", "abc.def.ghi", true},
		{"mixed-case scheme", "BeArEr abc.def.ghi", "abc.def.ghi", true},
		{"trailing space trimmed", "Bearer abc.def.ghi ", "abc.def.ghi", true},
		{"empty header", "", "", false},
		{"scheme only", "Bearer", "", false},
		{"scheme and space no token", "Bearer ", "", false},
		{"wrong scheme", "Basic abc", "", false},
		{"embedded space", "Bearer abc def", "", false},
		{"embedded tab", "Bearer abc\tdef", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, ok := bearerToken(tt.header)
			if ok != tt.wantOK || tok != tt.wantTok {
				t.Errorf("bearerToken(%q) = (%q, %v), want (%q, %v)", tt.header, tok, ok, tt.wantTok, tt.wantOK)
			}
		})
	}
}

func TestWriteUnauthorized(t *testing.T) {
	rec := httptest.NewRecorder()
	writeUnauthorized(rec, "invalid token")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != `Bearer error="invalid_token"` {
		t.Errorf("WWW-Authenticate = %q, want Bearer error=\"invalid_token\"", got)
	}
}

// fakeIdP is an httptest-backed OIDC provider: it serves a discovery doc and a JWKS derived
// from a locally generated RSA key, and can mint signed JWTs for tests.
type fakeIdP struct {
	server *httptest.Server
	key    *rsa.PrivateKey
	kid    string
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	idp := &fakeIdP{key: key, kid: "test-key-1"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                idp.server.URL,
			"jwks_uri":                              idp.server.URL + "/jwks",
			"authorization_endpoint":                idp.server.URL + "/auth",
			"token_endpoint":                        idp.server.URL + "/token",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": idp.kid, "n": n, "e": e},
			},
		})
	})
	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)
	return idp
}

func (idp *fakeIdP) issuer() string { return idp.server.URL }

// mint signs an RS256 JWT with the given claims using idp's key (or override, if non-nil).
func (idp *fakeIdP) mint(t *testing.T, claims map[string]any, override *rsa.PrivateKey) string {
	t.Helper()
	signKey := idp.key
	if override != nil {
		signKey = override
	}
	header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": idp.kid}
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	signingInput := enc(header) + "." + enc(claims)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, signKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func validClaims(idp *fakeIdP, aud string) map[string]any {
	now := time.Now()
	return map[string]any{
		"iss": idp.issuer(),
		"sub": "user-1",
		"aud": aud,
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
	}
}

// stubOK is the protected handler; a 200 proves auth let the request through.
func stubOK(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func TestRequireAuth(t *testing.T) {
	const audience = "critical-thinking"
	idp := newFakeIdP(t)
	verifier, err := newOIDCVerifier(context.Background(), idp.issuer(), audience)
	if err != nil {
		t.Fatalf("newOIDCVerifier: %v", err)
	}
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	tests := []struct {
		name     string
		authFn   func() string // returns the Authorization header value
		wantCode int
	}{
		{"valid", func() string { return "Bearer " + idp.mint(t, validClaims(idp, audience), nil) }, http.StatusOK},
		{"no header", func() string { return "" }, http.StatusUnauthorized},
		{"malformed scheme", func() string { return "Basic xyz" }, http.StatusUnauthorized},
		{"wrong audience", func() string { return "Bearer " + idp.mint(t, validClaims(idp, "other-service"), nil) }, http.StatusUnauthorized},
		{"expired", func() string {
			c := validClaims(idp, audience)
			c["exp"] = time.Now().Add(-time.Hour).Unix()
			return "Bearer " + idp.mint(t, c, nil)
		}, http.StatusUnauthorized},
		{"wrong signing key", func() string { return "Bearer " + idp.mint(t, validClaims(idp, audience), otherKey) }, http.StatusUnauthorized},
		{"wrong issuer", func() string {
			c := validClaims(idp, audience)
			c["iss"] = "https://attacker.example"
			return "Bearer " + idp.mint(t, c, nil)
		}, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := requireAuth(verifier, http.HandlerFunc(stubOK))
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if v := tt.authFn(); v != "" {
				req.Header.Set("Authorization", v)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

func TestNewOIDCVerifierDiscoveryFailure(t *testing.T) {
	// An unreachable issuer must return an error (fail fast at startup).
	_, err := newOIDCVerifier(context.Background(), "http://127.0.0.1:1/nope", "aud")
	if err == nil {
		t.Fatal("expected discovery error for unreachable issuer, got nil")
	}
}
