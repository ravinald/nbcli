package columns

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
