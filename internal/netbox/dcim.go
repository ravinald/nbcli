package netbox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// NestedRef is the inline {id, url, display, slug, name} object Netbox returns
// for foreign-key references. Several fields are optional depending on the
// related resource — we keep them as omitempty pointers/strings as needed.
type NestedRef struct {
	ID      int    `json:"id"`
	URL     string `json:"url,omitempty"`
	Display string `json:"display,omitempty"`
	Name    string `json:"name,omitempty"`
	Slug    string `json:"slug,omitempty"`
}

// Site is the DCIM site resource (subset; expanded as commands need fields).
type Site struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	Display     string     `json:"display"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Status      LabelValue `json:"status"`
	Region      *NestedRef `json:"region,omitempty"`
	Tenant      *NestedRef `json:"tenant,omitempty"`
	Facility    string     `json:"facility,omitempty"`
	Description string     `json:"description,omitempty"`
	Comments    string     `json:"comments,omitempty"`
	Created     string     `json:"created,omitempty"`
	LastUpdated string     `json:"last_updated,omitempty"`
}

// LabelValue is Netbox's enum shape: {value, label}.
//
// Netbox is inconsistent about whether "value" is a string or a number:
// status/type enums use strings ("active", "10gbase-x-sfpp"), while IPAM
// family uses an int (4, 6). LabelValue normalizes both into a single
// string field via a custom unmarshaler so callers always read
// `lv.Value` as a string regardless of how the API delivered it.
type LabelValue struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// UnmarshalJSON accepts either {"value": "x", ...} or {"value": 4, ...}
// and stuffs the result into Value as a string ("x" or "4").
func (lv *LabelValue) UnmarshalJSON(b []byte) error {
	var raw struct {
		Value json.RawMessage `json:"value"`
		Label string          `json:"label"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	lv.Label = raw.Label
	switch {
	case len(raw.Value) == 0, string(raw.Value) == "null":
		lv.Value = ""
	case raw.Value[0] == '"':
		if err := json.Unmarshal(raw.Value, &lv.Value); err != nil {
			return err
		}
	default:
		// Number, bool, or anything else: store the raw token. Trim outer
		// whitespace so {"value":  4} → "4" rather than "  4".
		lv.Value = strings.TrimSpace(string(raw.Value))
	}
	return nil
}

// ListSitesOptions filters the DCIM /sites/ list endpoint.
// Only the most common filters are wired here; raw extra filters can be
// supplied via Extra.
type ListSitesOptions struct {
	Name   string
	Slug   string
	Status string
	Region string
	Tenant string
	Limit  int
	Offset int
	Extra  url.Values
}

// ListSites returns one page of sites. Pagination follows the standard
// Netbox v2 limit/offset scheme; the caller iterates by bumping Offset until
// Page.Next is nil.
func (c *Client) ListSites(ctx context.Context, opts ListSitesOptions) (Page[Site], error) {
	q := url.Values{}
	if opts.Extra != nil {
		for k, v := range opts.Extra {
			q[k] = v
		}
	}
	if opts.Name != "" {
		q.Set("name", opts.Name)
	}
	if opts.Slug != "" {
		q.Set("slug", opts.Slug)
	}
	if opts.Status != "" {
		q.Set("status", opts.Status)
	}
	if opts.Region != "" {
		q.Set("region", opts.Region)
	}
	if opts.Tenant != "" {
		q.Set("tenant", opts.Tenant)
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}

	var page Page[Site]
	if err := c.Do(ctx, "GET", "/api/dcim/sites/", q, nil, &page); err != nil {
		return Page[Site]{}, fmt.Errorf("dcim: list sites: %w", err)
	}
	return page, nil
}

// SitesFetcher returns a PageFetcher bound to opts, mirroring the convention
// every other DCIM/IPAM/Virtualization resource exposes.
func (c *Client) SitesFetcher(opts ListSitesOptions) PageFetcher[Site] {
	return func(ctx context.Context, offset, limit int) (Page[Site], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListSites(ctx, opts)
	}
}
