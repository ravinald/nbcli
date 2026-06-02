// dcim.go holds the bubbletea factories for the Netbox DCIM module: Sites,
// Racks, Devices, and Interfaces.

package views

import (
	"context"

	"github.com/ravinald/nbcli/internal/netbox"
)

// NewSites returns a View listing /dcim/sites/.
func NewSites(client *netbox.Client, resolve ColumnsResolver) View {
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Site], error) {
		listOpts := netbox.ListSitesOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListSites(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Site]{}, err
		}
		return FetchResult[netbox.Site]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Site]("Sites", "sites", resolve, func(s netbox.Site) int { return s.ID }, fetcher)
}

// NewRacks returns a View listing /dcim/racks/.
func NewRacks(client *netbox.Client, resolve ColumnsResolver) View {
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Rack], error) {
		listOpts := netbox.ListRacksOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListRacks(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Rack]{}, err
		}
		return FetchResult[netbox.Rack]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Rack]("Racks", "racks", resolve, func(r netbox.Rack) int { return r.ID }, fetcher)
}

// NewDevices returns a View listing /dcim/devices/.
func NewDevices(client *netbox.Client, resolve ColumnsResolver) View {
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Device], error) {
		listOpts := netbox.ListDevicesOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListDevices(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Device]{}, err
		}
		return FetchResult[netbox.Device]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Device]("Devices", "devices", resolve, func(d netbox.Device) int { return d.ID }, fetcher)
}

// NewInterfaces returns a View listing /dcim/interfaces/.
func NewInterfaces(client *netbox.Client, resolve ColumnsResolver) View {
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Interface], error) {
		listOpts := netbox.ListInterfacesOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListInterfaces(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Interface]{}, err
		}
		return FetchResult[netbox.Interface]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Interface]("Interfaces", "interfaces", resolve, func(i netbox.Interface) int { return i.ID }, fetcher)
}
