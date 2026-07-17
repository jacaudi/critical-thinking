package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
