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
		assert.Equal(t, "Bearer nbt_KEY.TOKEN", r.Header.Get("Authorization"),
			"default scheme is v2 → Bearer")
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

func TestAuthHeader_V1AndV2(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		scheme AuthScheme
		want   string
	}{
		{"default is v2 bearer", "", "Bearer nbt_K.T"},
		{"explicit v2", AuthSchemeV2, "Bearer nbt_K.T"},
		{"explicit v1", AuthSchemeV1, "Token nbt_K.T"},
		{"mixed case ok", "V1", "Token nbt_K.T"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			c, err := New(Options{BaseURL: srv.URL, Token: "nbt_K.T", AuthScheme: tc.scheme})
			require.NoError(t, err)
			require.NoError(t, c.Do(context.Background(), "GET", "/api/", nil, nil, nil))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAuthHeader_V2AutoPrependsNbtPrefix(t *testing.T) {
	t.Parallel()
	// Bare KEY.TOKEN (no nbt_ prefix) → client adds it for v2.
	c, err := New(Options{BaseURL: "https://x", Token: "K.T", AuthScheme: AuthSchemeV2})
	require.NoError(t, err)
	assert.Equal(t, "Bearer nbt_K.T", c.authHeader)

	// Already prefixed → left alone (no double prefix).
	c, err = New(Options{BaseURL: "https://x", Token: "nbt_K.T", AuthScheme: AuthSchemeV2})
	require.NoError(t, err)
	assert.Equal(t, "Bearer nbt_K.T", c.authHeader)

	// v1 is untouched even when the token happens to lack nbt_.
	c, err = New(Options{BaseURL: "https://x", Token: "raw40hex", AuthScheme: AuthSchemeV1})
	require.NoError(t, err)
	assert.Equal(t, "Token raw40hex", c.authHeader)
}

func TestAuthHeader_UnknownSchemeRejected(t *testing.T) {
	t.Parallel()
	_, err := New(Options{BaseURL: "https://x", Token: "t", AuthScheme: "v9"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown AuthScheme")
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
