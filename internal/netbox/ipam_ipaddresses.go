package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// IPAddress is the /ipam/ip-addresses/ resource (subset).
type IPAddress struct {
	ID                 int        `json:"id"`
	URL                string     `json:"url"`
	Display            string     `json:"display"`
	Family             LabelValue `json:"family"`
	Address            string     `json:"address"`
	VRF                *NestedRef `json:"vrf,omitempty"`
	Tenant             *NestedRef `json:"tenant,omitempty"`
	Status             LabelValue `json:"status"`
	Role               LabelValue `json:"role,omitempty"`
	AssignedObjectType string     `json:"assigned_object_type,omitempty"`
	AssignedObjectID   *int       `json:"assigned_object_id,omitempty"`
	DNSName            string     `json:"dns_name,omitempty"`
	Description        string     `json:"description,omitempty"`
}

// ListIPAddressesOptions filters /ipam/ip-addresses/.
type ListIPAddressesOptions struct {
	Address string
	VRF     string
	Family  string
	Status  string
	Role    string
	Tenant  string
	Parent  string // parent prefix in CIDR
	Device  string
	VM      string
	Limit   int
	Offset  int
	Extra   url.Values
}

// ListIPAddresses returns one page of IP addresses.
func (c *Client) ListIPAddresses(ctx context.Context, opts ListIPAddressesOptions) (Page[IPAddress], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	setIf(q, "address", opts.Address)
	setIf(q, "vrf", opts.VRF)
	setIf(q, "family", opts.Family)
	setIf(q, "status", opts.Status)
	setIf(q, "role", opts.Role)
	setIf(q, "tenant", opts.Tenant)
	setIf(q, "parent", opts.Parent)
	setIf(q, "device", opts.Device)
	setIf(q, "virtual_machine", opts.VM)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[IPAddress]
	if err := c.Do(ctx, "GET", "/api/ipam/ip-addresses/", q, nil, &page); err != nil {
		return Page[IPAddress]{}, fmt.Errorf("ipam: list ip-addresses: %w", err)
	}
	return page, nil
}

// IPAddressesFetcher returns a PageFetcher bound to opts.
func (c *Client) IPAddressesFetcher(opts ListIPAddressesOptions) PageFetcher[IPAddress] {
	return func(ctx context.Context, offset, limit int) (Page[IPAddress], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListIPAddresses(ctx, opts)
	}
}
