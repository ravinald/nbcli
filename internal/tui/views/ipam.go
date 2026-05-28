// ipam.go holds the bubbletea factories for the Netbox IPAM module:
// Prefixes, IP Addresses, VLANs, and VRFs.

package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"

	"github.com/ravinald/nbcli/internal/netbox"
)

// NewPrefixes returns a View listing /ipam/prefixes/.
func NewPrefixes(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Prefix", Width: 24},
		{Title: "Family", Width: 7},
		{Title: "VRF", Width: 14},
		{Title: "Site", Width: 14},
		{Title: "Status", Width: 12},
	}
	mapper := func(p netbox.Prefix) table.Row {
		return table.Row{
			strconv.Itoa(p.ID),
			p.Prefix,
			p.Family.Label,
			nestedName(p.VRF),
			nestedName(p.Site),
			p.Status.Label,
		}
	}
	fetcher := func(ctx context.Context) ([]netbox.Prefix, error) {
		return netbox.ListAll(ctx,
			client.PrefixesFetcher(netbox.ListPrefixesOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
	}
	return newBaseView[netbox.Prefix]("Prefixes", cols, mapper, fetcher)
}

// NewIPAddresses returns a View listing /ipam/ip-addresses/.
func NewIPAddresses(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Address", Width: 22},
		{Title: "Family", Width: 7},
		{Title: "VRF", Width: 14},
		{Title: "Status", Width: 12},
		{Title: "DNS", Width: 24},
	}
	mapper := func(ip netbox.IPAddress) table.Row {
		return table.Row{
			strconv.Itoa(ip.ID),
			ip.Address,
			ip.Family.Label,
			nestedName(ip.VRF),
			ip.Status.Label,
			ip.DNSName,
		}
	}
	fetcher := func(ctx context.Context) ([]netbox.IPAddress, error) {
		return netbox.ListAll(ctx,
			client.IPAddressesFetcher(netbox.ListIPAddressesOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
	}
	return newBaseView[netbox.IPAddress]("IP Addresses", cols, mapper, fetcher)
}

// NewVLANs returns a View listing /ipam/vlans/.
func NewVLANs(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "VID", Width: 6},
		{Title: "Name", Width: 22},
		{Title: "Site", Width: 14},
		{Title: "Group", Width: 14},
		{Title: "Status", Width: 12},
		{Title: "Role", Width: 14},
	}
	mapper := func(l netbox.VLAN) table.Row {
		return table.Row{
			strconv.Itoa(l.ID),
			strconv.Itoa(l.VID),
			l.Name,
			nestedName(l.Site),
			nestedName(l.Group),
			l.Status.Label,
			nestedName(l.Role),
		}
	}
	fetcher := func(ctx context.Context) ([]netbox.VLAN, error) {
		return netbox.ListAll(ctx,
			client.VLANsFetcher(netbox.ListVLANsOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
	}
	return newBaseView[netbox.VLAN]("VLANs", cols, mapper, fetcher)
}

// NewVRFs returns a View listing /ipam/vrfs/.
func NewVRFs(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 20},
		{Title: "RD", Width: 18},
		{Title: "Tenant", Width: 16},
		{Title: "Description", Width: 30},
	}
	mapper := func(r netbox.VRF) table.Row {
		return table.Row{
			strconv.Itoa(r.ID),
			r.Name,
			r.RD,
			nestedName(r.Tenant),
			r.Description,
		}
	}
	fetcher := func(ctx context.Context) ([]netbox.VRF, error) {
		return netbox.ListAll(ctx,
			client.VRFsFetcher(netbox.ListVRFsOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
	}
	return newBaseView[netbox.VRF]("VRFs", cols, mapper, fetcher)
}
