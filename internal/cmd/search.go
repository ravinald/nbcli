// search.go implements `nbcli search [all|<module>] <key> [limit value] [pager]`.
//
// Two execution paths share a common shell:
//
//   - `search all <key>` hits Netbox's cross-resource /api/search/ endpoint
//     and renders heterogeneous results via the "search" column set.
//   - `search <module> <key>` reuses the per-resource List* endpoint with
//     Extra["q"]=<key>, rendering with that resource's existing columns
//     (same set the matching `nbcli show <module>` would produce).
//
// Trailing positional grammar is the same `limit value` and `pager` switch
// every show command supports — implemented by handing the trailing args
// to cmdutils.ParseShowArgs against a small whitelist.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
	"github.com/ravinald/nbcli/internal/pager"
)

// searchAllKeyword is the literal users type for the global cross-resource path.
const searchAllKeyword = "all"

// searchTrailingKeywords is the positional grammar allowed after `<module> <key>`.
// Mirrors the show commands' trailing slice: limit (with optional 0 for stream-all)
// and pager (switch). Offset is intentionally omitted — the pager handles browsing.
var searchTrailingKeywords = []cmdutils.KeywordSpec{
	{Name: cmdutils.LimitKeyword, Description: "page size (0 = stream all pages)", Example: "100"},
	cmdutils.PagerKeyword(),
}

// searchHandler runs one `search <module> ...` invocation. The signature
// matches every per-module helper below. kv carries the parsed trailing
// keywords (limit, pager); query is arg[1] from the command line.
type searchHandler func(cmd *cobra.Command, io IO, client *netbox.Client, query string, kv map[string]string) error

// searchHandlers is the dispatch table. Keys are user-typed module names
// (matching the Netbox API path segments and the columns registry keys),
// plus the literal "all" for the global endpoint.
//
// To add a new resource: add an entry here and a per-module helper below.
var searchHandlers = map[string]searchHandler{
	searchAllKeyword:   searchAll,
	"sites":            searchSites,
	"racks":            searchRacks,
	"devices":          searchDevices,
	"interfaces":       searchInterfaces,
	"prefixes":         searchPrefixes,
	"ip-addresses":     searchIPAddresses,
	"vlans":            searchVLANs,
	"vrfs":             searchVRFs,
	"tenants":          searchTenants,
	"contacts":         searchContacts,
	"virtual-machines": searchVMs,
	"clusters":         searchClusters,
}

// searchModuleNames returns the sorted list of acceptable arg[0] values.
// Used by the validator and completion to keep the menu consistent.
func searchModuleNames() []string {
	out := make([]string, 0, len(searchHandlers))
	for k := range searchHandlers {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// newSearchCmd is the top-level `nbcli search` factory. The cobra command
// only validates the grammar and dispatches; per-resource logic lives in
// the searchHandlers map.
func newSearchCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "search [all|<module>] <key> [limit <value>] [pager]",
		Short: "Free-text search across one resource or every resource",
		Long: "Search Netbox by free-text. Two shapes:\n\n" +
			"  nbcli search all <key>          # cross-resource (uses /api/search/)\n" +
			"  nbcli search <module> <key>     # one resource (uses ?q=<key>)\n\n" +
			"Modules: " + strings.Join(searchModuleNames(), ", ") + "\n\n" +
			"Trailing positional grammar:\n" +
			cmdutils.HelpTable(searchTrailingKeywords) +
			"\nExamples:\n" +
			"  nbcli search sites hq\n" +
			"  nbcli search ip-addresses 10.0.0\n" +
			"  nbcli search vrfs prod limit 200\n" +
			"  nbcli search all hq pager\n" +
			"  nbcli search all foo limit 0 --format json | jq .\n",
		Args:              validateSearchArgs,
		ValidArgsFunction: searchCompletionFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validator guarantees arg[0] is a valid key and arg[1] exists.
			handler := searchHandlers[args[0]]
			query := args[1]
			kv, err := cmdutils.ParseShowArgs(args[2:], searchTrailingKeywords)
			if err != nil {
				return err
			}
			client, err := clientFromCtx(cmd)
			if err != nil {
				return err
			}
			return handler(cmd, io, client, query, kv)
		},
	}
}

