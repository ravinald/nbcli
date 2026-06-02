// Package netbox · global-search support.
//
// Netbox v3.4+ exposes /api/search/ as a cross-resource full-text endpoint.
// One request returns hits from sites, devices, IPs, VRFs, etc., each tagged
// with the matched field and value. nbcli's `search all <key>` wires here;
// `search <module> <key>` reuses the per-resource `ListXxx` endpoints with
// Extra["q"]=key.
package netbox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// SearchResult is one hit from /api/search/. The shape is intentionally
// minimal — only the fields nbcli renders are typed. Object holds the full
// embedded payload as raw JSON so column extractors can pull `display`,
// `id`, `url` without us having to know every resource type at compile time.
type SearchResult struct {
	// Type is the dotted Netbox object type (e.g. "dcim.site",
	// "ipam.ipaddress", "virtualization.virtualmachine").
	Type string `json:"object_type"`

	// Object is the full embedded resource. Decode on demand into a small
	// shim struct ({Display, ID, URL}) — every Netbox object guarantees those.
	Object json.RawMessage `json:"object"`

	// Field is the attribute that matched the query (e.g. "name",
	// "description", "comments"). Tells the user *why* the row came back.
	Field string `json:"field"`

	// Value is the matched substring as returned by Netbox.
	Value string `json:"value"`

	// Attributes is per-type metadata Netbox attaches to a hit (e.g. parent,
	// scope). Kept as raw JSON — not rendered by default.
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

// SearchOptions filters /api/search/. Mirrors the ListXxxOptions shape so the
// rest of nbcli looks the same regardless of which endpoint backs a command.
type SearchOptions struct {
	// Q is the free-text query. Netbox sends it through as `?q=<value>`.
	Q string

	// Limit / Offset follow the standard v2 pagination scheme. Zero limit
	// uses Netbox's server default; convert to "all pages" at the caller via
	// SearchFetcher + netbox.ListAll.
	Limit  int
	Offset int

	// Extra is the catch-all for advanced filters (e.g. `object_types=dcim.site`).
	// Wins over Q if a key collides.
	Extra url.Values
}

// Search hits /api/search/ and returns one page of hits.
func (c *Client) Search(ctx context.Context, opts SearchOptions) (Page[SearchResult], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	if opts.Q != "" {
		q.Set("q", opts.Q)
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}

	var page Page[SearchResult]
	if err := c.Do(ctx, "GET", "/api/search/", q, nil, &page); err != nil {
		return Page[SearchResult]{}, fmt.Errorf("search: %w", err)
	}
	return page, nil
}

// SearchFetcher binds opts to a PageFetcher so `limit 0` streaming and
// netbox.ListAll work the same as for every other resource.
func (c *Client) SearchFetcher(opts SearchOptions) PageFetcher[SearchResult] {
	return func(ctx context.Context, offset, limit int) (Page[SearchResult], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.Search(ctx, opts)
	}
}
