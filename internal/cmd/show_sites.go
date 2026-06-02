package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/pager"
)

// siteKeywords is the positional keyword set for `nbcli show sites`.
// Filters interact with the Netbox API → positional. Operational concerns
// (--format, --url, ...) stay as flags on root.
var siteKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact site name"},
	{Name: "slug", Description: "site slug"},
	{Name: "status", Description: "status value", Example: "active",
		Values: []string{"active", "planned", "staging", "decommissioning", "retired"}},
	{Name: "region", Description: "region slug"},
	{Name: "tenant", Description: "tenant slug"},
}, cmdutils.PaginationKeywords()...)

// newShowSitesCmd is the reference command shape. Every show subcommand
// follows the same flow: parse positional keywords → typed Options → either
// stream all pages (limit 0, streaming-friendly format) or fetch one page.
func newShowSitesCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "sites " + cmdutils.UsageLine(siteKeywords),
		Short: "List DCIM sites",
		Long: "List DCIM sites. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(siteKeywords) +
			"\nExamples:\n" +
			"  nbcli show sites\n" +
			"  nbcli show sites status active\n" +
			"  nbcli show sites region us-west status active limit 100\n" +
			"  nbcli show sites limit 0 --format json   # streams full inventory\n",
		Args:              cmdutils.Validator(siteKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(siteKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, siteKeywords)
			opts := netbox.ListSitesOptions{
				Name:   kv["name"],
				Slug:   kv["slug"],
				Status: kv["status"],
				Region: kv["region"],
				Tenant: kv["tenant"],
				Limit:  50,
			}
			fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
			if err != nil {
				return err
			}
			client, err := clientFromCtx(cmd)
			if err != nil {
				return err
			}

			cols := resolveColumns(cmd, "sites")

			if interactiveFlag(cmd) {
				return runPager(io, "Sites", cols, func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
					listOpts := opts
					listOpts.Offset, listOpts.Limit = po.Offset, po.Limit
					applyPagerQuery(&listOpts.Extra, po.Query)
					page, err := client.ListSites(ctx, listOpts)
					if err != nil {
						return pager.FetchResult{}, err
					}
					return pager.FetchResult{Rows: page.Results, Total: page.Count}, nil
				})
			}

			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}
			if fetchAll {
				return renderStreaming[netbox.Site](cmd, io, client.SitesFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListSites(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
