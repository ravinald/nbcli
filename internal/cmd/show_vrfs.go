package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/pager"
)

var vrfKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact VRF name"},
	{Name: "rd", Description: "route distinguisher"},
	{Name: "tenant", Description: "tenant slug"},
}, cmdutils.TrailingKeywords()...)

func newShowVRFsCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "vrfs " + cmdutils.UsageLine(vrfKeywords),
		Short: "List IPAM VRFs",
		Long: "List IPAM VRFs. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(vrfKeywords) +
			"\nExamples:\n" +
			"  nbcli show vrfs\n" +
			"  nbcli show vrfs tenant acme\n",
		Args:              cmdutils.Validator(vrfKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(vrfKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, vrfKeywords)
			opts := netbox.ListVRFsOptions{
				Name:   kv["name"],
				RD:     kv["rd"],
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
			cols := resolveColumns(cmd, "vrfs", kv)

			if kv["pager"] == "true" {
				return runPager(io, "VRFs", cols, func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
					listOpts := opts
					listOpts.Offset, listOpts.Limit = po.Offset, po.Limit
					applyPagerQuery(&listOpts.Extra, po.Query)
					page, err := client.ListVRFs(ctx, listOpts)
					if err != nil {
						return pager.FetchResult{}, err
					}
					return pager.FetchResult{Rows: page.Results, Total: page.Count}, nil
				})
			}

			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}
			if fetchAll {
				return renderStreaming[netbox.VRF](cmd, io, client.VRFsFetcher(opts), iterOpts, cols, kv)
			}
			page, err := client.ListVRFs(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols, kv)
		},
	}
}
