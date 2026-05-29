// ipam.go holds the bubbletea factories for the Netbox IPAM module:
// Prefixes, IP Addresses, VLANs, and VRFs.

package views

import (
	"context"

	"github.com/ravinald/nbcli/internal/netbox"
)

// NewPrefixes returns a View listing /ipam/prefixes/.
func NewPrefixes(client *netbox.Client, resolve ColumnsResolver) View {
	visible := resolve("prefixes")
	cols, mapper := buildCols[netbox.Prefix](visible)
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
func NewIPAddresses(client *netbox.Client, resolve ColumnsResolver) View {
	visible := resolve("ip-addresses")
	cols, mapper := buildCols[netbox.IPAddress](visible)
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
func NewVLANs(client *netbox.Client, resolve ColumnsResolver) View {
	visible := resolve("vlans")
	cols, mapper := buildCols[netbox.VLAN](visible)
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
func NewVRFs(client *netbox.Client, resolve ColumnsResolver) View {
	visible := resolve("vrfs")
	cols, mapper := buildCols[netbox.VRF](visible)
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
