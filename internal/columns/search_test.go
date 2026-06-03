package columns

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ravinald/nbcli/internal/netbox"
)

// TestSearchSet_DefaultsIncludeAttributes confirms the column is on by
// default — it's the whole point of the per-type Attributes extractor.
func TestSearchSet_DefaultsIncludeAttributes(t *testing.T) {
	t.Parallel()
	got := SearchSet().VisibleNames(nil)
	assert.Contains(t, got, "attributes")
}

func TestAttributes_Contact(t *testing.T) {
	t.Parallel()
	hit := netbox.SearchResult{
		Type: "tenancy.contact",
		Object: json.RawMessage(`{
            "id": 7, "display": "Ravi Pina", "name": "Ravi Pina",
            "title": "SRE", "email": "ravi@cow.org", "phone": "+1-555"
        }`),
	}
	got := attributesCol(hit)
	assert.Contains(t, got, "title=SRE")
	assert.Contains(t, got, "email=ravi@cow.org")
	assert.Contains(t, got, "phone=+1-555")
}

func TestAttributes_VirtualMachine(t *testing.T) {
	t.Parallel()
	hit := netbox.SearchResult{
		Type: "virtualization.virtualmachine",
		Object: json.RawMessage(`{
            "id": 1, "display": "i-deadbeef", "name": "i-deadbeef",
            "site": {"id": 1, "name": "hq"},
            "cluster": {"id": 2, "name": "prod-1"},
            "tenant": {"id": 3, "name": "Acme"},
            "status": {"value": "active", "label": "Active"}
        }`),
	}
	got := attributesCol(hit)
	for _, want := range []string{"site=hq", "cluster=prod-1", "tenant=Acme", "status=Active"} {
		assert.Containsf(t, got, want, "missing %q in attributes: %s", want, got)
	}
}

func TestAttributes_IPAddress(t *testing.T) {
	t.Parallel()
	hit := netbox.SearchResult{
		Type: "ipam.ipaddress",
		Object: json.RawMessage(`{
            "id": 42, "display": "10.0.0.1/24", "address": "10.0.0.1/24",
            "vrf": {"id": 1, "name": "mgmt"},
            "tenant": {"id": 2, "name": "Acme"},
            "status": {"value": "active", "label": "Active"},
            "dns_name": "edge1.hq"
        }`),
	}
	got := attributesCol(hit)
	for _, want := range []string{"vrf=mgmt", "tenant=Acme", "status=Active", "dns_name=edge1.hq"} {
		assert.Containsf(t, got, want, "missing %q in attributes: %s", want, got)
	}
}

func TestAttributes_OmitsEmptyFields(t *testing.T) {
	t.Parallel()
	// Contact with only the name set — title/email/phone empty, so they
	// should be dropped from the output rather than rendered as "title=".
	hit := netbox.SearchResult{
		Type:   "tenancy.contact",
		Object: json.RawMessage(`{"id": 1, "display": "Anon", "name": "Anon"}`),
	}
	got := attributesCol(hit)
	for _, banned := range []string{"title=", "email=", "phone="} {
		assert.NotContainsf(t, got, banned, "empty field %q should not render", banned)
	}
}

func TestAttributes_UnknownTypeEmpty(t *testing.T) {
	t.Parallel()
	hit := netbox.SearchResult{
		Type:   "wireless.wirelessLAN", // not in the extractor map
		Object: json.RawMessage(`{"id": 1, "display": "x"}`),
	}
	got := attributesCol(hit)
	assert.Empty(t, got, "unknown type should produce no attributes rather than guess")
}

func TestAttributes_AllRegisteredTypesAreInSearchTypes(t *testing.T) {
	t.Parallel()
	// Drift guard: every type the netbox package fans out across should have
	// an attribute extractor. Catches the case where someone adds a resource
	// to SearchTypes but forgets to register the extractor.
	for _, st := range netbox.SearchTypes {
		_, ok := searchAttrExtractors[st.Dotted]
		assert.Truef(t, ok, "no attribute extractor for %s — add one in columns/search.go", st.Dotted)
	}
}

// Sanity: the attributes string is at least syntactically reasonable —
// pairs separated by ", " with no trailing comma. Guards against the
// joinAttrs helper regressing.
func TestAttributes_FormatShape(t *testing.T) {
	t.Parallel()
	hit := netbox.SearchResult{
		Type: "tenancy.contact",
		Object: json.RawMessage(`{
            "id": 1, "title": "A", "email": "b@c", "phone": "1"
        }`),
	}
	got := attributesCol(hit)
	assert.False(t, strings.HasSuffix(got, ", "), "trailing comma in: %s", got)
	assert.False(t, strings.HasPrefix(got, ", "), "leading comma in: %s", got)
}
