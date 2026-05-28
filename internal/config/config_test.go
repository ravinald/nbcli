package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateEnv resets every env var that nbcli reads, plus HOME and
// XDG_CONFIG_HOME, so test runs don't pick up the developer's real
// ~/.env.netbox or ~/.config/nbcli/secrets.env. Returns the temp dir used
// as HOME so callers can drop fixture files into it.
func isolateEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home+"/xdg")
	t.Setenv("NBCLI_TOKEN", "")
	t.Setenv("NETBOX_TOKEN", "")
	t.Setenv(TokenKeyV2, "")
	t.Setenv(TokenSecretV2, "")
	t.Setenv("NBCLI_URL", "")
	t.Setenv("NBCLI_FORMAT", "")
	return home
}

func TestLoad_Defaults(t *testing.T) {
	isolateEnv(t)
	cfg, err := Load(nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.TimeoutSeconds)
	assert.False(t, cfg.InsecureSkipVerify)
	assert.Empty(t, cfg.URL)
	assert.Empty(t, cfg.Token)
	assert.Equal(t, "v2", cfg.AuthScheme, "v2 is the default")
}

func TestLoad_AuthSchemeFromFile(t *testing.T) {
	isolateEnv(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath,
		[]byte("url: https://x\nauth_scheme: v1\n"), 0o600))
	cfg, err := Load(nil, cfgPath, "")
	require.NoError(t, err)
	assert.Equal(t, "v1", cfg.AuthScheme)
}

func TestLoad_EnvOverridesDefault(t *testing.T) {
	isolateEnv(t)
	t.Setenv("NBCLI_URL", "https://nb.example.com")
	t.Setenv("NBCLI_FORMAT", "json")
	t.Setenv("NBCLI_TOKEN", "nbt_K.T")
	cfg, err := Load(nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, "https://nb.example.com", cfg.URL)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, "nbt_K.T", cfg.Token)
}

func TestLoad_FileLowerThanEnv(t *testing.T) {
	isolateEnv(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`url: https://from-file.example.com
format: yaml
`), 0o600))

	t.Setenv("NBCLI_URL", "https://from-env.example.com")
	cfg, err := Load(nil, cfgPath, "")
	require.NoError(t, err)
	assert.Equal(t, "https://from-env.example.com", cfg.URL, "env beats file")
	assert.Equal(t, "yaml", cfg.Format, "file value used when env absent")
}

func TestLoad_ComposesTokenFromV2Pair(t *testing.T) {
	isolateEnv(t)
	t.Setenv(TokenKeyV2, "nbt_abc")
	t.Setenv(TokenSecretV2, "def")
	cfg, err := Load(nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, "nbt_abc.def", cfg.Token)
}

func TestLoad_ExistingTokenBeatsV2Pair(t *testing.T) {
	isolateEnv(t)
	t.Setenv("NBCLI_TOKEN", "explicit")
	t.Setenv(TokenKeyV2, "nbt_abc")
	t.Setenv(TokenSecretV2, "def")
	cfg, err := Load(nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, "explicit", cfg.Token)
}

func TestLoad_EnvFileSuppliesPair(t *testing.T) {
	home := isolateEnv(t)
	envPath := filepath.Join(home, ".env.netbox")
	require.NoError(t, os.WriteFile(envPath, []byte(
		"NETBOX_API_V2_KEY=nbt_kkk\nNETBOX_API_V2_TOKEN=sss\n"), 0o600))

	cfg, err := Load(nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, "nbt_kkk.sss", cfg.Token)
	require.NotEmpty(t, cfg.EnvFiles)
	assert.Equal(t, envPath, cfg.EnvFiles[0])
}

func TestLoad_RealEnvBeatsEnvFile(t *testing.T) {
	home := isolateEnv(t)
	envPath := filepath.Join(home, ".env.netbox")
	require.NoError(t, os.WriteFile(envPath, []byte("NBCLI_TOKEN=from-file\n"), 0o600))
	t.Setenv("NBCLI_TOKEN", "from-real-env")

	cfg, err := Load(nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, "from-real-env", cfg.Token)
}

func TestLoad_ExplicitEnvFileBeatsDefaults(t *testing.T) {
	home := isolateEnv(t)
	// Default-path file says one thing...
	defaultEnv := filepath.Join(home, ".env.netbox")
	require.NoError(t, os.WriteFile(defaultEnv, []byte("NBCLI_TOKEN=default-file\n"), 0o600))
	// ...explicit --env-file says another. Explicit wins.
	explicit := filepath.Join(home, "explicit.env")
	require.NoError(t, os.WriteFile(explicit, []byte("NBCLI_TOKEN=explicit-file\n"), 0o600))

	cfg, err := Load(nil, "", explicit)
	require.NoError(t, err)
	assert.Equal(t, "explicit-file", cfg.Token)
}

func TestValidate_RequiresURL(t *testing.T) {
	t.Parallel()
	require.Error(t, Config{}.Validate())
	require.NoError(t, Config{URL: "https://x"}.Validate())
}

func TestRequireToken(t *testing.T) {
	t.Parallel()
	err := Config{}.RequireToken()
	require.Error(t, err)
	assert.Contains(t, err.Error(), TokenKeyV2)
	require.NoError(t, Config{Token: "t"}.RequireToken())
}
