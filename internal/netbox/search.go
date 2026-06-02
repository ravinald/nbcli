// Package netbox · global-search support.
//
// Netbox has no server-side cross-resource free-text search exposed via the
// REST or GraphQL APIs. `/api/search/` doesn't exist; GraphQL's auto-generated
// filter types (SiteFilter, DeviceFilter, ...) don't expose `q` — only typed
// per-field operators (`i_contains`, `exact`, ...). The web UI's global search
// runs through Django internals not surfaced to either API.
//
// nbcli's `search all` therefore fans out across every typed REST list
// endpoint in parallel with `?q=<key>`. Tradeoffs:
//
//   - 12 concurrent requests per invocation.
//   - The aggregate is bounded by perEndpointCap (default 100 rows per
//     endpoint). Browsing past that point: `nbcli search <module> <key>`.
//   - We don't know which field matched on each hit (Netbox's typed
//     endpoints don't tell us); SearchResult.Field / Value stay empty.
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

// SearchResult is one hit from the cross-resource fan-out.
//
// Type is the dotted Netbox object type (e.g. "dcim.site"). Object holds the
// resource row as raw JSON; column extractors decode the {id, display, url}
// shim on demand so new Netbox object types render with no code change.
// Field/Value are reserved for the day Netbox exposes a /api/search/-style
// endpoint that tells us which attribute matched; today they stay empty.
type SearchResult struct {
	Type       string          `json:"object_type"`
	Object     json.RawMessage `json:"object"`
	Field      string          `json:"field,omitempty"`
	Value      string          `json:"value,omitempty"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

// SearchOptions filters a global search. Mirrors the ListXxxOptions shape so
// the rest of nbcli stays uniform regardless of which endpoint backs it.
type SearchOptions struct {
	Q      string     // free-text query, applied as ?q= on every fan-out request
	Limit  int        // page size of the aggregated view
	Offset int        // offset into the aggregated view
	Extra  url.Values // forwarded to every fan-out request
}

// SearchType maps one Netbox resource into the fan-out. Dotted lands on
// SearchResult.Type. RESTPath is the typed list endpoint hit with `?q=`.
type SearchType struct {
	Dotted   string
	RESTPath string
}

// SearchTypes is the resource set `search all` covers. Exported so tests and
// callers can introspect the schema. Order is stable so result ordering across
// runs is reproducible.
var SearchTypes = []SearchType{
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

// perEndpointCap bounds what one endpoint contributes to the aggregate. 100
// keeps fan-out latency reasonable while letting a moderately wide search
// browse with the pager.
const perEndpointCap = 100

// Search fans the query across every typed endpoint in SearchTypes in
// parallel and returns one merged page. Offset and Limit slice into the
// aggregate. Count reflects the merged total before slicing so the pager
// can show "page N of M" accurately.
//
// Partial failures are silent: if `device_list` 500s but the rest succeed,
// the user gets the survivors. Returns an error only when every endpoint
// fails; the joined error names the failing types.
func (c *Client) Search(ctx context.Context, opts SearchOptions) (Page[SearchResult], error) {
	if opts.Q == "" {
		return Page[SearchResult]{}, nil
	}
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
		Err     error
	}
	out := make(chan fanResult, len(SearchTypes))
	var wg sync.WaitGroup
	for _, t := range SearchTypes {
		wg.Add(1)
		go func(t SearchType) {
			defer wg.Done()
			q := url.Values{}
			for k, v := range opts.Extra {
				q[k] = v
			}
			q.Set("q", opts.Q)
			q.Set("limit", strconv.Itoa(perLimit))

			var page Page[json.RawMessage]
			if err := c.Do(ctx, "GET", t.RESTPath, q, nil, &page); err != nil {
				out <- fanResult{Type: t.Dotted, Err: fmt.Errorf("%s: %w", t.Dotted, err)}
				return
			}
			results := make([]SearchResult, len(page.Results))
			for i, raw := range page.Results {
				results[i] = SearchResult{Type: t.Dotted, Object: raw}
			}
			out <- fanResult{Type: t.Dotted, Results: results}
		}(t)
	}
	wg.Wait()
	close(out)

	all := make([]SearchResult, 0, len(SearchTypes)*perLimit)
	var errs []error
	for r := range out {
		if r.Err != nil {
			errs = append(errs, r.Err)
			continue
		}
		all = append(all, r.Results...)
	}
	if len(all) == 0 && len(errs) > 0 {
		return Page[SearchResult]{}, errors.Join(errs...)
	}

	total := len(all)
	start := opts.Offset
	if start > total {
		start = total
	}
	end := total
	if opts.Limit > 0 && start+opts.Limit < end {
		end = start + opts.Limit
	}
	return Page[SearchResult]{Count: total, Results: all[start:end]}, nil
}

// SearchFetcher binds opts to a PageFetcher so streaming and ListAll work the
// same as for every other resource. Each PageFetcher call re-runs the full
// fan-out — no cross-call caching — so `search all <key> limit 0` re-fetches
// per page. Cheap for browsing, expensive for streaming the entire aggregate;
// for that, prefer `nbcli search <module> <key> limit 0`.
func (c *Client) SearchFetcher(opts SearchOptions) PageFetcher[SearchResult] {
	return func(ctx context.Context, offset, limit int) (Page[SearchResult], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.Search(ctx, opts)
	}
}
