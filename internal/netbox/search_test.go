package netbox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// graphqlOK is the test fixture: a server that pretends to be Netbox's
// /api/graphql/, captures the POST body, and returns one synthetic hit per
// requested list field.
func graphqlOK(t *testing.T) (*httptest.Server, *atomic.Value, *atomic.Int32) {
	t.Helper()
	var seenBody atomic.Value
	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/graphql/", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		hits.Add(1)
		body, _ := io.ReadAll(r.Body)
		seenBody.Store(string(body))

		// Build the response dynamically: every list_field in SearchTypes
		// gets one canned row. Keeps the test in sync with the schema.
		data := map[string]any{}
		for _, t := range SearchTypes {
			data[t.ListField] = []map[string]any{
				{"id": "1", "display": t.Dotted + "-hit"},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	return srv, &seenBody, &hits
}

func TestSearch_BatchedOverGraphQL(t *testing.T) {
	t.Parallel()
	srv, seenBody, hits := graphqlOK(t)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	page, err := c.Search(context.Background(), SearchOptions{Q: "ravi", Limit: 50})
	require.NoError(t, err)

	// Single round trip — not one per type.
	assert.Equal(t, int32(1), hits.Load())

	// POST body carries the GraphQL query and the variable.
	body := seenBody.Load().(string)
	assert.Contains(t, body, `"q":"ravi"`)
	assert.Contains(t, body, "device_list")
	assert.Contains(t, body, "site_list")
	assert.Contains(t, body, "ip_address_list")
	assert.Contains(t, body, "virtual_machine_list")

	// Every type appears in the aggregated result.
	types := make(map[string]bool, len(page.Results))
	for _, r := range page.Results {
		types[r.Type] = true
	}
	for _, st := range SearchTypes {
		assert.Truef(t, types[st.Dotted], "missing %s in fan-out result", st.Dotted)
	}
}

func TestSearch_SynthesizesURLFromTypeAndID(t *testing.T) {
	t.Parallel()
	srv, _, _ := graphqlOK(t)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	page, err := c.Search(context.Background(), SearchOptions{Q: "x", Limit: 50})
	require.NoError(t, err)

	// Find the site hit; check Object decodes to {id, display, url} where
	// url is the REST path Netbox would return for that resource.
	var site SearchResult
	for _, r := range page.Results {
		if r.Type == "dcim.site" {
			site = r
			break
		}
	}
	require.NotEmpty(t, site.Object, "site row should exist")
	var decoded struct {
		ID      int    `json:"id"`
		Display string `json:"display"`
		URL     string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(site.Object, &decoded))
	assert.Equal(t, 1, decoded.ID, "id parsed from GraphQL string to int")
	assert.Equal(t, "dcim.site-hit", decoded.Display)
	assert.Equal(t, "/api/dcim/sites/1/", decoded.URL, "url synthesized from REST path + id")
}

func TestSearch_OffsetLimitSlicesAggregate(t *testing.T) {
	t.Parallel()
	srv, _, _ := graphqlOK(t)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	page, err := c.Search(context.Background(), SearchOptions{Q: "x", Offset: 3, Limit: 5})
	require.NoError(t, err)
	assert.Len(t, page.Results, 5)
	// Past-the-end returns empty, not an error.
	far, err := c.Search(context.Background(), SearchOptions{Q: "x", Offset: 1000, Limit: 5})
	require.NoError(t, err)
	assert.Empty(t, far.Results)
}

func TestSearch_EmptyQueryShortCircuits(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	page, err := c.Search(context.Background(), SearchOptions{Q: ""})
	require.NoError(t, err)
	assert.Empty(t, page.Results)
	assert.Equal(t, int32(0), hits.Load(), "no HTTP traffic when Q is empty")
}

func TestSearch_PartialErrorsKeepData(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {"site_list": [{"id":"1","display":"HQ"}]},
			"errors": [{"message":"permission denied on device_list"}]
		}`))
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	page, err := c.Search(context.Background(), SearchOptions{Q: "x"})
	require.NoError(t, err, "partial errors must not fail the call when data is present")
	require.Len(t, page.Results, 1)
	assert.Equal(t, "dcim.site", page.Results[0].Type)
}

func TestSearch_AllErrorsNoDataIsError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"schema mismatch"},{"message":"auth bad"}]}`))
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	_, err = c.Search(context.Background(), SearchOptions{Q: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema mismatch")
	assert.Contains(t, err.Error(), "auth bad")
}

func TestSearch_HTTPErrorPropagates(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	_, err = c.Search(context.Background(), SearchOptions{Q: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestSearchFetcher_BindsOpts(t *testing.T) {
	t.Parallel()
	srv, _, _ := graphqlOK(t)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	f := c.SearchFetcher(SearchOptions{Q: "x"})
	page, err := f(context.Background(), 2, 5)
	require.NoError(t, err)
	assert.Len(t, page.Results, 5, "offset 2 + limit 5 against the 12-row aggregate")
}

func TestBuildSearchQuery_StaysInSyncWithSearchTypes(t *testing.T) {
	t.Parallel()
	q := buildSearchQuery()
	for _, st := range SearchTypes {
		assert.Containsf(t, q, st.ListField,
			"generated query missing field for %s", st.Dotted)
	}
	assert.True(t, strings.HasPrefix(q, "query GlobalSearch"), "expected named operation")
}

// --- REST fallback + dispatcher --------------------------------------------

// mixedServer fakes a Netbox where /api/graphql/ returns 404 (GraphQL off)
// but every REST list endpoint works. Mirrors the failure mode reported on
// netbox.scale.internal v4.5.5.
func mixedServer(t *testing.T) (*httptest.Server, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	var graphqlHits, restHits atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/api/graphql/", func(w http.ResponseWriter, _ *http.Request) {
		graphqlHits.Add(1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<!DOCTYPE html><html>404</html>"))
	})
	for _, st := range SearchTypes {
		mux.HandleFunc(st.RESTPath, func(w http.ResponseWriter, r *http.Request) {
			restHits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			obj, _ := json.Marshal(map[string]any{
				"id":      1,
				"display": st.Dotted + "-rest-hit",
				"url":     st.RESTPath + "1/",
			})
			_ = json.NewEncoder(w).Encode(Page[json.RawMessage]{
				Count: 1, Results: []json.RawMessage{obj},
			})
		})
	}
	return httptest.NewServer(mux), &graphqlHits, &restHits
}

func TestSearch_FallsBackToRESTWhenGraphQL404s(t *testing.T) {
	t.Parallel()
	srv, graphqlHits, restHits := mixedServer(t)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	page, err := c.Search(context.Background(), SearchOptions{Q: "ravi", Limit: 50})
	require.NoError(t, err)
	assert.Equal(t, int32(1), graphqlHits.Load(), "GraphQL probed exactly once")
	assert.Equal(t, len(SearchTypes), int(restHits.Load()), "REST fan-out across every type")
	// Aggregate covers every type.
	assert.Len(t, page.Results, len(SearchTypes))
}

func TestSearch_CachesRESTChoiceAfterFirst404(t *testing.T) {
	t.Parallel()
	srv, graphqlHits, _ := mixedServer(t)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err := c.Search(context.Background(), SearchOptions{Q: "x"})
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), graphqlHits.Load(),
		"after the first 404, subsequent calls skip the GraphQL probe")
}

func TestSearch_NonGraphQLErrorsPropagate(t *testing.T) {
	t.Parallel()
	// GraphQL returns 401 (not 404) — this is auth failure, NOT "GraphQL is
	// off." Must propagate, not fall back, so the user fixes their token.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad token", http.StatusUnauthorized)
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	_, err = c.Search(context.Background(), SearchOptions{Q: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.False(t, c.searchUsesREST.Load(), "401 must not flip the cached choice")
}
