package columns

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ravinald/nbcli/internal/netbox"
)

// searchObjectShim is the lowest-common-denominator subset every Netbox
// object guarantees. Decoded lazily inside extractors so we don't pay for
// it unless the column is visible.
type searchObjectShim struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	Display string `json:"display"`
}

func searchObject(r any) searchObjectShim {
	hit, ok := r.(netbox.SearchResult)
	if !ok || len(hit.Object) == 0 {
		return searchObjectShim{}
	}
	var s searchObjectShim
	_ = json.Unmarshal(hit.Object, &s)
	return s
}

// SearchSet returns the column menu for global search hits.
//
// Defaults — type, field, value, display, attributes — answer "what kind of
// thing matched, why it matched, what it's called, and what context belongs
// to it." Attributes mirrors Netbox's web-UI global-search Attributes column
// (per-type metadata like Site/Cluster/Tenant for VMs, Title/Email for
// Contacts). id and url are available but hidden by default.
func SearchSet() Set {
	return Set{
		Resource: "search",
		Columns: []Column{
			col("type", "Type", 22, func(r any) string { return r.(netbox.SearchResult).Type }),
			col("field", "Field", 16, func(r any) string { return r.(netbox.SearchResult).Field }),
			col("value", "Value", 28, func(r any) string { return r.(netbox.SearchResult).Value }),
			col("display", "Display", 28, func(r any) string { return searchObject(r).Display }),
			col("attributes", "Attributes", 60, attributesCol),
			opt("id", "ID", 8, func(r any) string { return strconv.Itoa(searchObject(r).ID) }),
			opt("url", "URL", 48, func(r any) string { return searchObject(r).URL }),
		},
	}
}

// attributesCol dispatches on SearchResult.Type to the right per-type
// extractor. Empty for unknown types or undecodeable objects — we don't
// fabricate context we don't have.
func attributesCol(r any) string {
	hit, ok := r.(netbox.SearchResult)
	if !ok || len(hit.Object) == 0 {
		return ""
	}
	if fn, ok := searchAttrExtractors[hit.Type]; ok {
		return fn(hit.Object)
	}
	return ""
}

// searchAttrExtractors maps a dotted Netbox object type to a function that
// pulls the per-type "Attributes" view out of the raw row JSON. Mirrors what
// the Netbox web UI's global-search renders in its Attributes column. To
// support a new resource: add an entry plus the extractor below.
var searchAttrExtractors = map[string]func(json.RawMessage) string{
	"dcim.site":                     siteAttrs,
	"dcim.rack":                     rackAttrs,
	"dcim.device":                   deviceAttrs,
	"dcim.interface":                interfaceAttrs,
	"ipam.prefix":                   prefixAttrs,
	"ipam.ipaddress":                ipAddressAttrs,
	"ipam.vlan":                     vlanAttrs,
	"ipam.vrf":                      vrfAttrs,
	"tenancy.tenant":                tenantAttrs,
	"tenancy.contact":               contactAttrs,
	"virtualization.virtualmachine": vmAttrs,
	"virtualization.cluster":        clusterAttrs,
}

// joinAttrs renders a list of "k=v" pairs as a single comma-separated string,
// dropping any whose value is empty so the column stays compact.
func joinAttrs(pairs ...string) string {
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ", ")
}

// kv formats one attribute as "k=v", returning "" when v is empty so the
// caller can drop it via joinAttrs.
func kv(k, v string) string {
	if v == "" {
		return ""
	}
	return k + "=" + v
}

// refName returns the Name of a NestedRef or "" if the ref is nil.
func refName(r *netbox.NestedRef) string {
	if r == nil {
		return ""
	}
	return r.Name
}

// --- Per-type extractors ---------------------------------------------------

func siteAttrs(raw json.RawMessage) string {
	var s netbox.Site
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return joinAttrs(
		kv("status", s.Status.Label),
		kv("region", refName(s.Region)),
		kv("tenant", refName(s.Tenant)),
	)
}

