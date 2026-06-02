package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/pager"
)

var prefixKeywords = append([]cmdutils.KeywordSpec{
	{Name: "prefix", Description: "exact CIDR (e.g. 10.0.0.0/24)"},
	{Name: "contains", Description: "containing CIDR/IP"},
	{Name: "vrf", Description: "VRF RD"},
	{Name: "family", Description: "4 or 6", Values: []string{"4", "6"}},
	{Name: "status", Description: "status value", Example: "active",
		Values: []string{"container", "active", "reserved", "deprecated"}},
	{Name: "role", Description: "prefix role slug"},
	{Name: "site", Description: "site slug"},
	{Name: "tenant", Description: "tenant slug"},
}, cmdutils.PaginationKeywords()...)

func newShowPrefixesCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "prefixes " + cmdutils.UsageLine(prefixKeywords),
		Short: "List IPAM prefixes",
		Long: "List IPAM prefixes. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(prefixKeywords) +
			"\nExamples:\n" +
			"  nbcli show prefixes site hq family 4\n" +
			"  nbcli show prefixes contains 10.0.0.5\n",
		Args:              cmdutils.Validator(prefixKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(prefixKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, prefixKeywords)
			opts := netbox.ListPrefixesOptions{
				Prefix:   kv["prefix"],
				Contains: kv["contains"],
				VRF:      kv["vrf"],
				Family:   kv["family"],
				Status:   kv["status"],
				Role:     kv["role"],
				Site:     kv["site"],
				Tenant:   kv["tenant"],
				Limit:    50,
			}
			fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
			if err != nil {
				return err
			}
			client, err := clientFromCtx(cmd)
			if err != nil {
				return err
			}
			cols := resolveColumns(cmd, "prefixes")

			if interactiveFlag(cmd) {
				return runPager(io, "Prefixes", cols, func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
					listOpts := opts
					listOpts.Offset, listOpts.Limit = po.Offset, po.Limit
					applyPagerQuery(&listOpts.Extra, po.Query)
					page, err := client.ListPrefixes(ctx, listOpts)
					if err != nil {
						return pager.FetchResult{}, err
					}
					return pager.FetchResult{Rows: page.Results, Total: page.Count}, nil
				})
			}

			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}
			if fetchAll {
				return renderStreaming[netbox.Prefix](cmd, io, client.PrefixesFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListPrefixes(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
