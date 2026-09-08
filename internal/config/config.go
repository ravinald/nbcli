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
	"gopkg.in/yaml.v3"
)

// errNoURL is returned when no URL was configured for live API calls.
var errNoURL = errors.New("config: url is required (set NBCLI_URL, --url, or url: in config)")

// EnvPrefix prefixes every env var nbcli reads, e.g. NBCLI_URL, NBCLI_FORMAT.
const EnvPrefix = "NBCLI"

// Env var names that can supply the Netbox API token. Listed here so the
// "no token found" error can name them all and so test code can clear them
// uniformly.
const (
	TokenEnv       = "NBCLI_TOKEN"         //nolint:gosec // env var name, not a credential
	TokenEnvLegacy = "NETBOX_TOKEN"        //nolint:gosec // env var name, not a credential
	TokenKeyV2     = "NETBOX_API_V2_KEY"   //nolint:gosec // env var name, not a credential
	TokenSecretV2  = "NETBOX_API_V2_TOKEN" //nolint:gosec // env var name, not a credential
)

// tokenEnvNames is the precedence list of env vars that already hold a
// fully-formed Netbox token (highest priority first).
var tokenEnvNames = []string{TokenEnv, TokenEnvLegacy}

// tokenPairs is the precedence list of env-var pairs that, when both halves
// are present, get composed as "<key>.<secret>" into a Netbox token. Same
// project convention as nbt_${KEY}.${TOKEN}, just split across two vars.
var tokenPairs = [][2]string{
	{TokenKeyV2, TokenSecretV2},
}

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

	// AuthScheme selects the Netbox token Authorization style.
	//   "v2" (default) → "Authorization: Bearer nbt_KEY.TOKEN"
	//   "v1"           → "Authorization: Token <token>"   (legacy)
	// Netbox docs: https://netboxlabs.com/docs/netbox/integrations/rest-api/#v1-and-v2-tokens
	AuthScheme string `mapstructure:"auth_scheme" yaml:"auth_scheme,omitempty"`

	// Columns maps a resource name (e.g. "sites", "devices") to the column
	// names to display, in order. Available column names come from the
	// internal/columns registry. When a resource isn't listed, the registry's
	// Default-flagged columns are used. Same config drives CLI and TUI.
	Columns map[string][]string `mapstructure:"columns" yaml:"columns,omitempty"`

	// ConfigFile is the resolved path of the config file that was loaded
	// (empty if none was found).
	ConfigFile string `mapstructure:"-" yaml:"-"`

	// EnvFiles is the ordered list of env files that contributed values
	// (lowest to highest priority). Empty when no file was read. Exposed
	// for diagnostics — never serialized.
	EnvFiles []string `mapstructure:"-" yaml:"-"`
}

