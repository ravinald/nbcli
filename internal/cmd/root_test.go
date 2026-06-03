package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// newJSONServer returns an httptest server that responds with the given JSON
// body on path. Any other path 404s. Used by tests that drive `nbcli show ...`
// end-to-end without needing a full mux per resource.
func newJSONServer(t *testing.T, path, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
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
	// Root help advertises subcommands + the persistent session flags. Per-call
	// presentation modifiers (format, columns, pager) are positional on each
	// show/search command, so they show up under those commands' help — not
	// here.
	for _, fragment := range []string{"show", "tui", "plugin", "version", "--url", "--verbose"} {
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

// --- Positional presentation overrides (format / columns) -----------------

func TestExecute_FormatPositionalOverridesConfig(t *testing.T) {
	// no t.Parallel() — isolateEnv uses t.Setenv which forbids parallel.
	isolateEnv(t)
	srv := newJSONServer(t, "/api/dcim/sites/", `{"count":1,"results":[{"id":1,"name":"hq","slug":"hq","status":{"value":"active","label":"Active"}}]}`)
	defer srv.Close()
	t.Setenv("NBCLI_URL", srv.URL)
	t.Setenv("NBCLI_TOKEN", "nbt_a.b")
	t.Setenv("NBCLI_FORMAT", "table") // env says table

	io, out, errb := makeIO()
	code := cmd.Execute([]string{"show", "sites", "format", "json"}, io) // positional says json
	require.Equalf(t, 0, code, "stderr=%s", errb.String())
	body := out.String()
	assert.Truef(t, strings.HasPrefix(strings.TrimSpace(body), "["),
		"positional `format json` should yield JSON output, got: %q", body)
}

func TestExecute_ColumnsPositionalRestrictsHeaders(t *testing.T) {
	// no t.Parallel() — isolateEnv uses t.Setenv which forbids parallel.
	isolateEnv(t)
	srv := newJSONServer(t, "/api/dcim/sites/", `{"count":1,"results":[{"id":1,"name":"hq","slug":"hq","status":{"value":"active","label":"Active"}}]}`)
	defer srv.Close()
	t.Setenv("NBCLI_URL", srv.URL)
	t.Setenv("NBCLI_TOKEN", "nbt_a.b")
	t.Setenv("NBCLI_FORMAT", "table")

	io, out, errb := makeIO()
	code := cmd.Execute([]string{"show", "sites", "columns", "id,name"}, io)
	require.Equalf(t, 0, code, "stderr=%s", errb.String())
	body := out.String()
	assert.Contains(t, body, "ID")
	assert.Contains(t, body, "NAME")
	// Status was a default column; excluding via `columns id,name` should drop it.
	assert.NotContains(t, body, "STATUS",
		"explicit `columns id,name` should suppress default columns")
}

func TestExecute_FormatPositionalRejectsBadValue(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"show", "sites", "format", "xml"}, io)
	require.NotEqual(t, 0, code)
	// Error comes from output.Parse since "xml" isn't a known format.
	// (Validator allowed it through — Values is a completion hint, not a
	// hard whitelist.)
	assert.Contains(t, errb.String(), "format")
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

func TestExecute_ShowSites_ValueKeywordMissingValue(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"show", "sites", "status"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), `keyword "status" expects a value`)
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
