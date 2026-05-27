package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// VRF is the /ipam/vrfs/ resource (subset).
type VRF struct {
	ID            int        `json:"id"`
	URL           string     `json:"url"`
	Display       string     `json:"display"`
	Name          string     `json:"name"`
	RD            string     `json:"rd,omitempty"`
	Tenant        *NestedRef `json:"tenant,omitempty"`
	EnforceUnique bool       `json:"enforce_unique,omitempty"`
	Description   string     `json:"description,omitempty"`
}

// ListVRFsOptions filters /ipam/vrfs/.
type ListVRFsOptions struct {
	Name   string
	RD     string
	Tenant string
	Limit  int
	Offset int
	Extra  url.Values
}

// ListVRFs returns one page of VRFs.
func (c *Client) ListVRFs(ctx context.Context, opts ListVRFsOptions) (Page[VRF], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	setIf(q, "name", opts.Name)
	setIf(q, "rd", opts.RD)
	setIf(q, "tenant", opts.Tenant)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[VRF]
	if err := c.Do(ctx, "GET", "/api/ipam/vrfs/", q, nil, &page); err != nil {
		return Page[VRF]{}, fmt.Errorf("ipam: list vrfs: %w", err)
	}
	return page, nil
}

// VRFsFetcher returns a PageFetcher bound to opts.
func (c *Client) VRFsFetcher(opts ListVRFsOptions) PageFetcher[VRF] {
	return func(ctx context.Context, offset, limit int) (Page[VRF], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListVRFs(ctx, opts)
	}
}
