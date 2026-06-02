// Package netbox · global-search support via GraphQL.
//
// Netbox has no server-side cross-resource search exposed via REST (the
// /api/search/ path is a 404 — the web UI's search is Django-only). Netbox
// 4.x exposes a GraphQL endpoint at /api/graphql/ that natively supports
// batched cross-resource queries, which is the right tool for `search all`:
//
//   - One HTTP request instead of 12 parallel REST fan-outs.
//   - Server-side batching; lower latency on high-latency Netbox connections.
//   - Smaller payload — we ask for only the fields we render.
//
// `nbcli search <module> <key>` still routes through the typed ListXxx
// REST methods with Extra["q"]. Only `nbcli search all <key>` uses GraphQL.
package netbox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// SearchResult is one hit from the cross-resource global search.
//
// Type is the dotted Netbox object type (e.g. "dcim.site"). Object holds
// the rendered fields as JSON ({id, display, url}) so column extractors
// can decode the same shim regardless of which Netbox type backed the hit.
// Field/Value carry the matched-attribute pair Netbox returns from its
// /api/search/ endpoint when available; GraphQL doesn't surface those
// today, so for v1 they stay empty for fan-out-style results.
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
	Extra  url.Values // reserved for future per-call extensions
}

// SearchType describes one Netbox resource included in the global GraphQL
// query. Dotted lands on SearchResult.Type. ListField is the GraphQL list
// query field name (Netbox 4.x snake_case convention: `Model` → `model_list`).
// RESTPath is the REST path prefix the client synthesizes the URL with,
// since GraphQL doesn't expose object URLs directly.
type SearchType struct {
	Dotted    string
	ListField string
	RESTPath  string
}

// SearchTypes is the registry of resources `search all` covers. Exported so
// tests and callers can introspect the schema (e.g. to confirm a new resource
// is wired in). Order is stable so result ordering across runs is reproducible.
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

// searchQuery is the static GraphQL document built once at init from
// SearchTypes. One document covers every resource; Netbox batches the work
// server-side and returns a single response. Adding a new resource to
// SearchTypes automatically extends the query.
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

// graphqlRequest is the standard GraphQL POST payload.
type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// graphqlError is one entry from the GraphQL `errors` array.
type graphqlError struct {
	Message string `json:"message"`
	Path    []any  `json:"path,omitempty"`
}

// graphqlResponse is what Netbox returns from /api/graphql/. Data is keyed
// by the field name we requested (e.g. "device_list"); each value is the
// list of objects for that type as raw JSON.
type graphqlResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []graphqlError             `json:"errors,omitempty"`
}

// Search runs the cross-resource global search via /api/graphql/ and returns
// one merged page. Offset and Limit slice the aggregated view client-side.
// Count is the sum of returned-row counts across types (not Netbox's true
// per-type totals — GraphQL doesn't expose those without a separate query).
//
// Partial type errors are silently dropped — if `device_list` 500s but the
// rest succeed, the user gets the survivors. An error is returned only when
// the GraphQL response has no `data` at all.
func (c *Client) Search(ctx context.Context, opts SearchOptions) (Page[SearchResult], error) {
	if opts.Q == "" {
		return Page[SearchResult]{}, nil
	}
	body := graphqlRequest{
		Query:     searchQuery,
		Variables: map[string]any{"q": opts.Q},
	}
	var resp graphqlResponse
	if err := c.Do(ctx, "POST", "/api/graphql/", nil, body, &resp); err != nil {
		return Page[SearchResult]{}, fmt.Errorf("search graphql: %w", err)
	}

	// All-types-failed: surface the joined error messages so the user sees
	// what GraphQL is complaining about (schema drift, permission errors, ...).
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
			// Schema drift on one type shouldn't kill the whole search.
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

	totalAcross := len(all)
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

// normalizeIDAndDisplay pulls the canonical id (int) and display (string)
// out of a GraphQL row. GraphQL spec serializes IDs as strings; some Netbox
// builds emit numbers. Accept both.
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

// SearchFetcher binds opts to a PageFetcher so streaming and ListAll work
// the same as for every other resource. Note: streaming a global search
// re-runs the whole GraphQL query per page (no cross-call caching), so
// `limit 0` on `search all` re-fetches every iteration — usually cheap
// since the response is one request, but worth knowing.
func (c *Client) SearchFetcher(opts SearchOptions) PageFetcher[SearchResult] {
	return func(ctx context.Context, offset, limit int) (Page[SearchResult], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.Search(ctx, opts)
	}
}