func rackAttrs(raw json.RawMessage) string {
	var r netbox.Rack
	if json.Unmarshal(raw, &r) != nil {
		return ""
	}
	return joinAttrs(
		kv("site", refName(r.Site)),
		kv("location", refName(r.Location)),
		kv("status", r.Status.Label),
		kv("tenant", refName(r.Tenant)),
	)
}

func deviceAttrs(raw json.RawMessage) string {
	var d netbox.Device
	if json.Unmarshal(raw, &d) != nil {
		return ""
	}
	manu := ""
	if d.DeviceType != nil && d.DeviceType.Manufacturer != nil {
		manu = d.DeviceType.Manufacturer.Name
	}
	return joinAttrs(
		kv("site", refName(d.Site)),
		kv("rack", refName(d.Rack)),
		kv("role", refName(d.Role)),
		kv("manufacturer", manu),
		kv("status", d.Status.Label),
		kv("tenant", refName(d.Tenant)),
	)
}

func interfaceAttrs(raw json.RawMessage) string {
	var i netbox.Interface
	if json.Unmarshal(raw, &i) != nil {
		return ""
	}
	enabled := "no"
	if i.Enabled {
		enabled = "yes"
	}
	return joinAttrs(
		kv("device", refName(i.Device)),
		kv("type", i.Type.Label),
		kv("enabled", enabled),
	)
}

func prefixAttrs(raw json.RawMessage) string {
	var p netbox.Prefix
	if json.Unmarshal(raw, &p) != nil {
		return ""
	}
	return joinAttrs(
		kv("vrf", refName(p.VRF)),
		kv("site", refName(p.Site)),
		kv("status", p.Status.Label),
		kv("tenant", refName(p.Tenant)),
	)
}

func ipAddressAttrs(raw json.RawMessage) string {
	var ip netbox.IPAddress
	if json.Unmarshal(raw, &ip) != nil {
		return ""
	}
	return joinAttrs(
		kv("vrf", refName(ip.VRF)),
		kv("status", ip.Status.Label),
		kv("tenant", refName(ip.Tenant)),
		kv("dns_name", ip.DNSName),
	)
}

func vlanAttrs(raw json.RawMessage) string {
	var v netbox.VLAN
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	return joinAttrs(
		kv("vid", strconv.Itoa(v.VID)),
		kv("site", refName(v.Site)),
		kv("status", v.Status.Label),
		kv("tenant", refName(v.Tenant)),
	)
}

func vrfAttrs(raw json.RawMessage) string {
	var v netbox.VRF
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	return joinAttrs(
		kv("rd", v.RD),
		kv("tenant", refName(v.Tenant)),
	)
}

func tenantAttrs(raw json.RawMessage) string {
	var t netbox.Tenant
	if json.Unmarshal(raw, &t) != nil {
		return ""
	}
	return joinAttrs(
		kv("group", refName(t.Group)),
		kv("slug", t.Slug),
	)
}

func contactAttrs(raw json.RawMessage) string {
	var c netbox.Contact
	if json.Unmarshal(raw, &c) != nil {
		return ""
	}
	return joinAttrs(
		kv("title", c.Title),
		kv("email", c.Email),
		kv("phone", c.Phone),
		kv("group", refName(c.Group)),
	)
}

func vmAttrs(raw json.RawMessage) string {
	var v netbox.VirtualMachine
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	return joinAttrs(
		kv("site", refName(v.Site)),
		kv("cluster", refName(v.Cluster)),
		kv("tenant", refName(v.Tenant)),
		kv("status", v.Status.Label),
		kv("role", refName(v.Role)),
	)
}

func clusterAttrs(raw json.RawMessage) string {
	var c netbox.Cluster
	if json.Unmarshal(raw, &c) != nil {
		return ""
	}
	return joinAttrs(
		kv("type", refName(c.Type)),
		kv("site", refName(c.Site)),
		kv("status", c.Status.Label),
		kv("tenant", refName(c.Tenant)),
	)
}
