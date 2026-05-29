package columns

import (
	"strconv"

	"github.com/ravinald/nbcli/internal/netbox"
)

// PrefixesSet returns the column menu for /ipam/prefixes/.
func PrefixesSet() Set {
	return Set{
		Resource: "prefixes",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.Prefix).ID) }),
			col("prefix", "Prefix", 24, func(r any) string { return r.(netbox.Prefix).Prefix }),
			col("family", "Family", 7, func(r any) string { return r.(netbox.Prefix).Family.Label }),
			col("vrf", "VRF", 14, func(r any) string {
				p := r.(netbox.Prefix)
				if p.VRF == nil {
					return ""
				}
				return p.VRF.Name
			}),
			col("site", "Site", 14, func(r any) string {
				p := r.(netbox.Prefix)
				if p.Site == nil {
					return ""
				}
				return p.Site.Name
			}),
			col("status", "Status", 12, func(r any) string { return r.(netbox.Prefix).Status.Label }),
			opt("role", "Role", 14, func(r any) string {
				p := r.(netbox.Prefix)
				if p.Role == nil {
					return ""
				}
				return p.Role.Name
			}),
			opt("tenant", "Tenant", 16, func(r any) string {
				p := r.(netbox.Prefix)
				if p.Tenant == nil {
					return ""
				}
				return p.Tenant.Name
			}),
			opt("vlan", "VLAN", 12, func(r any) string {
				p := r.(netbox.Prefix)
				if p.VLAN == nil {
					return ""
				}
				return p.VLAN.Name
			}),
			opt("is_pool", "Is Pool", 8, func(r any) string { return strconv.FormatBool(r.(netbox.Prefix).IsPool) }),
			opt("description", "Description", 30, func(r any) string { return r.(netbox.Prefix).Description }),
		},
	}
}

// IPAddressesSet returns the column menu for /ipam/ip-addresses/.
func IPAddressesSet() Set {
	return Set{
		Resource: "ip-addresses",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.IPAddress).ID) }),
			col("address", "Address", 22, func(r any) string { return r.(netbox.IPAddress).Address }),
			col("family", "Family", 7, func(r any) string { return r.(netbox.IPAddress).Family.Label }),
			col("vrf", "VRF", 14, func(r any) string {
				i := r.(netbox.IPAddress)
				if i.VRF == nil {
					return ""
				}
				return i.VRF.Name
			}),
			col("status", "Status", 12, func(r any) string { return r.(netbox.IPAddress).Status.Label }),
			col("dns_name", "DNS", 24, func(r any) string { return r.(netbox.IPAddress).DNSName }),
			opt("role", "Role", 12, func(r any) string { return r.(netbox.IPAddress).Role.Label }),
			opt("tenant", "Tenant", 16, func(r any) string {
				i := r.(netbox.IPAddress)
				if i.Tenant == nil {
					return ""
				}
				return i.Tenant.Name
			}),
			opt("description", "Description", 30, func(r any) string { return r.(netbox.IPAddress).Description }),
		},
	}
}

// VLANsSet returns the column menu for /ipam/vlans/.
func VLANsSet() Set {
	return Set{
		Resource: "vlans",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.VLAN).ID) }),
			col("vid", "VID", 6, func(r any) string { return strconv.Itoa(r.(netbox.VLAN).VID) }),
			col("name", "Name", 22, func(r any) string { return r.(netbox.VLAN).Name }),
			col("site", "Site", 14, func(r any) string {
				v := r.(netbox.VLAN)
				if v.Site == nil {
					return ""
				}
				return v.Site.Name
			}),
			col("group", "Group", 14, func(r any) string {
				v := r.(netbox.VLAN)
				if v.Group == nil {
					return ""
				}
				return v.Group.Name
			}),
			col("status", "Status", 12, func(r any) string { return r.(netbox.VLAN).Status.Label }),
			col("role", "Role", 14, func(r any) string {
				v := r.(netbox.VLAN)
				if v.Role == nil {
					return ""
				}
				return v.Role.Name
			}),
			opt("tenant", "Tenant", 16, func(r any) string {
				v := r.(netbox.VLAN)
				if v.Tenant == nil {
					return ""
				}
				return v.Tenant.Name
			}),
			opt("description", "Description", 30, func(r any) string { return r.(netbox.VLAN).Description }),
		},
	}
}

// VRFsSet returns the column menu for /ipam/vrfs/.
func VRFsSet() Set {
	return Set{
		Resource: "vrfs",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.VRF).ID) }),
			col("name", "Name", 20, func(r any) string { return r.(netbox.VRF).Name }),
			col("rd", "RD", 18, func(r any) string { return r.(netbox.VRF).RD }),
			col("tenant", "Tenant", 16, func(r any) string {
				v := r.(netbox.VRF)
				if v.Tenant == nil {
					return ""
				}
				return v.Tenant.Name
			}),
			col("description", "Description", 30, func(r any) string { return r.(netbox.VRF).Description }),
			opt("enforce_unique", "Enforce Unique", 14, func(r any) string { return strconv.FormatBool(r.(netbox.VRF).EnforceUnique) }),
		},
	}
}
