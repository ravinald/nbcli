// Package netbox · global-search support.
//
// Netbox has NO server-side cross-resource search API — /api/search/ is a
// 404 (the web UI's search bar is Django-side only). nbcli's `search all`
// emulates it by fanning out across the typed list endpoints with ?q=<key>
// in parallel and merging the hits into a uniform SearchResult stream.
//
// Tradeoffs that come with fan-out:
//
//   - 12 concurrent requests per `search all` invocation.
//   - The aggregate is bounded by perEndpointCap (default 100 rows per
//     endpoint). Browsing past that point requires `nbcli search <module>`.
//   - We don't know *which field* matched on each hit (Netbox's typed
//     endpoints don't tell us); SearchResult.Field / Value are empty for
//     fan-out hits.
//
// `nbcli search <module> <key>` still goes through the typed ListXxx
// methods directly with Extra["q"] — only the `all` form fans out.
package netbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
)

// SearchResult is one hit from a global search fan-out.
//
// Type carries the dotted Netbox object type (e.g. "dcim.site"). Object is
// the full row as raw JSON so column extractors decode `{id, display, url}`
// on demand. Field/Value are reserved for the original API shape (matched
// field + matched value); the fan-out path leaves them empty because the
// typed endpoints don't surface them.
type SearchResult struct {
	Type       string          `json:"object_type"`
	Object     json.RawMessage `json:"object"`
	Field      string          `json:"field,omitempty"`
	Value      string          `json:"value,omitempty"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

// SearchOptions filters the global search. Mirrors the ListXxxOptions shape
// so the rest of nbcli looks the same regardless of which endpoint backs it.
type SearchOptions struct {
	Q      string     // free-text query
	Limit  int        // page size of the aggregated view
	Offset int        // offset into the aggregated view
	Extra  url.Values // forwarded to every fan-out request
}

// searchEndpoint maps a Netbox typed list endpoint to its dotted object type.
type searchEndpoint struct {
	Type string
	Path string
}

// SearchEndpoints is the fan-out target set. Exported so tests and callers
// can introspect or override (e.g. drop modules a user lacks permission for).
// Order is stable so result ordering across runs is reproducible.
var SearchEndpoints = []searchEndpoint{
	{"dcim.site", "/api/dcim/sites/"},
	{"dcim.rack", "/api/dcim/racks/"},
	{"dcim.device", "/api/dcim/devices/"},
	{"dcim.interface", "/api/dcim/interfaces/"},
	{"ipam.prefix", "/api/ipam/prefixes/"},
	{"ipam.ipaddress", "/api/ipam/ip-addresses/"},
	{"ipam.vlan", "/api/ipam/vlans/"},
	{"ipam.vrf", "/api/ipam/vrfs/"},
	{"tenancy.tenant", "/api/tenancy/tenants/"},
	{"tenancy.contact", "/api/tenancy/contacts/"},
	{"virtualization.virtualmachine", "/api/virtualization/virtual-machines/"},
	{"virtualization.cluster", "/api/virtualization/clusters/"},
}

// perEndpointCap bounds what one endpoint contributes to the aggregate.
// 100 keeps the fan-out latency reasonable while still letting a moderately
// wide search browse with the pager.
const perEndpointCap = 100

// Search fans out the query across every typed endpoint in SearchEndpoints
// and returns one merged page. Offset and Limit slice into the aggregate.
// Count is the SUM of per-endpoint Counts (a true global total, even when
// fewer rows are returned because of perEndpointCap).
//
// Partial failures: if any endpoint succeeds, returns its rows and silently
// drops the failed ones. Returns an error only when every endpoint fails —
// then the joined error explains what went wrong on each.
func (c *Client) Search(ctx context.Context, opts SearchOptions) (Page[SearchResult], error) {
	perLimit := opts.Offset + opts.Limit
	if perLimit <= 0 {
		perLimit = 50
	}
	if perLimit > perEndpointCap {
		perLimit = perEndpointCap
	}

	type fanResult struct {
		Type    string
		Results []SearchResult
		Total   int
		Err     error
	}
	out := make(chan fanResult, len(SearchEndpoints))
	var wg sync.WaitGroup
	for _, ep := range SearchEndpoints {
		wg.Add(1)
		go func(ep searchEndpoint) {
			defer wg.Done()
			q := url.Values{}
			for k, v := range opts.Extra {
				q[k] = v
			}
			q.Set("q", opts.Q)
			q.Set("limit", strconv.Itoa(perLimit))

			var page Page[json.RawMessage]
			if err := c.Do(ctx, "GET", ep.Path, q, nil, &page); err != nil {
				out <- fanResult{Type: ep.Type, Err: fmt.Errorf("%s: %w", ep.Type, err)}
				return
			}
			results := make([]SearchResult, len(page.Results))
			for i, raw := range page.Results {
				results[i] = SearchResult{Type: ep.Type, Object: raw}
			}
			out <- fanResult{Type: ep.Type, Results: results, Total: page.Count}
		}(ep)
	}
	wg.Wait()
	close(out)

	all := make([]SearchResult, 0, len(SearchEndpoints)*perLimit)
	var totalAcross int
	var errs []error
	for r := range out {
		if r.Err != nil {
			errs = append(errs, r.Err)
			continue
		}
		all = append(all, r.Results...)
		totalAcross += r.Total
	}
	if len(all) == 0 && len(errs) > 0 {
		return Page[SearchResult]{}, errors.Join(errs...)
	}

	// Slice the aggregated view by offset/limit. Past the end is empty,
	// not an error.
	start := opts.Offset
	if start > len(all) {
		start = len(all)
	}
	end := len(all)
	if opts.Limit > 0 && start+opts.Limit < end {
		end = start + opts.Limit
	}
	return Page[SearchResult]{Count: totalAcross, Results: all[start:end]}, nil
}

// SearchFetcher binds opts to a PageFetcher so streaming and ListAll work the
// same as for every other resource. NOTE: streaming a fan-out re-runs the
// whole fan-out per page (no cross-call caching), so `limit 0` on `search all`
// is expensive — prefer `nbcli search <module> <key> limit 0` for streams.
func (c *Client) SearchFetcher(opts SearchOptions) PageFetcher[SearchResult] {
	return func(ctx context.Context, offset, limit int) (Page[SearchResult], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.Search(ctx, opts)
	}
}
