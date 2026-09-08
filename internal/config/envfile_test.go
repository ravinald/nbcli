package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEnvFile_BasicLines(t *testing.T) {
	t.Parallel()
	in := `
# comment line
FOO=bar
BAZ=hello world
EMPTY=
QUOTED="quoted value"
SINGLE='single value'
export EXPORTED=ok
URL_WITH_HASH=https://example.com/path#frag
TRAILING_COMMENT=foo  # trailing
TAB_COMMENT=bar` + "\t" + `# tabbed
`
	out, err := ParseEnvFile(strings.NewReader(in))
	require.NoError(t, err)
	assert.Equal(t, "bar", out["FOO"])
	assert.Equal(t, "hello world", out["BAZ"])
	assert.Equal(t, "", out["EMPTY"])
	assert.Equal(t, "quoted value", out["QUOTED"])
	assert.Equal(t, "single value", out["SINGLE"])
	assert.Equal(t, "ok", out["EXPORTED"])
	assert.Equal(t, "https://example.com/path#frag", out["URL_WITH_HASH"],
		"# inside an unquoted value without leading space stays")
	assert.Equal(t, "foo", out["TRAILING_COMMENT"])
	assert.Equal(t, "bar", out["TAB_COMMENT"])
}

func TestParseEnvFile_QuotesPreserveSpaces(t *testing.T) {
	t.Parallel()
	out, err := ParseEnvFile(strings.NewReader(`SPACED="  padded  "`))
	require.NoError(t, err)
	assert.Equal(t, "  padded  ", out["SPACED"])
}

func TestParseEnvFile_BadLines(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
	}{
		{"no equals", "noequals\n"},
		{"empty key", "=value\n"},
		{"starts with digit", "1FOO=x\n"},
		{"space in key", "FO O=x\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseEnvFile(strings.NewReader(tc.in))
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrEnvFileFormat))
		})
	}
}

func TestLoadEnvFile_MissingIsNotError(t *testing.T) {
	t.Parallel()
	out, err := LoadEnvFile("/no/such/file/anywhere")
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestLoadEnvFile_RealFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "envfile")
	require.NoError(t, os.WriteFile(path, []byte("NETBOX_API_V2_KEY=abc\nNETBOX_API_V2_TOKEN=def\n"), 0o600))

	out, err := LoadEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, "abc", out["NETBOX_API_V2_KEY"])
	assert.Equal(t, "def", out["NETBOX_API_V2_TOKEN"])
}
