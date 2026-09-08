package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/pager"
)

// rackKeywords is the positional keyword set for `nbcli show racks`.
var rackKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact rack name"},
	{Name: "site", Description: "site slug"},
	{Name: "status", Description: "status value", Example: "active",
		Values: []string{"reserved", "available", "planned", "active", "deprecated"}},
	{Name: "role", Description: "rack role slug"},
	{Name: "location", Description: "location slug"},
	{Name: "tenant", Description: "tenant slug"},
}, cmdutils.TrailingKeywords()...)

func newShowRacksCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "racks " + cmdutils.UsageLine(rackKeywords),
		Short: "List DCIM racks",
		Long: "List DCIM racks. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(rackKeywords) +
			"\nExamples:\n" +
			"  nbcli show racks site hq\n" +
			"  nbcli show racks status active location row-a\n",
		Args:              cmdutils.Validator(rackKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(rackKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, rackKeywords)

			opts := netbox.ListRacksOptions{
				Name:     kv["name"],
				Site:     kv["site"],
				Status:   kv["status"],
				Role:     kv["role"],
				Location: kv["location"],
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

			cols := resolveColumns(cmd, "racks", kv)

			if kv["pager"] == "true" {
				return runPager(io, "Racks", cols, func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
					listOpts := opts
					listOpts.Offset, listOpts.Limit = po.Offset, po.Limit
					applyPagerQuery(&listOpts.Extra, po.Query)
					page, err := client.ListRacks(ctx, listOpts)
					if err != nil {
						return pager.FetchResult{}, err
					}
					return pager.FetchResult{Rows: page.Results, Total: page.Count}, nil
				})
			}

			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}
			if fetchAll {
				return renderStreaming[netbox.Rack](cmd, io, client.RacksFetcher(opts), iterOpts, cols, kv)
			}
			page, err := client.ListRacks(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols, kv)
		},
	}
}
