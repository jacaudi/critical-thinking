package main

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestNewConfigViperDefaults(t *testing.T) {
	v := newConfigViper()
	if got := v.GetString("http_host"); got != "127.0.0.1" {
		t.Errorf("http_host default = %q, want 127.0.0.1", got)
	}
	if got := v.GetString("allowed_origins"); got != "" {
		t.Errorf("allowed_origins default = %q, want empty", got)
	}
}

func TestHTTPConfigFromViper(t *testing.T) {
	t.Setenv("CTHINK_ALLOWED_ORIGINS", "https://a.example, https://b.example")
	t.Setenv("CTHINK_HTTP_HOST", "0.0.0.0")
	v := newConfigViper()
	cfg := httpConfigFromViper(v)
	if cfg.HTTPHost != "0.0.0.0" {
		t.Errorf("HTTPHost = %q, want 0.0.0.0", cfg.HTTPHost)
	}
	want := []string{"https://a.example", "https://b.example"}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != want[0] || cfg.AllowedOrigins[1] != want[1] {
		t.Errorf("AllowedOrigins = %v, want %v", cfg.AllowedOrigins, want)
	}
}

func TestHTTPConfigFromViperEmpty(t *testing.T) {
	v := newConfigViper()
	cfg := httpConfigFromViper(v)
	if cfg.AllowedOrigins != nil {
		t.Errorf("AllowedOrigins = %v, want nil when unset", cfg.AllowedOrigins)
	}
	if cfg.HTTPHost != "127.0.0.1" {
		t.Errorf("HTTPHost = %q, want default 127.0.0.1", cfg.HTTPHost)
	}
}

// newLoggingFlags builds a flag set with the same two persistent flags the root
// defines, so bindFlags can be exercised in isolation.
func newLoggingFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.Bool("verbose", false, "")
	fs.String("log-format", "text", "")
	return fs
}

func TestBindFlagsEnvWinsWhenFlagUnchanged(t *testing.T) {
	t.Setenv("CTHINK_LOG_FORMAT", "json")
	t.Setenv("CTHINK_VERBOSE", "true")
	v := newConfigViper()
	fs := newLoggingFlags()
	if err := bindFlags(v, fs); err != nil {
		t.Fatal(err)
	}
	// fs not parsed → flags unchanged → env wins.
	if got := v.GetString("log_format"); got != "json" {
		t.Errorf("log_format = %q, want json (env)", got)
	}
	if !v.GetBool("verbose") {
		t.Error("verbose = false, want true (env)")
	}
}

func TestBindFlagsFlagWinsOverEnv(t *testing.T) {
	t.Setenv("CTHINK_LOG_FORMAT", "json")
	// Parse BEFORE binding, mirroring cobra (flags parsed before PersistentPreRunE).
	fs := newLoggingFlags()
	if err := fs.Parse([]string{"--log-format", "text"}); err != nil {
		t.Fatal(err)
	}
	v := newConfigViper()
	if err := bindFlags(v, fs); err != nil {
		t.Fatal(err)
	}
	if got := v.GetString("log_format"); got != "text" {
		t.Errorf("log_format = %q, want text (flag overrides env)", got)
	}
}

func TestBindFlagsDefaultWhenNeither(t *testing.T) {
	v := newConfigViper()
	fs := newLoggingFlags()
	if err := bindFlags(v, fs); err != nil {
		t.Fatal(err)
	}
	if got := v.GetString("log_format"); got != "text" {
		t.Errorf("log_format = %q, want text (default)", got)
	}
	if v.GetBool("verbose") {
		t.Error("verbose = true, want false (default)")
	}
}

func TestHTTPConfigFromViperOIDC(t *testing.T) {
	t.Setenv("CTHINK_OIDC_ISSUER", "https://issuer.example")
	t.Setenv("CTHINK_OIDC_AUDIENCE", "critical-thinking")
	v := newConfigViper()
	cfg := httpConfigFromViper(v)
	if cfg.OIDCIssuer != "https://issuer.example" {
		t.Errorf("OIDCIssuer = %q, want https://issuer.example", cfg.OIDCIssuer)
	}
	if cfg.OIDCAudience != "critical-thinking" {
		t.Errorf("OIDCAudience = %q, want critical-thinking", cfg.OIDCAudience)
	}
}

func TestHTTPConfigFromViperOIDCTrimsWhitespace(t *testing.T) {
	t.Setenv("CTHINK_OIDC_ISSUER", "  https://issuer.example  ")
	t.Setenv("CTHINK_OIDC_AUDIENCE", "   ") // whitespace-only → must collapse to empty
	v := newConfigViper()
	cfg := httpConfigFromViper(v)
	if cfg.OIDCIssuer != "https://issuer.example" {
		t.Errorf("OIDCIssuer = %q, want trimmed https://issuer.example", cfg.OIDCIssuer)
	}
	if cfg.OIDCAudience != "" {
		t.Errorf("OIDCAudience = %q, want empty after trim", cfg.OIDCAudience)
	}
}

func TestOtelEnabledDefaultsFalse(t *testing.T) {
	v := newConfigViper()
	if v.GetBool("otel_enabled") {
		t.Fatal("otel_enabled should default to false")
	}
}

func TestOtelEnabledFromEnv(t *testing.T) {
	t.Setenv("CTHINK_OTEL_ENABLED", "true")
	v := newConfigViper()
	if !v.GetBool("otel_enabled") {
		t.Fatal("CTHINK_OTEL_ENABLED=true not picked up")
	}
}

func TestValidateAuth(t *testing.T) {
	tests := []struct {
		name    string
		cfg     httpConfig
		wantErr bool
	}{
		{"both empty (disabled)", httpConfig{}, false},
		{"both set", httpConfig{OIDCIssuer: "https://i.example", OIDCAudience: "aud"}, false},
		{"issuer set, audience empty", httpConfig{OIDCIssuer: "https://i.example"}, true},
		{"audience set, issuer empty (disabled)", httpConfig{OIDCAudience: "aud"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validateAuth()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