// validateSearchArgs is the Args func enforcing:
//
//	search <all|module> <key> [limit <value>] [pager]
//
// Errors point at the exact problem so the user knows whether they
// mistyped the module, forgot the key, or used an unknown trailing keyword.
func validateSearchArgs(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("expected: search [all|<module>] <key> [limit <value>] [pager]")
	}
	if _, ok := searchHandlers[args[0]]; !ok {
		return fmt.Errorf("unknown module %q (expected: %s)", args[0], strings.Join(searchModuleNames(), ", "))
	}
	if len(args) < 2 {
		return fmt.Errorf("search %s: requires a query key (e.g. `nbcli search %s hq`)", args[0], args[0])
	}
	if _, err := cmdutils.ParseShowArgs(args[2:], searchTrailingKeywords); err != nil {
		return err
	}
	return nil
}

// searchCompletionFunc drives shell completion across the three positions:
//
//	arg[0] → modules (all + every registered resource)
//	arg[1] → free-form key, no completion
//	arg[2+]→ trailing keyword grammar (limit / pager)
func searchCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch {
	case len(args) == 0:
		return searchModuleNames(), cobra.ShellCompDirectiveNoFileComp
	case len(args) == 1:
		return nil, cobra.ShellCompDirectiveNoFileComp
	default:
		return cmdutils.CompletionFunc(searchTrailingKeywords)(cmd, args[2:], toComplete)
	}
}

// runSearch is the common postlude shared by every per-module handler.
// Picks between pager / streaming / one-shot rendering based on kv and the
// resolved format. Mirrors the show commands' tail block so a search
// invocation behaves identically to the analogous show invocation.
func runSearch[T any](
	cmd *cobra.Command,
	io IO,
	title string,
	cols []output.Column,
	kv map[string]string,
	fetchAll bool,
	oneShot func(ctx context.Context) (netbox.Page[T], error),
	stream netbox.PageFetcher[T],
	pagerFetch func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error),
) error {
	if kv["pager"] == "true" {
		return runPager(io, title, cols, pagerFetch)
	}
	if fetchAll {
		return renderStreaming[T](cmd, io, stream, netbox.IterateOptions{PageSize: 100, MaxPages: 200}, cols)
	}
	page, err := oneShot(cmd.Context())
	if err != nil {
		return err
	}
	return renderRows(cmd, io, page.Results, cols)
}

// qExtra builds the url.Values containing only ?q=<query>. Helper so each
// per-module handler stays a single Options literal.
func qExtra(query string) url.Values { return url.Values{"q": {query}} }

// pagerExtraForQuery returns Extra values for one pager iteration. If the
// pager committed a search (po.Query != ""), it overrides the CLI's original
// key; otherwise the original key persists across paging.
func pagerExtraForQuery(cliQuery string, po pager.FetchOpts) url.Values {
	if po.Query != "" {
		return qExtra(po.Query)
	}
	return qExtra(cliQuery)
}

// searchTitle is the title shown at the top of the pager.
func searchTitle(module, query string) string {
	return fmt.Sprintf("Search · %s · %q", module, query)
}

// --- Per-module handlers --------------------------------------------------
//
// Each one is the same shape: build typed Options with Extra=?q=<key>, apply
// kv limit/offset, resolve columns, and pass three typed closures to
// runSearch[T] for the pager / stream / one-shot paths. T differs per
// handler so the Page[T] and PageFetcher[T] stay strongly typed end-to-end.

