package columns

import (
	"strconv"

	"github.com/ravinald/nbcli/internal/netbox"
)

// SitesSet returns the column menu for /dcim/sites/.
func SitesSet() Set {
	return Set{
		Resource: "sites",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.Site).ID) }),
			col("name", "Name", 24, func(r any) string { return r.(netbox.Site).Name }),
			col("slug", "Slug", 24, func(r any) string { return r.(netbox.Site).Slug }),
			col("status", "Status", 12, func(r any) string { return r.(netbox.Site).Status.Label }),
			col("region", "Region", 16, func(r any) string {
				s := r.(netbox.Site)
				if s.Region == nil {
					return ""
				}
				return s.Region.Name
			}),
			col("tenant", "Tenant", 16, func(r any) string {
				s := r.(netbox.Site)
				if s.Tenant == nil {
					return ""
				}
				return s.Tenant.Name
			}),
			opt("facility", "Facility", 16, func(r any) string { return r.(netbox.Site).Facility }),
			opt("description", "Description", 30, func(r any) string { return r.(netbox.Site).Description }),
			opt("created", "Created", 24, func(r any) string { return r.(netbox.Site).Created }),
			opt("last_updated", "Last Updated", 24, func(r any) string { return r.(netbox.Site).LastUpdated }),
		},
	}
}

// RacksSet returns the column menu for /dcim/racks/.
func RacksSet() Set {
	return Set{
		Resource: "racks",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.Rack).ID) }),
			col("name", "Name", 18, func(r any) string { return r.(netbox.Rack).Name }),
			col("site", "Site", 14, func(r any) string {
				k := r.(netbox.Rack)
				if k.Site == nil {
					return ""
				}
				return k.Site.Name
			}),
			col("location", "Location", 18, func(r any) string {
				k := r.(netbox.Rack)
				if k.Location == nil {
					return ""
				}
				return k.Location.Name
			}),
			col("role", "Role", 14, func(r any) string {
				k := r.(netbox.Rack)
				if k.Role == nil {
					return ""
				}
				return k.Role.Name
			}),
			col("status", "Status", 12, func(r any) string { return r.(netbox.Rack).Status.Label }),
			col("u_height", "U", 4, func(r any) string { return strconv.Itoa(r.(netbox.Rack).UHeight) }),
			col("tenant", "Tenant", 16, func(r any) string {
				k := r.(netbox.Rack)
				if k.Tenant == nil {
					return ""
				}
				return k.Tenant.Name
			}),
			opt("width", "Width", 8, func(r any) string { return r.(netbox.Rack).Width.Label }),
			opt("serial", "Serial", 18, func(r any) string { return r.(netbox.Rack).Serial }),
			opt("asset_tag", "Asset Tag", 16, func(r any) string { return r.(netbox.Rack).AssetTag }),
			opt("description", "Description", 30, func(r any) string { return r.(netbox.Rack).Description }),
		},
	}
}

// DevicesSet returns the column menu for /dcim/devices/.
func DevicesSet() Set {
	return Set{
		Resource: "devices",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.Device).ID) }),
			col("name", "Name", 22, func(r any) string { return r.(netbox.Device).Name }),
			col("type", "Type", 22, func(r any) string {
				d := r.(netbox.Device)
				if d.DeviceType == nil {
					return ""
				}
				mfr := ""
				if d.DeviceType.Manufacturer != nil {
					mfr = d.DeviceType.Manufacturer.Name + " "
				}
				return mfr + d.DeviceType.Model
			}),
			col("role", "Role", 14, func(r any) string {
				d := r.(netbox.Device)
				if d.Role == nil {
					return ""
				}
				return d.Role.Name
			}),
			col("site", "Site", 14, func(r any) string {
				d := r.(netbox.Device)
				if d.Site == nil {
					return ""
				}
				return d.Site.Name
			}),
			col("rack", "Rack", 12, func(r any) string {
				d := r.(netbox.Device)
				if d.Rack == nil {
					return ""
				}
				return d.Rack.Name
			}),
			col("status", "Status", 12, func(r any) string { return r.(netbox.Device).Status.Label }),
			opt("tenant", "Tenant", 16, func(r any) string {
				d := r.(netbox.Device)
				if d.Tenant == nil {
					return ""
				}
				return d.Tenant.Name
			}),
			opt("platform", "Platform", 16, func(r any) string {
				d := r.(netbox.Device)
				if d.Platform == nil {
					return ""
				}
				return d.Platform.Name
			}),
			opt("primary_ip4", "Primary IPv4", 22, func(r any) string {
				d := r.(netbox.Device)
				if d.PrimaryIP4 == nil {
					return ""
				}
				return d.PrimaryIP4.Display
			}),
			opt("primary_ip6", "Primary IPv6", 28, func(r any) string {
				d := r.(netbox.Device)
				if d.PrimaryIP6 == nil {
					return ""
				}
				return d.PrimaryIP6.Display
			}),
			opt("serial", "Serial", 18, func(r any) string { return r.(netbox.Device).Serial }),
			opt("asset_tag", "Asset Tag", 16, func(r any) string { return r.(netbox.Device).AssetTag }),
			opt("description", "Description", 30, func(r any) string { return r.(netbox.Device).Description }),
		},
	}
}

// InterfacesSet returns the column menu for /dcim/interfaces/.
func InterfacesSet() Set {
	return Set{
		Resource: "interfaces",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.Interface).ID) }),
			col("device", "Device", 20, func(r any) string {
				i := r.(netbox.Interface)
				if i.Device == nil {
					return ""
				}
				return i.Device.Name
			}),
			col("name", "Name", 18, func(r any) string { return r.(netbox.Interface).Name }),
			col("type", "Type", 18, func(r any) string { return r.(netbox.Interface).Type.Label }),
			col("enabled", "Enabled", 8, func(r any) string { return strconv.FormatBool(r.(netbox.Interface).Enabled) }),
			col("mac_address", "MAC", 18, func(r any) string { return r.(netbox.Interface).MACAddress }),
			opt("label", "Label", 16, func(r any) string { return r.(netbox.Interface).Label }),
			opt("lag", "LAG", 12, func(r any) string {
				i := r.(netbox.Interface)
				if i.LAG == nil {
					return ""
				}
				return i.LAG.Name
			}),
			opt("mtu", "MTU", 6, func(r any) string {
				i := r.(netbox.Interface)
				if i.MTU == nil {
					return ""
				}
				return strconv.Itoa(*i.MTU)
			}),
			opt("speed", "Speed", 10, func(r any) string {
				i := r.(netbox.Interface)
				if i.Speed == nil {
					return ""
				}
				return strconv.Itoa(*i.Speed)
			}),
			opt("mgmt_only", "Mgmt Only", 10, func(r any) string { return strconv.FormatBool(r.(netbox.Interface).MgmtOnly) }),
			opt("description", "Description", 30, func(r any) string { return r.(netbox.Interface).Description }),
		},
	}
}
