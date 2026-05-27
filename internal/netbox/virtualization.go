package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// VirtualMachine is the /virtualization/virtual-machines/ resource (subset).
type VirtualMachine struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	Display     string     `json:"display"`
	Name        string     `json:"name"`
	Status      LabelValue `json:"status"`
	Site        *NestedRef `json:"site,omitempty"`
	Cluster     *NestedRef `json:"cluster,omitempty"`
	Tenant      *NestedRef `json:"tenant,omitempty"`
	Platform    *NestedRef `json:"platform,omitempty"`
	Role        *NestedRef `json:"role,omitempty"`
	PrimaryIP4  *NestedRef `json:"primary_ip4,omitempty"`
	PrimaryIP6  *NestedRef `json:"primary_ip6,omitempty"`
	VCPUs       *float64   `json:"vcpus,omitempty"`
	Memory      *int       `json:"memory,omitempty"`
	Disk        *int       `json:"disk,omitempty"`
	Description string     `json:"description,omitempty"`
}

// ListVMsOptions filters /virtualization/virtual-machines/.
type ListVMsOptions struct {
	Name    string
	Status  string
	Site    string
	Cluster string
	Tenant  string
	Role    string
	Limit   int
	Offset  int
	Extra   url.Values
}

// ListVMs returns one page of virtual machines.
func (c *Client) ListVMs(ctx context.Context, opts ListVMsOptions) (Page[VirtualMachine], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	setIf(q, "name", opts.Name)
	setIf(q, "status", opts.Status)
	setIf(q, "site", opts.Site)
	setIf(q, "cluster", opts.Cluster)
	setIf(q, "tenant", opts.Tenant)
	setIf(q, "role", opts.Role)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[VirtualMachine]
	if err := c.Do(ctx, "GET", "/api/virtualization/virtual-machines/", q, nil, &page); err != nil {
		return Page[VirtualMachine]{}, fmt.Errorf("virtualization: list vms: %w", err)
	}
	return page, nil
}

// VMsFetcher returns a PageFetcher bound to opts.
func (c *Client) VMsFetcher(opts ListVMsOptions) PageFetcher[VirtualMachine] {
	return func(ctx context.Context, offset, limit int) (Page[VirtualMachine], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListVMs(ctx, opts)
	}
}

// Cluster is the /virtualization/clusters/ resource (subset).
type Cluster struct {
	ID                  int        `json:"id"`
	URL                 string     `json:"url"`
	Display             string     `json:"display"`
	Name                string     `json:"name"`
	Type                *NestedRef `json:"type,omitempty"`
	Group               *NestedRef `json:"group,omitempty"`
	Tenant              *NestedRef `json:"tenant,omitempty"`
	Site                *NestedRef `json:"site,omitempty"`
	Status              LabelValue `json:"status"`
	Description         string     `json:"description,omitempty"`
	VirtualMachineCount int        `json:"virtualmachine_count,omitempty"`
}

// ListClustersOptions filters /virtualization/clusters/.
type ListClustersOptions struct {
	Name   string
	Type   string
	Group  string
	Site   string
	Status string
	Limit  int
	Offset int
	Extra  url.Values
}

// ListClusters returns one page of clusters.
func (c *Client) ListClusters(ctx context.Context, opts ListClustersOptions) (Page[Cluster], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	setIf(q, "name", opts.Name)
	setIf(q, "type", opts.Type)
	setIf(q, "group", opts.Group)
	setIf(q, "site", opts.Site)
	setIf(q, "status", opts.Status)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[Cluster]
	if err := c.Do(ctx, "GET", "/api/virtualization/clusters/", q, nil, &page); err != nil {
		return Page[Cluster]{}, fmt.Errorf("virtualization: list clusters: %w", err)
	}
	return page, nil
}

// ClustersFetcher returns a PageFetcher bound to opts.
func (c *Client) ClustersFetcher(opts ListClustersOptions) PageFetcher[Cluster] {
	return func(ctx context.Context, offset, limit int) (Page[Cluster], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListClusters(ctx, opts)
	}
}
