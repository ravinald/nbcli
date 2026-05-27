package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Rack is the /dcim/racks/ resource (subset).
type Rack struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	Display     string     `json:"display"`
	Name        string     `json:"name"`
	Site        *NestedRef `json:"site,omitempty"`
	Location    *NestedRef `json:"location,omitempty"`
	Tenant      *NestedRef `json:"tenant,omitempty"`
	Role        *NestedRef `json:"role,omitempty"`
	Status      LabelValue `json:"status"`
	Serial      string     `json:"serial,omitempty"`
	AssetTag    string     `json:"asset_tag,omitempty"`
	UHeight     int        `json:"u_height,omitempty"`
	Width       LabelValue `json:"width,omitempty"`
	Description string     `json:"description,omitempty"`
}

// ListRacksOptions filters /dcim/racks/.
type ListRacksOptions struct {
	Name     string
	Site     string
	Status   string
	Role     string
	Location string
	Tenant   string
	Limit    int
	Offset   int
	Extra    url.Values
}

// ListRacks returns one page of racks.
func (c *Client) ListRacks(ctx context.Context, opts ListRacksOptions) (Page[Rack], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	setIf(q, "name", opts.Name)
	setIf(q, "site", opts.Site)
	setIf(q, "status", opts.Status)
	setIf(q, "role", opts.Role)
	setIf(q, "location", opts.Location)
	setIf(q, "tenant", opts.Tenant)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[Rack]
	if err := c.Do(ctx, "GET", "/api/dcim/racks/", q, nil, &page); err != nil {
		return Page[Rack]{}, fmt.Errorf("dcim: list racks: %w", err)
	}
	return page, nil
}

// RacksFetcher returns a PageFetcher bound to opts.
func (c *Client) RacksFetcher(opts ListRacksOptions) PageFetcher[Rack] {
	return func(ctx context.Context, offset, limit int) (Page[Rack], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListRacks(ctx, opts)
	}
}

// setIf adds k=v to q only when v is non-empty. Local to this file so other
// resource files can use it without dragging in a new dependency.
func setIf(q url.Values, k, v string) {
	if v != "" {
		q.Set(k, v)
	}
}
