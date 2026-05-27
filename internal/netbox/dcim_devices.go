package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// DeviceType is the inline /dcim/device-types/ object Netbox embeds on Device.
type DeviceType struct {
	ID           int        `json:"id"`
	URL          string     `json:"url,omitempty"`
	Display      string     `json:"display"`
	Manufacturer *NestedRef `json:"manufacturer,omitempty"`
	Model        string     `json:"model"`
	Slug         string     `json:"slug,omitempty"`
}

// Device is the /dcim/devices/ resource (subset).
type Device struct {
	ID          int         `json:"id"`
	URL         string      `json:"url"`
	Display     string      `json:"display"`
	Name        string      `json:"name"`
	DeviceType  *DeviceType `json:"device_type,omitempty"`
	Role        *NestedRef  `json:"role,omitempty"`
	Tenant      *NestedRef  `json:"tenant,omitempty"`
	Platform    *NestedRef  `json:"platform,omitempty"`
	Serial      string      `json:"serial,omitempty"`
	AssetTag    string      `json:"asset_tag,omitempty"`
	Site        *NestedRef  `json:"site,omitempty"`
	Location    *NestedRef  `json:"location,omitempty"`
	Rack        *NestedRef  `json:"rack,omitempty"`
	Position    *float64    `json:"position,omitempty"`
	Face        *LabelValue `json:"face,omitempty"`
	Status      LabelValue  `json:"status"`
	PrimaryIP4  *NestedRef  `json:"primary_ip4,omitempty"`
	PrimaryIP6  *NestedRef  `json:"primary_ip6,omitempty"`
	Description string      `json:"description,omitempty"`
}

// ListDevicesOptions filters /dcim/devices/. Only the ten most common filters
// are wired; everything else flows through Extra.
type ListDevicesOptions struct {
	Name         string
	Role         string
	Site         string
	Rack         string
	Status       string
	Tenant       string
	Manufacturer string
	Model        string
	Location     string
	Tag          string
	Limit        int
	Offset       int
	Extra        url.Values
}

// ListDevices returns one page of devices.
func (c *Client) ListDevices(ctx context.Context, opts ListDevicesOptions) (Page[Device], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	setIf(q, "name", opts.Name)
	setIf(q, "role", opts.Role)
	setIf(q, "site", opts.Site)
	setIf(q, "rack", opts.Rack)
	setIf(q, "status", opts.Status)
	setIf(q, "tenant", opts.Tenant)
	setIf(q, "manufacturer", opts.Manufacturer)
	setIf(q, "device_type", opts.Model)
	setIf(q, "location", opts.Location)
	setIf(q, "tag", opts.Tag)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[Device]
	if err := c.Do(ctx, "GET", "/api/dcim/devices/", q, nil, &page); err != nil {
		return Page[Device]{}, fmt.Errorf("dcim: list devices: %w", err)
	}
	return page, nil
}

// DevicesFetcher returns a PageFetcher bound to opts.
func (c *Client) DevicesFetcher(opts ListDevicesOptions) PageFetcher[Device] {
	return func(ctx context.Context, offset, limit int) (Page[Device], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListDevices(ctx, opts)
	}
}
