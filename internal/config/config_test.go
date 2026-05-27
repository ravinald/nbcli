package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("NBCLI_TOKEN", "")
	t.Setenv("NETBOX_TOKEN", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil, "")
	require.NoError(t, err)
	assert.Equal(t, 30, cfg.TimeoutSeconds)
	assert.False(t, cfg.InsecureSkipVerify)
	assert.Empty(t, cfg.URL)
	assert.Empty(t, cfg.Token)
}

func TestLoad_EnvOverridesDefault(t *testing.T) {
	t.Setenv("NBCLI_URL", "https://nb.example.com")
	t.Setenv("NBCLI_FORMAT", "json")
	t.Setenv("NBCLI_TOKEN", "nbt_K.T")
	cfg, err := Load(nil, "")
	require.NoError(t, err)
	assert.Equal(t, "https://nb.example.com", cfg.URL)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, "nbt_K.T", cfg.Token)
}

func TestLoad_FileLowerThanEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`url: https://from-file.example.com
format: yaml
`), 0o600))

	t.Setenv("NBCLI_URL", "https://from-env.example.com")
	cfg, err := Load(nil, cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "https://from-env.example.com", cfg.URL, "env beats file")
	assert.Equal(t, "yaml", cfg.Format, "file value used when env absent")
}

func TestValidate_RequiresURL(t *testing.T) {
	require.Error(t, Config{}.Validate())
	require.NoError(t, Config{URL: "https://x"}.Validate())
}

func TestRequireToken(t *testing.T) {
	require.Error(t, Config{}.RequireToken())
	require.NoError(t, Config{Token: "t"}.RequireToken())
}
