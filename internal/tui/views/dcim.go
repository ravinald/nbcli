// dcim.go holds the bubbletea factories for the Netbox DCIM module: Sites,
// Racks, Devices, and Interfaces.

package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"

	"github.com/ravinald/nbcli/internal/netbox"
)

// NewSites returns a View listing /dcim/sites/.
func NewSites(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 22},
		{Title: "Slug", Width: 22},
		{Title: "Status", Width: 12},
		{Title: "Region", Width: 16},
		{Title: "Tenant", Width: 16},
	}
	mapper := func(s netbox.Site) table.Row {
		return table.Row{
			strconv.Itoa(s.ID),
			s.Name,
			s.Slug,
			s.Status.Label,
			nestedName(s.Region),
			nestedName(s.Tenant),
		}
	}
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Site], error) {
		listOpts := netbox.ListSitesOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListSites(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Site]{}, err
		}
		return FetchResult[netbox.Site]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Site]("Sites", cols, mapper, func(s netbox.Site) int { return s.ID }, fetcher)
}

// NewRacks returns a View listing /dcim/racks/.
func NewRacks(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 18},
		{Title: "Site", Width: 14},
		{Title: "Location", Width: 18},
		{Title: "Role", Width: 14},
		{Title: "Status", Width: 12},
		{Title: "U", Width: 4},
	}
	mapper := func(r netbox.Rack) table.Row {
		return table.Row{
			strconv.Itoa(r.ID),
			r.Name,
			nestedName(r.Site),
			nestedName(r.Location),
			nestedName(r.Role),
			r.Status.Label,
			strconv.Itoa(r.UHeight),
		}
	}
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Rack], error) {
		listOpts := netbox.ListRacksOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListRacks(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Rack]{}, err
		}
		return FetchResult[netbox.Rack]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Rack]("Racks", cols, mapper, func(r netbox.Rack) int { return r.ID }, fetcher)
}

// NewDevices returns a View listing /dcim/devices/.
func NewDevices(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 22},
		{Title: "Type", Width: 22},
		{Title: "Site", Width: 14},
		{Title: "Rack", Width: 12},
		{Title: "Status", Width: 12},
	}
	mapper := func(d netbox.Device) table.Row {
		devType := ""
		if d.DeviceType != nil {
			if d.DeviceType.Manufacturer != nil {
				devType = d.DeviceType.Manufacturer.Name + " "
			}
			devType += d.DeviceType.Model
		}
		return table.Row{
			strconv.Itoa(d.ID),
			d.Name,
			devType,
			nestedName(d.Site),
			nestedName(d.Rack),
			d.Status.Label,
		}
	}
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Device], error) {
		listOpts := netbox.ListDevicesOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListDevices(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Device]{}, err
		}
		return FetchResult[netbox.Device]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Device]("Devices", cols, mapper, func(d netbox.Device) int { return d.ID }, fetcher)
}

// NewInterfaces returns a View listing /dcim/interfaces/. Interfaces are
// often hundreds of thousands across a fleet; pagination keeps the page
// load bounded.
func NewInterfaces(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Device", Width: 20},
		{Title: "Name", Width: 18},
		{Title: "Type", Width: 18},
		{Title: "Enabled", Width: 8},
		{Title: "MAC", Width: 18},
	}
	mapper := func(i netbox.Interface) table.Row {
		return table.Row{
			strconv.Itoa(i.ID),
			nestedName(i.Device),
			i.Name,
			i.Type.Label,
			strconv.FormatBool(i.Enabled),
			i.MACAddress,
		}
	}
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Interface], error) {
		listOpts := netbox.ListInterfacesOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListInterfaces(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Interface]{}, err
		}
		return FetchResult[netbox.Interface]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Interface]("Interfaces", cols, mapper, func(i netbox.Interface) int { return i.ID }, fetcher)
}
