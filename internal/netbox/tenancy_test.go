package netbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTenants_HappyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tenancy/tenants/", r.URL.Path)
		assert.Equal(t, "engineering", r.URL.Query().Get("group"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[Tenant]{
			Count: 1,
			Results: []Tenant{{
				ID: 7, Name: "Acme", Slug: "acme",
				Group: &NestedRef{ID: 1, Name: "Engineering"},
			}},
		})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	page, err := c.ListTenants(context.Background(), ListTenantsOptions{Group: "engineering"})
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	assert.Equal(t, "Acme", page.Results[0].Name)
	assert.Equal(t, "Engineering", page.Results[0].Group.Name)
}

func TestListContacts_HappyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tenancy/contacts/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[Contact]{
			Count: 2,
			Results: []Contact{
				{ID: 1, Name: "Alice", Email: "a@example.com"},
				{ID: 2, Name: "Bob", Email: "b@example.com"},
			},
		})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	page, err := c.ListContacts(context.Background(), ListContactsOptions{})
	require.NoError(t, err)
	assert.Len(t, page.Results, 2)
}

func TestTenantsFetcher_OverridesOffsetAndLimit(t *testing.T) {
	t.Parallel()
	var seenOffset, seenLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenOffset = r.URL.Query().Get("offset")
		seenLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[Tenant]{Count: 0})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)

	// Caller-set Offset/Limit get clobbered by the iterator-supplied values.
	fetch := c.TenantsFetcher(ListTenantsOptions{Offset: 999, Limit: 999})
	_, err = fetch(context.Background(), 25, 10)
	require.NoError(t, err)
	assert.Equal(t, "25", seenOffset)
	assert.Equal(t, "10", seenLimit)
}