func searchAll(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.SearchOptions{Q: query, Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "search")
	return runSearch(cmd, io, searchTitle(searchAllKeyword, query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.SearchResult], error) { return c.Search(ctx, opts) },
		c.SearchFetcher(netbox.SearchOptions{Q: query}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.SearchOptions{Offset: po.Offset, Limit: po.Limit}
			if po.Query != "" {
				o.Q = po.Query
			} else {
				o.Q = query
			}
			p, err := c.Search(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchSites(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListSitesOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "sites")
	return runSearch(cmd, io, searchTitle("sites", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.Site], error) { return c.ListSites(ctx, opts) },
		c.SitesFetcher(netbox.ListSitesOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListSitesOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListSites(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchRacks(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListRacksOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "racks")
	return runSearch(cmd, io, searchTitle("racks", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.Rack], error) { return c.ListRacks(ctx, opts) },
		c.RacksFetcher(netbox.ListRacksOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListRacksOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListRacks(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchDevices(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListDevicesOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "devices")
	return runSearch(cmd, io, searchTitle("devices", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.Device], error) { return c.ListDevices(ctx, opts) },
		c.DevicesFetcher(netbox.ListDevicesOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListDevicesOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListDevices(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchInterfaces(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListInterfacesOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "interfaces")
	return runSearch(cmd, io, searchTitle("interfaces", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.Interface], error) { return c.ListInterfaces(ctx, opts) },
		c.InterfacesFetcher(netbox.ListInterfacesOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListInterfacesOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListInterfaces(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchPrefixes(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListPrefixesOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "prefixes")
	return runSearch(cmd, io, searchTitle("prefixes", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.Prefix], error) { return c.ListPrefixes(ctx, opts) },
		c.PrefixesFetcher(netbox.ListPrefixesOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListPrefixesOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListPrefixes(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchIPAddresses(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListIPAddressesOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "ip-addresses")
	return runSearch(cmd, io, searchTitle("ip-addresses", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.IPAddress], error) { return c.ListIPAddresses(ctx, opts) },
		c.IPAddressesFetcher(netbox.ListIPAddressesOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListIPAddressesOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListIPAddresses(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchVLANs(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListVLANsOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "vlans")
	return runSearch(cmd, io, searchTitle("vlans", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.VLAN], error) { return c.ListVLANs(ctx, opts) },
		c.VLANsFetcher(netbox.ListVLANsOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListVLANsOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListVLANs(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchVRFs(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListVRFsOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "vrfs")
	return runSearch(cmd, io, searchTitle("vrfs", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.VRF], error) { return c.ListVRFs(ctx, opts) },
		c.VRFsFetcher(netbox.ListVRFsOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListVRFsOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListVRFs(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchTenants(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListTenantsOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "tenants")
	return runSearch(cmd, io, searchTitle("tenants", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.Tenant], error) { return c.ListTenants(ctx, opts) },
		c.TenantsFetcher(netbox.ListTenantsOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListTenantsOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListTenants(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchContacts(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListContactsOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "contacts")
	return runSearch(cmd, io, searchTitle("contacts", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.Contact], error) { return c.ListContacts(ctx, opts) },
		c.ContactsFetcher(netbox.ListContactsOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListContactsOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListContacts(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchVMs(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListVMsOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "virtual-machines")
	return runSearch(cmd, io, searchTitle("virtual-machines", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.VirtualMachine], error) { return c.ListVMs(ctx, opts) },
		c.VMsFetcher(netbox.ListVMsOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListVMsOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListVMs(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}

func searchClusters(cmd *cobra.Command, io IO, c *netbox.Client, query string, kv map[string]string) error {
	opts := netbox.ListClustersOptions{Extra: qExtra(query), Limit: 50}
	fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
	if err != nil {
		return err
	}
	cols := resolveColumns(cmd, "clusters")
	return runSearch(cmd, io, searchTitle("clusters", query), cols, kv, fetchAll,
		func(ctx context.Context) (netbox.Page[netbox.Cluster], error) { return c.ListClusters(ctx, opts) },
		c.ClustersFetcher(netbox.ListClustersOptions{Extra: qExtra(query)}),
		func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
			o := netbox.ListClustersOptions{Extra: pagerExtraForQuery(query, po), Offset: po.Offset, Limit: po.Limit}
			p, err := c.ListClusters(ctx, o)
			if err != nil {
				return pager.FetchResult{}, err
			}
			return pager.FetchResult{Rows: p.Results, Total: p.Count}, nil
		})
}
