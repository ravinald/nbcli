package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Prefix is the /ipam/prefixes/ resource (subset).
type Prefix struct {
	ID           int        `json:"id"`
	URL          string     `json:"url"`
	Display      string     `json:"display"`
	Family       LabelValue `json:"family"`
	Prefix       string     `json:"prefix"`
	Site         *NestedRef `json:"site,omitempty"`
	VRF          *NestedRef `json:"vrf,omitempty"`
	Tenant       *NestedRef `json:"tenant,omitempty"`
	Role         *NestedRef `json:"role,omitempty"`
	VLAN         *NestedRef `json:"vlan,omitempty"`
	Status       LabelValue `json:"status"`
	IsPool       bool       `json:"is_pool,omitempty"`
	MarkUtilized bool       `json:"mark_utilized,omitempty"`
	Description  string     `json:"description,omitempty"`
}

// ListPrefixesOptions filters /ipam/prefixes/.
type ListPrefixesOptions struct {
	Prefix   string
	VRF      string
	Family   string
	Status   string
	Role     string
	Site     string
	Tenant   string
	Contains string
	Limit    int
	Offset   int
	Extra    url.Values
}

// ListPrefixes returns one page of prefixes.
func (c *Client) ListPrefixes(ctx context.Context, opts ListPrefixesOptions) (Page[Prefix], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	setIf(q, "prefix", opts.Prefix)
	setIf(q, "vrf", opts.VRF)
	setIf(q, "family", opts.Family)
	setIf(q, "status", opts.Status)
	setIf(q, "role", opts.Role)
	setIf(q, "site", opts.Site)
	setIf(q, "tenant", opts.Tenant)
	setIf(q, "contains", opts.Contains)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[Prefix]
	if err := c.Do(ctx, "GET", "/api/ipam/prefixes/", q, nil, &page); err != nil {
		return Page[Prefix]{}, fmt.Errorf("ipam: list prefixes: %w", err)
	}
	return page, nil
}

// PrefixesFetcher returns a PageFetcher bound to opts.
func (c *Client) PrefixesFetcher(opts ListPrefixesOptions) PageFetcher[Prefix] {
	return func(ctx context.Context, offset, limit int) (Page[Prefix], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListPrefixes(ctx, opts)
	}
}
