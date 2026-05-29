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
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Prefix], error) {
		listOpts := netbox.ListPrefixesOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListPrefixes(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Prefix]{}, err
		}
		return FetchResult[netbox.Prefix]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Prefix]("Prefixes", cols, mapper, func(p netbox.Prefix) int { return p.ID }, fetcher)
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
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.IPAddress], error) {
		listOpts := netbox.ListIPAddressesOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListIPAddresses(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.IPAddress]{}, err
		}
		return FetchResult[netbox.IPAddress]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.IPAddress]("IP Addresses", cols, mapper, func(i netbox.IPAddress) int { return i.ID }, fetcher)
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
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.VLAN], error) {
		listOpts := netbox.ListVLANsOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListVLANs(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.VLAN]{}, err
		}
		return FetchResult[netbox.VLAN]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.VLAN]("VLANs", cols, mapper, func(v netbox.VLAN) int { return v.ID }, fetcher)
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
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.VRF], error) {
		listOpts := netbox.ListVRFsOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListVRFs(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.VRF]{}, err
		}
		return FetchResult[netbox.VRF]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.VRF]("VRFs", cols, mapper, func(v netbox.VRF) int { return v.ID }, fetcher)
}
