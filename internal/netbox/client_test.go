package netbox

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ValidatesInputs(t *testing.T) {
	t.Parallel()
	_, err := New(Options{})
	require.Error(t, err)
	_, err = New(Options{BaseURL: "https://nb", Token: ""})
	require.Error(t, err)
	_, err = New(Options{BaseURL: "ftp://nb", Token: "t"})
	require.Error(t, err)
	_, err = New(Options{BaseURL: "https://nb", Token: "t"})
	require.NoError(t, err)
}

func TestListSites_HappyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/dcim/sites/", r.URL.Path)
		assert.Equal(t, "Token nbt_KEY.TOKEN", r.Header.Get("Authorization"))
		assert.Equal(t, "active", r.URL.Query().Get("status"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[Site]{
			Count: 1,
			Results: []Site{{
				ID: 1, Name: "hq", Slug: "hq",
				Status: LabelValue{Value: "active", Label: "Active"},
			}},
		})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "nbt_KEY.TOKEN"}) //nolint:gosec // test fixture
	require.NoError(t, err)

	page, err := c.ListSites(context.Background(), ListSitesOptions{Status: "active"})
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	assert.Equal(t, "hq", page.Results[0].Name)
}

func TestListSites_PropagatesAPIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"detail":"forbidden"}`, http.StatusForbidden)
	}))
	defer srv.Close()
	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	_, err = c.ListSites(context.Background(), ListSitesOptions{})
	require.Error(t, err)
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr), "should be *APIError, got %T", err)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

func TestDo_RespectsContextCancel(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()
	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = c.Do(ctx, "GET", "/api/dcim/sites/", nil, nil, nil)
	require.Error(t, err)
}
