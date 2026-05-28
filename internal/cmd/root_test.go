package cmd_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ravinald/nbcli/internal/cmd"
)

// makeIO wires Stdin/Stdout/Stderr to in-memory buffers so tests can assert
// against output. Returns the IO struct plus pointers to the two buffers.
func makeIO() (cmd.IO, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return cmd.IO{In: strings.NewReader(""), Out: &out, Err: &errb}, &out, &errb
}

func TestRootCmd_HasExpectedChildren(t *testing.T) {
	t.Parallel()
	root := cmd.NewRootCmd(cmd.StdIO())
	have := map[string]bool{}
	for _, c := range root.Commands() {
		have[c.Name()] = true
	}
	for _, want := range []string{"show", "tui", "plugin", "version"} {
		assert.Truef(t, have[want], "expected %q subcommand", want)
	}
}

func TestExecute_VersionJSON(t *testing.T) {
	t.Parallel()
	io, out, _ := makeIO()
	code := cmd.Execute([]string{"version", "--json"}, io)
	require.Equal(t, 0, code)
	var info map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &info))
	assert.NotEmpty(t, info["version"])
	assert.NotEmpty(t, info["os"])
	assert.NotEmpty(t, info["arch"])
}

func TestExecute_Help(t *testing.T) {
	t.Parallel()
	io, out, _ := makeIO()
	code := cmd.Execute([]string{"--help"}, io)
	require.Equal(t, 0, code)
	s := out.String()
	for _, fragment := range []string{"show", "tui", "plugin", "version", "--format"} {
		assert.Containsf(t, s, fragment, "help should mention %q", fragment)
	}
}

// isolateEnv clears all credential and config-related env vars and reroutes
// HOME/XDG_CONFIG_HOME so the tests don't pick up the developer's real
// ~/.env.netbox, secrets.env, or config.yaml.
func isolateEnv(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home+"/xdg")
	for _, k := range []string{
		"NBCLI_URL", "NBCLI_FORMAT", "NBCLI_TOKEN",
		"NETBOX_TOKEN", "NETBOX_API_V2_KEY", "NETBOX_API_V2_TOKEN",
	} {
		t.Setenv(k, "")
	}
}

func TestExecute_ShowSites_RequiresURL(t *testing.T) {
	isolateEnv(t)
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"show", "sites"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), "url is required")
}

func TestExecute_ShowSites_RequiresToken(t *testing.T) {
	isolateEnv(t)
	t.Setenv("NBCLI_URL", "https://netbox.example.com")
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"show", "sites"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), "no token found")
}

func TestExecute_ShowTenants_BadKeyword(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"show", "tenants", "naem", "x"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), `unknown keyword "naem"`)
}

func TestExecute_ShowContacts_DuplicateKeyword(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"show", "contacts", "name", "a", "name", "b"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), `duplicate keyword "name"`)
}

func TestExecute_ShowSites_OddArgsFail(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"show", "sites", "status"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), "keyword/value pairs")
}

func TestExecute_Passthrough_RequiresPluginAndSubpath(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"plugin", "passthrough"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), "<plugin> <subpath>")
}

func TestExecute_Passthrough_BadPairsCount(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"plugin", "passthrough", "name", "path", "key1"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), "key/value pairs")
}

func TestExecute_PluginList_EmptyOK(t *testing.T) {
	t.Parallel()
	io, out, _ := makeIO()
	code := cmd.Execute([]string{"plugin", "list"}, io)
	require.Equal(t, 0, code)
	assert.Contains(t, out.String(), "no named plugins")
}