// Defaults returns a Config with built-in defaults applied. Flag/env/file
// values overlay on top of these.
func Defaults() Config {
	return Config{
		TimeoutSeconds: 30,
		AuthScheme:     "v2",
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
//
// The token follows a separate, env-only chain (a stray `cat config.yaml`
// must never leak credentials). Highest priority first:
//
//  1. process env (NBCLI_TOKEN / NETBOX_TOKEN / NETBOX_API_V2_KEY+_TOKEN)
//  2. --env-file <path> (extraEnvFile arg)
//  3. $XDG_CONFIG_HOME/nbcli/secrets.env
//  4. ~/.env.netbox
//
// At any layer, a fully-formed token wins over the KEY+SECRET pair; the pair
// composes as "<KEY>.<SECRET>" matching nbt_${KEY}.${TOKEN}.
func Load(flags *pflag.FlagSet, configFile, extraEnvFile string) (Config, error) {
	v := viper.New()
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	for k, val := range defaultsMap() {
		v.SetDefault(k, val)
	}

	if flags != nil {
		// Presentation modifiers (format / columns) live on each show + search
		// command's positional grammar, not as flags, so we don't need a viper
		// alias for them. Every other persistent flag binds normally.
		var bindErr error
		flags.VisitAll(func(f *pflag.Flag) {
			if bindErr != nil {
				return
			}
			bindErr = v.BindPFlag(f.Name, f)
		})
		if bindErr != nil {
			return Config{}, fmt.Errorf("config: bind flags: %w", bindErr)
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

	effective, loaded := loadEffectiveEnv(extraEnvFile)
	cfg.Token = composeToken(effective)
	cfg.EnvFiles = loaded

	return cfg, nil
}

// loadEffectiveEnv merges env-file values with the real process environment.
// File order (lowest priority first; later overrides):
//
//  1. ~/.env.netbox        — community convention nbcli honors for free
//  2. $XDG_CONFIG_HOME/nbcli/secrets.env  (or ~/.config/nbcli/secrets.env)
//  3. extraEnvFile         — --env-file flag, when set
//  4. process env (os.Environ)            — always wins
//
// loadedFiles tracks which files actually contributed, in load order, for
// diagnostics. Missing files and parse failures are silently skipped — the
// caller still gets a usable effective env from the layers that did succeed.
func loadEffectiveEnv(extraEnvFile string) (env map[string]string, loadedFiles []string) {
	env = map[string]string{}
	var paths []string
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".env.netbox"),
			filepath.Join(configDirFor(home), "secrets.env"),
		)
	}
	if extraEnvFile != "" {
		paths = append(paths, extraEnvFile)
	}
	for _, p := range paths {
		vals, err := LoadEnvFile(p)
		if err != nil || vals == nil {
			continue
		}
		for k, v := range vals {
			env[k] = v
		}
		loadedFiles = append(loadedFiles, p)
	}
	// Real env overrides everything from files — but only for non-empty
	// values. Empty values are treated as "not set" so a file value can
	// fill them (matches the conventional shell semantics where an empty
	// var is functionally the same as unset).
	for _, e := range os.Environ() {
		i := strings.IndexByte(e, '=')
		if i <= 0 {
			continue
		}
		if v := e[i+1:]; v != "" {
			env[e[:i]] = v
		}
	}
	return env, loadedFiles
}

// composeToken returns the Netbox API token from the effective env map.
// Already-formed tokens win over the KEY+SECRET composition.
func composeToken(env map[string]string) string {
	for _, n := range tokenEnvNames {
		if t := env[n]; t != "" {
			return t
		}
	}
	for _, p := range tokenPairs {
		k, s := env[p[0]], env[p[1]]
		if k != "" && s != "" {
			return k + "." + s
		}
	}
	return ""
}

// configDirFor returns the nbcli config directory for a given home, honoring
// $XDG_CONFIG_HOME when set.
func configDirFor(home string) string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "nbcli")
	}
	return filepath.Join(home, ".config", "nbcli")
}

// Validate returns an error if the resolved config is unusable for live calls.
// Token is checked separately because read-only/local commands shouldn't need it.
func (c Config) Validate() error {
	if c.URL == "" {
		return errNoURL
	}
	return nil
}

// Save writes c to config.yaml at c.ConfigFile (or the default
// $XDG_CONFIG_HOME/nbcli/config.yaml when ConfigFile is empty). The write is
// atomic (tmpfile + rename) and creates the parent directory at mode 0700.
// Token / EnvFiles / ConfigFile fields carry yaml:"-" so they never land on
// disk — the secret-isolation invariant holds even through Save.
func (c Config) Save() error {
	path := c.ConfigFile
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("config: resolve home: %w", err)
		}
		path = filepath.Join(configDirFor(home), "config.yaml")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("config: write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("config: rename %s: %w", path, err)
	}
	return nil
}

// RequireToken returns an error if no token was found in env or env-files.
func (c Config) RequireToken() error {
	if c.Token == "" {
		return fmt.Errorf("config: no token found (set %s, %s, or %s + %s)",
			TokenEnv, TokenEnvLegacy, TokenKeyV2, TokenSecretV2)
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
		"auth_scheme":          d.AuthScheme,
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
