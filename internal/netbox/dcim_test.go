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

func TestListRacks_PassesFilters(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/dcim/racks/", r.URL.Path)
		assert.Equal(t, "hq", r.URL.Query().Get("site"))
		assert.Equal(t, "active", r.URL.Query().Get("status"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[Rack]{
			Count: 1,
			Results: []Rack{{
				ID: 5, Name: "R1",
				Site:    &NestedRef{Name: "HQ"},
				Status:  LabelValue{Value: "active", Label: "Active"},
				UHeight: 42,
			}},
		})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	page, err := c.ListRacks(context.Background(), ListRacksOptions{Site: "hq", Status: "active"})
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	assert.Equal(t, "R1", page.Results[0].Name)
	assert.Equal(t, 42, page.Results[0].UHeight)
}

func TestListDevices_PassesAllCommonFilters(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "hq", q.Get("site"))
		assert.Equal(t, "switch", q.Get("role"))
		assert.Equal(t, "active", q.Get("status"))
		assert.Equal(t, "juniper", q.Get("manufacturer"))
		assert.Equal(t, "ex4300", q.Get("device_type"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[Device]{
			Count: 1,
			Results: []Device{{
				ID: 1, Name: "hq-sw-01",
				DeviceType: &DeviceType{Model: "EX4300-48T",
					Manufacturer: &NestedRef{Name: "Juniper"}},
				Site:   &NestedRef{Name: "HQ"},
				Status: LabelValue{Value: "active", Label: "Active"},
			}},
		})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	page, err := c.ListDevices(context.Background(), ListDevicesOptions{
		Site: "hq", Role: "switch", Status: "active",
		Manufacturer: "juniper", Model: "ex4300",
	})
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	assert.Equal(t, "hq-sw-01", page.Results[0].Name)
	assert.Equal(t, "EX4300-48T", page.Results[0].DeviceType.Model)
}

func TestListInterfaces_BoolFiltersEncoded(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "hq-sw-01", q.Get("device"))
		assert.Equal(t, "true", q.Get("enabled"))
		assert.Equal(t, "false", q.Get("mgmt_only"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[Interface]{Count: 0})
	}))
	defer srv.Close()

	c, err := New(Options{BaseURL: srv.URL, Token: "t"})
	require.NoError(t, err)
	enabled, mgmtOnly := true, false
	_, err = c.ListInterfaces(context.Background(), ListInterfacesOptions{
		Device:   "hq-sw-01",
		Enabled:  &enabled,
		MgmtOnly: &mgmtOnly,
	})
	require.NoError(t, err)
}

func TestRacksFetcher_OverridesPagination(t *testing.T) {
	t.Parallel()
	var off, lim string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		off, lim = r.URL.Query().Get("offset"), r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Page[Rack]{})
	}))
	defer srv.Close()
	c, _ := New(Options{BaseURL: srv.URL, Token: "t"})
	_, err := c.RacksFetcher(ListRacksOptions{Offset: 999, Limit: 999})(context.Background(), 25, 10)
	require.NoError(t, err)
	assert.Equal(t, "25", off)
	assert.Equal(t, "10", lim)
}
