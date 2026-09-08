package columns

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ravinald/nbcli/internal/netbox"
)

func TestSet_VisibleNames_Defaults(t *testing.T) {
	t.Parallel()
	s := Set{
		Resource: "x",
		Columns: []Column{
			{Name: "a", Default: true},
			{Name: "b", Default: false},
			{Name: "c", Default: true},
		},
	}
	assert.Equal(t, []string{"a", "c"}, s.VisibleNames(nil))
	assert.Equal(t, []string{"a", "c"}, s.VisibleNames([]string{}))
}

func TestSet_VisibleNames_OverrideWins(t *testing.T) {
	t.Parallel()
	s := Set{
		Resource: "x",
		Columns: []Column{
			{Name: "a", Default: true},
			{Name: "b", Default: false},
			{Name: "c", Default: true},
		},
	}
	got := s.VisibleNames([]string{"c", "b"})
	assert.Equal(t, []string{"c", "b"}, got, "override order is preserved verbatim")
}

func TestSet_VisibleColumns_SkipsUnknown(t *testing.T) {
	t.Parallel()
	s := Set{
		Resource: "x",
		Columns: []Column{
			{Name: "a", Default: true, Header: "A"},
			{Name: "b", Default: true, Header: "B"},
		},
	}
	got := s.VisibleColumns([]string{"a", "garbage", "b"})
	if assert.Len(t, got, 2) {
		assert.Equal(t, "A", got[0].Header)
		assert.Equal(t, "B", got[1].Header)
	}
}

func TestRegistry_CoversEveryResource(t *testing.T) {
	t.Parallel()
	want := []string{
		"sites", "racks", "devices", "interfaces",
		"prefixes", "ip-addresses", "vlans", "vrfs",
		"tenants", "contacts",
		"virtual-machines", "clusters",
	}
	got := Registry()
	for _, w := range want {
		set, ok := got[w]
		assert.Truef(t, ok, "missing registry entry %q", w)
		assert.NotEmptyf(t, set.Columns, "set %q has no columns", w)
		// Default-visible columns drive what the user sees out of the box —
		// at least one must be default-true or the resource is unusable.
		hasDefault := false
		for _, c := range set.Columns {
			if c.Default {
				hasDefault = true
				break
			}
		}
		assert.Truef(t, hasDefault, "set %q has no Default columns", w)
	}
}

func TestRegistry_AllColumnsHaveExtractAndUniqueNames(t *testing.T) {
	t.Parallel()
	for resource, set := range Registry() {
		seen := make(map[string]bool, len(set.Columns))
		for _, c := range set.Columns {
			assert.NotEmptyf(t, c.Name, "%s: column with empty Name", resource)
			assert.NotEmptyf(t, c.Header, "%s/%s: empty Header", resource, c.Name)
			assert.NotNilf(t, c.Extract, "%s/%s: nil Extract", resource, c.Name)
			assert.Falsef(t, seen[c.Name], "%s: duplicate column name %q", resource, c.Name)
			seen[c.Name] = true
		}
	}
}

func TestResolve_UnknownResourceReturnsNil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, Resolve("not-a-thing", nil))
}

// extractCell renders one named column of a set against a row, failing the
// test if the set has no such column.
func extractCell(t *testing.T, set Set, name string, row any) string {
	t.Helper()
	for _, c := range set.Columns {
		if c.Name == name {
			return c.Extract(row)
		}
	}
	t.Fatalf("column %q not found in set %q", name, set.Resource)
	return ""
}

// Bodies below are Netbox 4.7 API shapes: prefixes and clusters carry a
// generic scope FK instead of a site, contacts a groups array instead of a
// single group.
func TestPrefixesSet_SiteColumnHonorsScopeType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "site scope renders the site",
			body: `{"id":1,"prefix":"10.0.0.0/24","scope_type":"dcim.site","scope_id":24,
			        "scope":{"id":24,"name":"DM-Akron","slug":"dm-akron"}}`,
			want: "DM-Akron",
		},
		{
			name: "region scope is not a site",
			body: `{"id":2,"prefix":"10.0.1.0/24","scope_type":"dcim.region","scope_id":3,
			        "scope":{"id":3,"name":"North Carolina","slug":"north-carolina"}}`,
			want: "",
		},
		{
			name: "location scope is not a site",
			body: `{"id":3,"prefix":"10.0.2.0/24","scope_type":"dcim.location","scope_id":7,
			        "scope":{"id":7,"name":"Cage 4","slug":"cage-4"}}`,
			want: "",
		},
		{
			name: "site group scope is not a site",
			body: `{"id":4,"prefix":"10.0.3.0/24","scope_type":"dcim.sitegroup","scope_id":2,
			        "scope":{"id":2,"name":"Branch Offices","slug":"branch-offices"}}`,
			want: "",
		},
		{
			name: "null scope renders empty",
			body: `{"id":5,"prefix":"10.0.4.0/24","scope_type":null,"scope_id":null,"scope":null}`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var p netbox.Prefix
			require.NoError(t, json.Unmarshal([]byte(tt.body), &p))
			assert.Equal(t, tt.want, extractCell(t, PrefixesSet(), "site", p))
		})
	}
}

func TestClustersSet_SiteColumnHonorsScopeType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "site scope renders the site",
			body: `{"id":9,"name":"DM-Akron","scope_type":"dcim.site","scope_id":24,
			        "scope":{"id":24,"name":"DM-Akron","slug":"dm-akron"}}`,
			want: "DM-Akron",
		},
		{
			name: "region scope is not a site",
			body: `{"id":10,"name":"DM-Camden","scope_type":"dcim.region","scope_id":3,
			        "scope":{"id":3,"name":"North Carolina","slug":"north-carolina"}}`,
			want: "",
		},
		{
			name: "null scope renders empty",
			body: `{"id":11,"name":"DM-Nashua","scope_type":null,"scope_id":null,"scope":null}`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var c netbox.Cluster
			require.NoError(t, json.Unmarshal([]byte(tt.body), &c))
			assert.Equal(t, tt.want, extractCell(t, ClustersSet(), "site", c))
		})
	}
}

func TestContactsSet_GroupColumnJoinsGroups(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "one group",
			body: `{"id":1,"name":"Hank Miller","groups":[{"id":1,"name":"Dunder-Mifflin","slug":"dunder-mifflin"}]}`,
			want: "Dunder-Mifflin",
		},
		{
			name: "many groups join in returned order",
			body: `{"id":2,"name":"Deborah Lopez","groups":[
			        {"id":1,"name":"Dunder-Mifflin","slug":"dunder-mifflin"},
			        {"id":2,"name":"Vandelay","slug":"vandelay"}]}`,
			want: "Dunder-Mifflin, Vandelay",
		},
		{
			name: "no groups renders empty",
			body: `{"id":3,"name":"Kenneth Miller","groups":[]}`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var c netbox.Contact
			require.NoError(t, json.Unmarshal([]byte(tt.body), &c))
			assert.Equal(t, tt.want, extractCell(t, ContactsSet(), "group", c))
		})
	}
}
