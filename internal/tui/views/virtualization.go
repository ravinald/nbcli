// virtualization.go holds the bubbletea factories for the Netbox
// Virtualization module: Virtual Machines and Clusters.

package views

import (
	"context"

	"github.com/ravinald/nbcli/internal/netbox"
)

// NewVMs returns a View listing /virtualization/virtual-machines/.
func NewVMs(client *netbox.Client, resolve ColumnsResolver) View {
	visible := resolve("virtual-machines")
	cols, mapper := buildCols[netbox.VirtualMachine](visible)
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
func NewClusters(client *netbox.Client, resolve ColumnsResolver) View {
	visible := resolve("clusters")
	cols, mapper := buildCols[netbox.Cluster](visible)
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
