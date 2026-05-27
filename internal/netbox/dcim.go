package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
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
type LabelValue struct {
	Value string `json:"value"`
	Label string `json:"label"`
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
