package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Interface is the /dcim/interfaces/ resource (subset).
type Interface struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	Display     string     `json:"display"`
	Name        string     `json:"name"`
	Device      *NestedRef `json:"device,omitempty"`
	Label       string     `json:"label,omitempty"`
	Type        LabelValue `json:"type"`
	Enabled     bool       `json:"enabled"`
	LAG         *NestedRef `json:"lag,omitempty"`
	MTU         *int       `json:"mtu,omitempty"`
	MACAddress  string     `json:"mac_address,omitempty"`
	Speed       *int       `json:"speed,omitempty"`
	MgmtOnly    bool       `json:"mgmt_only"`
	Description string     `json:"description,omitempty"`
}

// ListInterfacesOptions filters /dcim/interfaces/. Enabled/MgmtOnly are
// pointers so "unset" is distinguishable from "false".
type ListInterfacesOptions struct {
	Name     string
	Device   string
	Type     string
	Enabled  *bool
	MgmtOnly *bool
	Limit    int
	Offset   int
	Extra    url.Values
}

// ListInterfaces returns one page of interfaces.
func (c *Client) ListInterfaces(ctx context.Context, opts ListInterfacesOptions) (Page[Interface], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	setIf(q, "name", opts.Name)
	setIf(q, "device", opts.Device)
	setIf(q, "type", opts.Type)
	if opts.Enabled != nil {
		q.Set("enabled", strconv.FormatBool(*opts.Enabled))
	}
	if opts.MgmtOnly != nil {
		q.Set("mgmt_only", strconv.FormatBool(*opts.MgmtOnly))
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[Interface]
	if err := c.Do(ctx, "GET", "/api/dcim/interfaces/", q, nil, &page); err != nil {
		return Page[Interface]{}, fmt.Errorf("dcim: list interfaces: %w", err)
	}
	return page, nil
}

// InterfacesFetcher returns a PageFetcher bound to opts.
func (c *Client) InterfacesFetcher(opts ListInterfacesOptions) PageFetcher[Interface] {
	return func(ctx context.Context, offset, limit int) (Page[Interface], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListInterfaces(ctx, opts)
	}
}
