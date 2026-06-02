// Package netbox · global-search support.
//
// Netbox has no /api/search/ endpoint (the web UI's search is Django-only).
// nbcli's `search all` reaches across resources two ways depending on what
// the operator has enabled:
//
//  1. GraphQL (preferred): one POST to /graphql/ that batches every
//     list_field. Lower latency, less server load, smaller payload.
//     NOTE: Netbox mounts GraphQL at /graphql/ (not /api/graphql/, which
//     is a common doc-induced mistake).
//  2. REST fan-out (fallback): 12 parallel ?q= calls to the typed list
//     endpoints. Used automatically when /graphql/ returns 404 — some
//     deployments disable GraphQL for security or operational reasons.
//
// The choice is per-Client and cached after the first GraphQL probe so
// subsequent searches in the same process skip the 404 round-trip.
//
// Per-module `search <module> <key>` always uses the typed REST endpoint
// directly; this file only matters for the `all` form.
package netbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// SearchBackend names a transport for `search all`. Empty string is
// equivalent to SearchAuto.
type SearchBackend string

// SearchBackend values.
const (
	// SearchAuto probes GraphQL and falls back to REST fan-out on 404.
	// Caches the decision on the Client. Default when SearchBackend is "".
	SearchAuto SearchBackend = "auto"

	// SearchGraphQL forces /graphql/. Surfaces any failure (no fallback).
	// Useful when the operator wants to be loud about GraphQL going down.
	SearchGraphQL SearchBackend = "graphql"

	// SearchREST forces parallel REST fan-out, skipping the GraphQL probe.
	// Useful when GraphQL is permanently disabled on the Netbox instance.
	SearchREST SearchBackend = "rest"
)

// normalizeSearchBackend validates the operator-supplied backend name and
// folds empty / "auto" into SearchAuto.
func normalizeSearchBackend(b SearchBackend) (SearchBackend, error) {
	switch SearchBackend(strings.ToLower(string(b))) {
	case "", SearchAuto:
		return SearchAuto, nil
	case SearchGraphQL:
		return SearchGraphQL, nil
	case SearchREST:
		return SearchREST, nil
	default:
		return "", fmt.Errorf("netbox: unknown SearchBackend %q (want %q, %q, or %q)",
			b, SearchAuto, SearchGraphQL, SearchREST)
	}
}

