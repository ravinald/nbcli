package netbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fanoutServer is the test fixture: a Netbox-shaped mux that returns one
// canned hit for every path in SearchEndpoints. seenQ captures the q
// parameter so tests can assert it propagates.
func fanoutServer(t *testing.T) (*httptest.Server, *atomic.Value, *atomic.Int32) {
	t.Helper()
	var seenQ atomic.Value
	var hitCount atomic.Int32

	mux := http.NewServeMux()
	for _, ep := range SearchEndpoints {
		mux.HandleFunc(ep.Path, func(w http.ResponseWriter, r *http.Request) {
			seenQ.Store(r.URL.Query().Get("q"))
			hitCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			// Return one hit per endpoint so the aggregate is deterministic.
			obj, _ := json.Marshal(map[string]any{
				"id":      1,
				"display": ep.Type + "-hit",
				"url":     ep.Path + "1/",
			})
			_ = json.NewEncoder(w).Encode(Page[json.RawMessage]{
				Count:   3,
				Results: []json.RawMessage{obj},
			})
		})
	}
	return httptest.NewServer(mux), &seenQ, &hitCount
}

func TestSearch_FanoutHitsEveryEndpoint(t *testing.T) {
	t.Parallel()
	srv, seenQ, hitCount := fanoutServer(t)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	page, err := c.Search(context.Background(), SearchOptions{Q: "ravi", Limit: 50})
	require.NoError(t, err)
	assert.Equal(t, "ravi", seenQ.Load())
	assert.Equal(t, len(SearchEndpoints), int(hitCount.Load()),
		"should fan out one request per endpoint")
	// Count is the sum across endpoints (3 per endpoint × 12).
	assert.Equal(t, 3*len(SearchEndpoints), page.Count)
	// Aggregate held one row per endpoint; with Limit=50 they all fit.
	assert.Len(t, page.Results, len(SearchEndpoints))
	// Every Type appears in the result set.
	types := make(map[string]bool)
	for _, r := range page.Results {
		types[r.Type] = true
	}
	for _, ep := range SearchEndpoints {
		assert.Truef(t, types[ep.Type], "missing %s in fan-out result", ep.Type)
	}
}

func TestSearch_OffsetLimitSlicesAggregate(t *testing.T) {
	t.Parallel()
	srv, _, _ := fanoutServer(t)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	// 12 endpoints × 1 row each = 12 in aggregate.
	page, err := c.Search(context.Background(), SearchOptions{Q: "x", Offset: 3, Limit: 5})
	require.NoError(t, err)
	assert.Len(t, page.Results, 5)
	// Past-the-end is empty, not an error.
	farPage, err := c.Search(context.Background(), SearchOptions{Q: "x", Offset: 1000, Limit: 5})
	require.NoError(t, err)
	assert.Empty(t, farPage.Results)
}

func TestSearch_PartialFailuresSilentlyDropped(t *testing.T) {
	t.Parallel()
	// One endpoint returns 500; the rest succeed. Search should still return
	// the rows from the successful endpoints without surfacing an error.
	var seenPaths sync.Map
	mux := http.NewServeMux()
	for i, ep := range SearchEndpoints {
		mux.HandleFunc(ep.Path, func(w http.ResponseWriter, _ *http.Request) {
			seenPaths.Store(ep.Path, true)
			if i == 0 {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			obj, _ := json.Marshal(map[string]any{"id": 1, "display": "x"})
			_ = json.NewEncoder(w).Encode(Page[json.RawMessage]{
				Count: 1, Results: []json.RawMessage{obj},
			})
		})
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	page, err := c.Search(context.Background(), SearchOptions{Q: "x", Limit: 50})
	require.NoError(t, err)
	assert.Len(t, page.Results, len(SearchEndpoints)-1, "lost rows only from the failing endpoint")
}

func TestSearch_AllEndpointsFailErrors(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	for _, ep := range SearchEndpoints {
		mux.HandleFunc(ep.Path, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "down", http.StatusServiceUnavailable)
		})
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	_, err = c.Search(context.Background(), SearchOptions{Q: "x", Limit: 50})
	require.Error(t, err)
	// Joined error names at least one of the failing types so the user can
	// see which subsystems are down.
	assert.True(t,
		strings.Contains(err.Error(), "dcim.site") || strings.Contains(err.Error(), "ipam.vrf"),
		"joined error should reference at least one failed endpoint, got: %s", err.Error())
}

func TestSearchFetcher_BindsOpts(t *testing.T) {
	t.Parallel()
	srv, _, _ := fanoutServer(t)
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	f := c.SearchFetcher(SearchOptions{Q: "x"})
	page, err := f(context.Background(), 2, 5)
	require.NoError(t, err)
	// Offset 2, limit 5 against a 12-row aggregate.
	assert.Len(t, page.Results, 5)
}
