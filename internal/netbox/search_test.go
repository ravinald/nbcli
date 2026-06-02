package netbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch_PassesQAndPagination(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/search/", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, "hq", q.Get("q"))
		assert.Equal(t, "25", q.Get("limit"))
		assert.Equal(t, "50", q.Get("offset"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[SearchResult]{
			Count: 3,
			Results: []SearchResult{
				{Type: "dcim.site", Field: "name", Value: "hq", Object: json.RawMessage(`{"id":1,"display":"HQ","url":"/api/dcim/sites/1/"}`)},
				{Type: "dcim.device", Field: "comments", Value: "hq backup", Object: json.RawMessage(`{"id":7,"display":"edge-1","url":"/api/dcim/devices/7/"}`)},
				{Type: "ipam.ipaddress", Field: "description", Value: "hq mgmt", Object: json.RawMessage(`{"id":42,"display":"10.0.0.1/24","url":"/api/ipam/ip-addresses/42/"}`)},
			},
		})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	page, err := c.Search(context.Background(), SearchOptions{Q: "hq", Limit: 25, Offset: 50})
	require.NoError(t, err)
	assert.Equal(t, 3, page.Count)
	require.Len(t, page.Results, 3)
	assert.Equal(t, "dcim.site", page.Results[0].Type)
	assert.Equal(t, "name", page.Results[0].Field)
	assert.Equal(t, "hq", page.Results[0].Value)
}

func TestSearch_ExtraOverlaysButQWins(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// q from Q (set after Extra) overrides Extra's q.
		assert.Equal(t, "live", q.Get("q"))
		// Other Extra keys flow through.
		assert.Equal(t, "dcim.site", q.Get("object_types"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[SearchResult]{Count: 0})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	_, err = c.Search(context.Background(), SearchOptions{
		Q:     "live",
		Extra: url.Values{"q": {"stale"}, "object_types": {"dcim.site"}},
	})
	require.NoError(t, err)
}

func TestSearchFetcher_BindsOpts(t *testing.T) {
	t.Parallel()
	var seenOffset, seenLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenOffset = r.URL.Query().Get("offset")
		seenLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[SearchResult]{Count: 0})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	f := c.SearchFetcher(SearchOptions{Q: "x"})
	_, err = f(context.Background(), 100, 25)
	require.NoError(t, err)
	assert.Equal(t, "100", seenOffset)
	assert.Equal(t, "25", seenLimit)
}
