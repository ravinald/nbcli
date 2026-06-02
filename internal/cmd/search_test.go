package cmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ravinald/nbcli/internal/cmd"
	"github.com/ravinald/nbcli/internal/netbox"
)

// --- Validator ------------------------------------------------------------
//
// The validator runs before any network IO and before any client setup, so
// these cases don't need a fake Netbox server — exercising cmd.Execute is
// enough.

func TestSearch_Validator_NoArgs(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"search"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), "expected: search [all|<module>] <key>")
}

func TestSearch_Validator_UnknownModule(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"search", "foo", "bar"}, io)
	require.NotEqual(t, 0, code)
	got := errb.String()
	assert.Contains(t, got, `unknown module "foo"`)
	// Discoverable: lists the menu.
	assert.Contains(t, got, "all")
	assert.Contains(t, got, "ip-addresses")
}

func TestSearch_Validator_MissingKey(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"search", "sites"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), "requires a query key")
}

func TestSearch_Validator_UnknownTrailingKeyword(t *testing.T) {
	t.Parallel()
	io, _, errb := makeIO()
	code := cmd.Execute([]string{"search", "sites", "hq", "nope"}, io)
	require.NotEqual(t, 0, code)
	assert.Contains(t, errb.String(), `unknown keyword "nope"`)
}

// --- End-to-end happy paths -----------------------------------------------
//
// Each per-module handler ends up hitting a specific Netbox endpoint with
// ?q=<key>. The fake server below verifies path + query then returns a
// minimal response the renderer can format.

func TestSearch_AllFansOutAcrossTypedEndpoints(t *testing.T) {
	isolateEnv(t)
	var seenQ atomic.Value
	var hits atomic.Int32

	mux := http.NewServeMux()
	for _, ep := range netbox.SearchEndpoints {
		mux.HandleFunc(ep.Path, func(w http.ResponseWriter, r *http.Request) {
			seenQ.Store(r.URL.Query().Get("q"))
			hits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			obj, _ := json.Marshal(map[string]any{
				"id": 1, "display": ep.Type + "-hit", "url": ep.Path + "1/",
			})
			_ = json.NewEncoder(w).Encode(netbox.Page[json.RawMessage]{
				Count: 1, Results: []json.RawMessage{obj},
			})
		})
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("NBCLI_URL", srv.URL)
	t.Setenv("NBCLI_TOKEN", "nbt_a.b")
	t.Setenv("NBCLI_FORMAT", "json")

	io, out, errb := makeIO()
	code := cmd.Execute([]string{"search", "all", "hq"}, io)
	require.Equalf(t, 0, code, "stderr=%s", errb.String())
	assert.Equal(t, "hq", seenQ.Load(), "?q= propagates to every endpoint")
	assert.Equal(t, len(netbox.SearchEndpoints), int(hits.Load()),
		"every endpoint in SearchEndpoints should receive a request")
	// Aggregate contains rows from multiple types.
	body := out.String()
	assert.Contains(t, body, "dcim.site")
	assert.Contains(t, body, "ipam.ipaddress")
}

func TestSearch_SitesHitsSitesEndpointWithQ(t *testing.T) {
	isolateEnv(t)
	var seenPath, seenQ atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath.Store(r.URL.Path)
		seenQ.Store(r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(netbox.Page[netbox.Site]{
			Count:   1,
			Results: []netbox.Site{{ID: 1, Name: "HQ", Slug: "hq", Status: netbox.LabelValue{Value: "active", Label: "Active"}}},
		})
	}))
	defer srv.Close()
	t.Setenv("NBCLI_URL", srv.URL)
	t.Setenv("NBCLI_TOKEN", "nbt_a.b")
	t.Setenv("NBCLI_FORMAT", "json")

	io, _, errb := makeIO()
	code := cmd.Execute([]string{"search", "sites", "hq"}, io)
	require.Equalf(t, 0, code, "stderr=%s", errb.String())
	assert.Equal(t, "/api/dcim/sites/", seenPath.Load())
	assert.Equal(t, "hq", seenQ.Load())
}

func TestSearch_IPAddressesHitsIPAMEndpoint(t *testing.T) {
	isolateEnv(t)
	var seenPath, seenQ atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath.Store(r.URL.Path)
		seenQ.Store(r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(netbox.Page[netbox.IPAddress]{Count: 0})
	}))
	defer srv.Close()
	t.Setenv("NBCLI_URL", srv.URL)
	t.Setenv("NBCLI_TOKEN", "nbt_a.b")
	t.Setenv("NBCLI_FORMAT", "json")

	io, _, errb := makeIO()
	code := cmd.Execute([]string{"search", "ip-addresses", "10.0.0"}, io)
	require.Equalf(t, 0, code, "stderr=%s", errb.String())
	assert.Equal(t, "/api/ipam/ip-addresses/", seenPath.Load())
	assert.Equal(t, "10.0.0", seenQ.Load())
}

func TestSearch_LimitFlowsThrough(t *testing.T) {
	isolateEnv(t)
	var seenLimit atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenLimit.Store(r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(netbox.Page[netbox.Site]{Count: 0})
	}))
	defer srv.Close()
	t.Setenv("NBCLI_URL", srv.URL)
	t.Setenv("NBCLI_TOKEN", "nbt_a.b")
	t.Setenv("NBCLI_FORMAT", "json")

	io, _, errb := makeIO()
	code := cmd.Execute([]string{"search", "sites", "hq", "limit", "200"}, io)
	require.Equalf(t, 0, code, "stderr=%s", errb.String())
	assert.Equal(t, "200", seenLimit.Load())
}

// --- Completion ----------------------------------------------------------
//
// __complete is cobra's hidden command that drives ValidArgsFunction. Each
// call exercises one position in `search <module> <key> <trailing...>`.

func runComplete(t *testing.T, args ...string) string {
	t.Helper()
	io, out, _ := makeIO()
	full := append([]string{"__complete", "search"}, args...)
	code := cmd.Execute(full, io)
	require.Equal(t, 0, code)
	return out.String()
}

func TestSearch_Completion_FirstArgListsModules(t *testing.T) {
	t.Parallel()
	got := runComplete(t, "")
	for _, want := range []string{"all", "sites", "ip-addresses", "virtual-machines"} {
		assert.Containsf(t, got, want+"\n", "missing module %q in completion: %s", want, got)
	}
}

func TestSearch_Completion_KeyPositionIsFreeForm(t *testing.T) {
	t.Parallel()
	got := runComplete(t, "sites", "")
	// No keyword suggestions for the free-form key; only the trailing
	// :directive line should appear.
	for _, leak := range []string{"all", "sites", "limit", "pager"} {
		assert.NotContainsf(t, got, leak+"\n", "key position leaked keyword %q: %s", leak, got)
	}
}

func TestSearch_Completion_TrailingOffersLimitAndPager(t *testing.T) {
	t.Parallel()
	got := runComplete(t, "sites", "hq", "")
	for _, want := range []string{"limit", "pager"} {
		assert.Containsf(t, got, want+"\n", "missing trailing keyword %q in: %s", want, got)
	}
}

func TestSearch_Completion_PagerSwitchAdvances(t *testing.T) {
	t.Parallel()
	// After typing the switch, the next position should still offer the
	// other unused keyword (limit) but not pager itself.
	got := runComplete(t, "sites", "hq", "pager", "")
	assert.Contains(t, got, "limit\n")
	assert.NotContains(t, got, "pager\n")
	_ = strings.TrimSpace(got)
}
