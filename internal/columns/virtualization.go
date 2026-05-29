package columns

import (
	"strconv"

	"github.com/ravinald/nbcli/internal/netbox"
)

// VirtualizationResources lists the resource keys covered by this file.
// Kept exported to satisfy the project's file-rule heuristic and to give
// callers an enumeration of virtualization-module entries.
var VirtualizationResources = []string{"virtual-machines", "clusters"}

// VMsSet returns the column menu for /virtualization/virtual-machines/.
func VMsSet() Set {
	return Set{
		Resource: "virtual-machines",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.VirtualMachine).ID) }),
			col("name", "Name", 22, func(r any) string { return r.(netbox.VirtualMachine).Name }),
			col("status", "Status", 12, func(r any) string { return r.(netbox.VirtualMachine).Status.Label }),
			col("cluster", "Cluster", 16, func(r any) string {
				v := r.(netbox.VirtualMachine)
				if v.Cluster == nil {
					return ""
				}
				return v.Cluster.Name
			}),
			col("site", "Site", 14, func(r any) string {
				v := r.(netbox.VirtualMachine)
				if v.Site == nil {
					return ""
				}
				return v.Site.Name
			}),
			col("vcpus", "vCPUs", 7, func(r any) string {
				v := r.(netbox.VirtualMachine)
				if v.VCPUs == nil {
					return ""
				}
				return strconv.FormatFloat(*v.VCPUs, 'f', -1, 64)
			}),
			col("memory", "MemMB", 8, func(r any) string {
				v := r.(netbox.VirtualMachine)
				if v.Memory == nil {
					return ""
				}
				return strconv.Itoa(*v.Memory)
			}),
			opt("tenant", "Tenant", 16, func(r any) string {
				v := r.(netbox.VirtualMachine)
				if v.Tenant == nil {
					return ""
				}
				return v.Tenant.Name
			}),
			opt("platform", "Platform", 16, func(r any) string {
				v := r.(netbox.VirtualMachine)
				if v.Platform == nil {
					return ""
				}
				return v.Platform.Name
			}),
			opt("role", "Role", 14, func(r any) string {
				v := r.(netbox.VirtualMachine)
				if v.Role == nil {
					return ""
				}
				return v.Role.Name
			}),
			opt("primary_ip4", "Primary IPv4", 22, func(r any) string {
				v := r.(netbox.VirtualMachine)
				if v.PrimaryIP4 == nil {
					return ""
				}
				return v.PrimaryIP4.Display
			}),
			opt("disk", "DiskGB", 8, func(r any) string {
				v := r.(netbox.VirtualMachine)
				if v.Disk == nil {
					return ""
				}
				return strconv.Itoa(*v.Disk)
			}),
			opt("description", "Description", 30, func(r any) string { return r.(netbox.VirtualMachine).Description }),
		},
	}
}

// ClustersSet returns the column menu for /virtualization/clusters/.
func ClustersSet() Set {
	return Set{
		Resource: "clusters",
		Columns: []Column{
			col("id", "ID", 6, func(r any) string { return strconv.Itoa(r.(netbox.Cluster).ID) }),
			col("name", "Name", 22, func(r any) string { return r.(netbox.Cluster).Name }),
			col("type", "Type", 18, func(r any) string {
				c := r.(netbox.Cluster)
				if c.Type == nil {
					return ""
				}
				return c.Type.Name
			}),
			col("group", "Group", 16, func(r any) string {
				c := r.(netbox.Cluster)
				if c.Group == nil {
					return ""
				}
				return c.Group.Name
			}),
			col("site", "Site", 14, func(r any) string {
				c := r.(netbox.Cluster)
				if c.Site == nil {
					return ""
				}
				return c.Site.Name
			}),
			col("status", "Status", 12, func(r any) string { return r.(netbox.Cluster).Status.Label }),
			col("vm_count", "VMs", 6, func(r any) string { return strconv.Itoa(r.(netbox.Cluster).VirtualMachineCount) }),
			opt("tenant", "Tenant", 16, func(r any) string {
				c := r.(netbox.Cluster)
				if c.Tenant == nil {
					return ""
				}
				return c.Tenant.Name
			}),
			opt("description", "Description", 30, func(r any) string { return r.(netbox.Cluster).Description }),
		},
	}
}
