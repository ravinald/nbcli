package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave_WritesAtomicallyAndRoundTrips(t *testing.T) {
	isolateEnv(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	c := Config{
		URL:                "https://nb.example.com",
		Format:             "json",
		TimeoutSeconds:     60,
		InsecureSkipVerify: true,
		AuthScheme:         "v1",
		Columns: map[string][]string{
			"sites":   {"id", "name", "status"},
			"devices": {"id", "name", "type", "site"},
		},
		ConfigFile: cfgPath,
	}
	require.NoError(t, c.Save())

	// File exists with restrictive perms.
	info, err := os.Stat(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	// Round-trip via Load: reading the just-saved file recovers the same fields.
	got, err := Load(nil, cfgPath, "")
	require.NoError(t, err)
	assert.Equal(t, "https://nb.example.com", got.URL)
	assert.Equal(t, "json", got.Format)
	assert.Equal(t, 60, got.TimeoutSeconds)
	assert.True(t, got.InsecureSkipVerify)
	assert.Equal(t, "v1", got.AuthScheme)
	assert.Equal(t, []string{"id", "name", "status"}, got.Columns["sites"])
}

func TestSave_NeverWritesTokenOrSecretFields(t *testing.T) {
	isolateEnv(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	c := Config{
		URL:        "https://nb.example.com",
		Token:      "nbt_super_secret.abcdef",
		ConfigFile: cfgPath,
		EnvFiles:   []string{"/some/path/.env.netbox"},
	}
	require.NoError(t, c.Save())

	raw, err := os.ReadFile(cfgPath) //nolint:gosec // test fixture path
	require.NoError(t, err)
	text := string(raw)
	assert.NotContains(t, text, "nbt_super_secret.abcdef", "token must never appear in saved config")
	assert.NotContains(t, text, "/some/path/.env.netbox", "EnvFiles is diagnostic-only, never persisted")
	assert.NotContains(t, strings.ToLower(text), "token", "no token field at all")
}

func TestSave_CreatesParentDirectory(t *testing.T) {
	isolateEnv(t)
	dir := filepath.Join(t.TempDir(), "nested", "config")
	cfgPath := filepath.Join(dir, "config.yaml")

	c := Config{URL: "https://nb.example.com", ConfigFile: cfgPath}
	require.NoError(t, c.Save())
	_, err := os.Stat(cfgPath)
	require.NoError(t, err, "Save should mkdir -p the parent")
}
