package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// VLAN is the /ipam/vlans/ resource (subset).
type VLAN struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	Display     string     `json:"display"`
	VID         int        `json:"vid"`
	Name        string     `json:"name"`
	Site        *NestedRef `json:"site,omitempty"`
	Group       *NestedRef `json:"group,omitempty"`
	Tenant      *NestedRef `json:"tenant,omitempty"`
	Status      LabelValue `json:"status"`
	Role        *NestedRef `json:"role,omitempty"`
	Description string     `json:"description,omitempty"`
}

// ListVLANsOptions filters /ipam/vlans/.
type ListVLANsOptions struct {
	VID    string
	Name   string
	Site   string
	Group  string
	Role   string
	Status string
	Tenant string
	Limit  int
	Offset int
	Extra  url.Values
}

// ListVLANs returns one page of VLANs.
func (c *Client) ListVLANs(ctx context.Context, opts ListVLANsOptions) (Page[VLAN], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	setIf(q, "vid", opts.VID)
	setIf(q, "name", opts.Name)
	setIf(q, "site", opts.Site)
	setIf(q, "group", opts.Group)
	setIf(q, "role", opts.Role)
	setIf(q, "status", opts.Status)
	setIf(q, "tenant", opts.Tenant)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[VLAN]
	if err := c.Do(ctx, "GET", "/api/ipam/vlans/", q, nil, &page); err != nil {
		return Page[VLAN]{}, fmt.Errorf("ipam: list vlans: %w", err)
	}
	return page, nil
}

// VLANsFetcher returns a PageFetcher bound to opts.
func (c *Client) VLANsFetcher(opts ListVLANsOptions) PageFetcher[VLAN] {
	return func(ctx context.Context, offset, limit int) (Page[VLAN], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListVLANs(ctx, opts)
	}
}
