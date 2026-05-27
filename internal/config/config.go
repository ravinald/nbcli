// Package config layers configuration the viper way:
//
//	flag > env (NBCLI_*) > config file > built-in default
//
// The Netbox token is intentionally treated as a secret: env-only by default,
// never written to a config file by this package.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// errNoURL is returned when no URL was configured for live API calls.
var errNoURL = errors.New("config: url is required (set NBCLI_URL, --url, or url: in config)")

// EnvPrefix prefixes every env var nbcli reads, e.g. NBCLI_URL, NBCLI_FORMAT.
const EnvPrefix = "NBCLI"

// TokenEnv is the env var name we read the Netbox API token from. We honor
// the conventional NETBOX_TOKEN name too — set either.
const (
	TokenEnv       = "NBCLI_TOKEN"  //nolint:gosec // env var name, not a credential
	TokenEnvLegacy = "NETBOX_TOKEN" //nolint:gosec // env var name, not a credential
)

// Config is the resolved runtime configuration. Anything sensitive (tokens)
// is loaded but never serialized back to disk by Save().
type Config struct {
	// URL is the Netbox base URL, e.g. "https://netbox.example.com".
	URL string `mapstructure:"url" yaml:"url"`

	// Token is the Netbox API token. Format per project convention:
	// "nbt_${KEY}.${TOKEN}". Sourced from env, never persisted.
	Token string `mapstructure:"-" yaml:"-"`

	// Format is the default output format ("table", "json", "yaml", "tsv").
	// Empty string means "let the renderer pick based on whether stdout is a TTY".
	Format string `mapstructure:"format" yaml:"format,omitempty"`

	// Timeout for HTTP requests, in seconds.
	TimeoutSeconds int `mapstructure:"timeout_seconds" yaml:"timeout_seconds,omitempty"`

	// InsecureSkipVerify disables TLS cert verification. Off by default.
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify" yaml:"insecure_skip_verify,omitempty"`

	// ConfigFile is the resolved path of the config file that was loaded
	// (empty if none was found).
	ConfigFile string `mapstructure:"-" yaml:"-"`
}

// Defaults returns a Config with built-in defaults applied. Flag/env/file
// values overlay on top of these.
func Defaults() Config {
	return Config{
		TimeoutSeconds: 30,
	}
}

// Load resolves configuration using viper's standard precedence:
//
//  1. flags bound via BindPFlags (highest)
//  2. NBCLI_* env vars
//  3. config file at --config or $XDG_CONFIG_HOME/nbcli/config.yaml
//  4. defaults
//
// configFile, when non-empty, forces that exact path. Otherwise nbcli looks
// in $XDG_CONFIG_HOME/nbcli/ then $HOME/.config/nbcli/.
func Load(flags *pflag.FlagSet, configFile string) (Config, error) {
	v := viper.New()
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	for k, val := range defaultsMap() {
		v.SetDefault(k, val)
	}

	if flags != nil {
		if err := v.BindPFlags(flags); err != nil {
			return Config{}, fmt.Errorf("config: bind flags: %w", err)
		}
	}

	switch {
	case configFile != "":
		v.SetConfigFile(configFile)
	default:
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		for _, dir := range searchPaths() {
			v.AddConfigPath(dir)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && configFile != "" {
			return Config{}, fmt.Errorf("config: read %s: %w", configFile, err)
		}
	}

	cfg := Defaults()
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: unmarshal: %w", err)
	}
	cfg.ConfigFile = v.ConfigFileUsed()

	cfg.Token = firstNonEmpty(os.Getenv(TokenEnv), os.Getenv(TokenEnvLegacy))

	return cfg, nil
}

// Validate returns an error if the resolved config is unusable for live calls.
// Token is checked separately because read-only/local commands shouldn't need it.
func (c Config) Validate() error {
	if c.URL == "" {
		return errNoURL
	}
	return nil
}

// RequireToken returns an error if no token was found in env.
func (c Config) RequireToken() error {
	if c.Token == "" {
		return fmt.Errorf("config: no token found in %s or %s", TokenEnv, TokenEnvLegacy)
	}
	return nil
}

func defaultsMap() map[string]any {
	d := Defaults()
	return map[string]any{
		"url":                  d.URL,
		"format":               d.Format,
		"timeout_seconds":      d.TimeoutSeconds,
		"insecure_skip_verify": d.InsecureSkipVerify,
	}
}

func searchPaths() []string {
	var paths []string
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		paths = append(paths, filepath.Join(x, "nbcli"))
	}
	if h, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(h, ".config", "nbcli"))
	}
	return paths
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
