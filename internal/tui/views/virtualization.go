// virtualization.go holds the bubbletea factories for the Netbox
// Virtualization module: Virtual Machines and Clusters.

package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"

	"github.com/ravinald/nbcli/internal/netbox"
)

// NewVMs returns a View listing /virtualization/virtual-machines/.
func NewVMs(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 22},
		{Title: "Status", Width: 12},
		{Title: "Cluster", Width: 16},
		{Title: "Site", Width: 14},
		{Title: "vCPUs", Width: 7},
		{Title: "MemMB", Width: 8},
	}
	mapper := func(vm netbox.VirtualMachine) table.Row {
		vcpu := ""
		if vm.VCPUs != nil {
			vcpu = strconv.FormatFloat(*vm.VCPUs, 'f', -1, 64)
		}
		mem := ""
		if vm.Memory != nil {
			mem = strconv.Itoa(*vm.Memory)
		}
		return table.Row{
			strconv.Itoa(vm.ID),
			vm.Name,
			vm.Status.Label,
			nestedName(vm.Cluster),
			nestedName(vm.Site),
			vcpu,
			mem,
		}
	}
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.VirtualMachine], error) {
		listOpts := netbox.ListVMsOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListVMs(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.VirtualMachine]{}, err
		}
		return FetchResult[netbox.VirtualMachine]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.VirtualMachine]("Virtual Machines", cols, mapper, func(v netbox.VirtualMachine) int { return v.ID }, fetcher)
}

// NewClusters returns a View listing /virtualization/clusters/.
func NewClusters(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 22},
		{Title: "Type", Width: 18},
		{Title: "Group", Width: 16},
		{Title: "Site", Width: 14},
		{Title: "Status", Width: 12},
		{Title: "VMs", Width: 6},
	}
	mapper := func(c netbox.Cluster) table.Row {
		return table.Row{
			strconv.Itoa(c.ID),
			c.Name,
			nestedName(c.Type),
			nestedName(c.Group),
			nestedName(c.Site),
			c.Status.Label,
			strconv.Itoa(c.VirtualMachineCount),
		}
	}
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Cluster], error) {
		listOpts := netbox.ListClustersOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListClusters(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Cluster]{}, err
		}
		return FetchResult[netbox.Cluster]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Cluster]("Clusters", cols, mapper, func(c netbox.Cluster) int { return c.ID }, fetcher)
}
