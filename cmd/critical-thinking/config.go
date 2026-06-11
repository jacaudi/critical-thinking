package main

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// newConfigViper builds the single Viper instance backing all runtime config.
// Keys use underscores so AutomaticEnv maps them to CTHINK_<KEY> with no
// key-replacer (e.g. log_format → CTHINK_LOG_FORMAT). The env-only keys
// (allowed_origins, http_host) resolve via AutomaticEnv; the flag-backed keys
// (verbose, log_format) are bound separately via bindFlags. Precedence is
// flag (if changed) > env > default.
func newConfigViper() *viper.Viper {
	v := viper.New()
	v.SetEnvPrefix("CTHINK")
	v.AutomaticEnv()
	v.SetDefault("http_host", "127.0.0.1")
	return v
}

// bindFlags binds the logging persistent flags to their Viper keys so a passed
// flag overrides env overrides default. flags is the running command's flag set
// (it inherits the root persistent flags). It returns an error only if a named
// flag is absent from flags — BindPFlag rejects a nil *pflag.Flag.
func bindFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	if err := v.BindPFlag("verbose", flags.Lookup("verbose")); err != nil {
		return err
	}
	return v.BindPFlag("log_format", flags.Lookup("log-format"))
}

// httpConfig is the HTTP server's resolved configuration.
type httpConfig struct {
	AllowedOrigins []string
	HTTPHost       string
}

// httpConfigFromViper extracts the HTTP settings, reusing parseAllowedOrigins as
// the single origin parser (NOT viper.GetStringSlice, whose splitting differs).
func httpConfigFromViper(v *viper.Viper) httpConfig {
	return httpConfig{
		AllowedOrigins: parseAllowedOrigins(v.GetString("allowed_origins")),
		HTTPHost:       v.GetString("http_host"),
	}
}