// SearchResult is one hit from the cross-resource global search.
//
// Type is the dotted Netbox object type (e.g. "dcim.site"). Object holds
// the rendered fields as JSON ({id, display, url}) so column extractors
// can decode the same shim regardless of which backend produced the hit.
// Field/Value are reserved for future use (Netbox's /api/search/ would
// have populated them); GraphQL and REST fan-out leave them empty.
type SearchResult struct {
	Type       string          `json:"object_type"`
	Object     json.RawMessage `json:"object"`
	Field      string          `json:"field,omitempty"`
	Value      string          `json:"value,omitempty"`
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

// SearchOptions filters a global search. Mirrors the ListXxxOptions shape
// so the rest of nbcli stays uniform regardless of which endpoint backs it.
type SearchOptions struct {
	Q      string     // free-text query
	Limit  int        // page size of the aggregated view
	Offset int        // offset into the aggregated view
	Extra  url.Values // forwarded to every REST fan-out request (ignored by GraphQL)
}

// SearchType describes one Netbox resource included in the global search.
// Dotted lands on SearchResult.Type. ListField is the GraphQL list field
// name (Netbox 4.x snake_case convention: `Model` → `model_list`). RESTPath
// is the typed list endpoint used both for synthesizing URLs in GraphQL
// hits and for the REST fan-out fallback.
type SearchType struct {
	Dotted    string
	ListField string
	RESTPath  string
}

// SearchTypes is the registry of resources `search all` covers. Exported so
// tests and operators can introspect the schema. Adding a row here extends
// both the GraphQL query (built at init from this list) and the REST
// fan-out target set.
var SearchTypes = []SearchType{
	{"dcim.site", "site_list", "/api/dcim/sites/"},
	{"dcim.rack", "rack_list", "/api/dcim/racks/"},
	{"dcim.device", "device_list", "/api/dcim/devices/"},
	{"dcim.interface", "interface_list", "/api/dcim/interfaces/"},
	{"ipam.prefix", "prefix_list", "/api/ipam/prefixes/"},
	{"ipam.ipaddress", "ip_address_list", "/api/ipam/ip-addresses/"},
	{"ipam.vlan", "vlan_list", "/api/ipam/vlans/"},
	{"ipam.vrf", "vrf_list", "/api/ipam/vrfs/"},
	{"tenancy.tenant", "tenant_list", "/api/tenancy/tenants/"},
	{"tenancy.contact", "contact_list", "/api/tenancy/contacts/"},
	{"virtualization.virtualmachine", "virtual_machine_list", "/api/virtualization/virtual-machines/"},
	{"virtualization.cluster", "cluster_list", "/api/virtualization/clusters/"},
}

// restFanoutCap bounds what one endpoint contributes to the aggregate when
// REST fan-out is in use. 100 keeps the fan-out latency reasonable while
// still letting a moderately wide search browse with the pager.
const restFanoutCap = 100

// Search dispatches a global search across every type in SearchTypes. The
// transport depends on Client.searchBackend:
//
//   - SearchREST: always REST fan-out (no GraphQL probe).
//   - SearchGraphQL: always GraphQL, propagates 404 as an error.
//   - SearchAuto (default): GraphQL with auto-fallback to REST on 404.
//     The decision is cached on the Client so subsequent calls skip the probe.
func (c *Client) Search(ctx context.Context, opts SearchOptions) (Page[SearchResult], error) {
	if opts.Q == "" {
		return Page[SearchResult]{}, nil
	}
	switch c.searchBackend {
	case SearchREST:
		return c.searchREST(ctx, opts)
	case SearchGraphQL:
		return c.searchGraphQL(ctx, opts)
	default: // SearchAuto
		if c.searchUsesREST.Load() {
			return c.searchREST(ctx, opts)
		}
		page, err := c.searchGraphQL(ctx, opts)
		if isGraphQLDisabled(err) {
			c.searchUsesREST.Store(true)
			slog.InfoContext(ctx, "netbox: GraphQL endpoint returned 404; falling back to REST fan-out for `search all` (set search_backend: rest in config.yaml to skip this probe)")
			return c.searchREST(ctx, opts)
		}
		return page, err
	}
}

// isGraphQLDisabled returns true if err is a 404 from the GraphQL endpoint —
// the signal that the operator has disabled GraphQL on this Netbox.
func isGraphQLDisabled(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// SearchFetcher binds opts to a PageFetcher so streaming and ListAll work the
// same as for every other resource. The backend (GraphQL vs REST fan-out) is
// chosen on each underlying Search call but cached on the Client after the
// first decision.
func (c *Client) SearchFetcher(opts SearchOptions) PageFetcher[SearchResult] {
	return func(ctx context.Context, offset, limit int) (Page[SearchResult], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.Search(ctx, opts)
	}
}

// --- GraphQL backend -------------------------------------------------------

// searchQuery is the static GraphQL document built once at init from
// SearchTypes. One document covers every resource; Netbox batches server-side.
var searchQuery = buildSearchQuery()

func buildSearchQuery() string {
	var b strings.Builder
	b.WriteString("query GlobalSearch($q: String!) {")
	for _, t := range SearchTypes {
		fmt.Fprintf(&b, " %s(filters: {q: $q}) { id display }", t.ListField)
	}
	b.WriteString(" }")
	return b.String()
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphqlError struct {
	Message string `json:"message"`
	Path    []any  `json:"path,omitempty"`
}

type graphqlResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []graphqlError             `json:"errors,omitempty"`
}

func (c *Client) searchGraphQL(ctx context.Context, opts SearchOptions) (Page[SearchResult], error) {
	body := graphqlRequest{
		Query:     searchQuery,
		Variables: map[string]any{"q": opts.Q},
	}
	var resp graphqlResponse
	if err := c.Do(ctx, "POST", "/graphql/", nil, body, &resp); err != nil {
		return Page[SearchResult]{}, err
	}
	if len(resp.Data) == 0 && len(resp.Errors) > 0 {
		msgs := make([]string, len(resp.Errors))
		for i, e := range resp.Errors {
			msgs[i] = e.Message
		}
		return Page[SearchResult]{}, fmt.Errorf("search graphql: %s", strings.Join(msgs, "; "))
	}

	all := make([]SearchResult, 0, 64)
	for _, t := range SearchTypes {
		raw, ok := resp.Data[t.ListField]
		if !ok || len(raw) == 0 {
			continue
		}
		var rows []map[string]any
		if err := json.Unmarshal(raw, &rows); err != nil {
			continue
		}
		for _, row := range rows {
			id, display := normalizeIDAndDisplay(row)
			obj, _ := json.Marshal(map[string]any{
				"id":      id,
				"display": display,
				"url":     t.RESTPath + strconv.Itoa(id) + "/",
			})
			all = append(all, SearchResult{Type: t.Dotted, Object: obj})
		}
	}
	return sliceAggregate(all, opts.Offset, opts.Limit), nil
}

// normalizeIDAndDisplay handles GraphQL ID coercion. The spec says IDs serialize
// as strings; some Netbox builds emit numbers. Accept both.
func normalizeIDAndDisplay(row map[string]any) (int, string) {
	var id int
	switch v := row["id"].(type) {
	case string:
		id, _ = strconv.Atoi(v)
	case float64:
		id = int(v)
	case int:
		id = v
	}
	display, _ := row["display"].(string)
	return id, display
}

// --- REST fan-out fallback -------------------------------------------------

// searchREST fans the query across every typed REST endpoint in SearchTypes
// in parallel. Used when GraphQL is disabled on the Netbox instance.
func (c *Client) searchREST(ctx context.Context, opts SearchOptions) (Page[SearchResult], error) {
	perLimit := opts.Offset + opts.Limit
	if perLimit <= 0 {
		perLimit = 50
	}
	if perLimit > restFanoutCap {
		perLimit = restFanoutCap
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
	return sliceAggregate(all, opts.Offset, opts.Limit), nil
}

// --- shared ----------------------------------------------------------------

// sliceAggregate applies offset/limit to a pre-aggregated result set.
// Count reflects the total before slicing so the pager shows accurate totals.
func sliceAggregate(all []SearchResult, offset, limit int) Page[SearchResult] {
	total := len(all)
	start := offset
	if start > total {
		start = total
	}
	end := total
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	return Page[SearchResult]{Count: total, Results: all[start:end]}
}
