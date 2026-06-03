package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/pager"
)

var vlanKeywords = append([]cmdutils.KeywordSpec{
	{Name: "vid", Description: "VLAN ID number"},
	{Name: "name", Description: "exact VLAN name"},
	{Name: "site", Description: "site slug"},
	{Name: "group", Description: "VLAN group slug"},
	{Name: "role", Description: "VLAN role slug"},
	{Name: "status", Description: "status value",
		Values: []string{"active", "reserved", "deprecated"}},
	{Name: "tenant", Description: "tenant slug"},
}, cmdutils.TrailingKeywords()...)

func newShowVLANsCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "vlans " + cmdutils.UsageLine(vlanKeywords),
		Short: "List IPAM VLANs",
		Long: "List IPAM VLANs. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(vlanKeywords) +
			"\nExamples:\n" +
			"  nbcli show vlans site hq\n" +
			"  nbcli show vlans vid 100\n",
		Args:              cmdutils.Validator(vlanKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(vlanKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, vlanKeywords)
			opts := netbox.ListVLANsOptions{
				VID:    kv["vid"],
				Name:   kv["name"],
				Site:   kv["site"],
				Group:  kv["group"],
				Role:   kv["role"],
				Status: kv["status"],
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
			cols := resolveColumns(cmd, "vlans", kv)

			if kv["pager"] == "true" {
				return runPager(io, "VLANs", cols, func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
					listOpts := opts
					listOpts.Offset, listOpts.Limit = po.Offset, po.Limit
					applyPagerQuery(&listOpts.Extra, po.Query)
					page, err := client.ListVLANs(ctx, listOpts)
					if err != nil {
						return pager.FetchResult{}, err
					}
					return pager.FetchResult{Rows: page.Results, Total: page.Count}, nil
				})
			}

			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}
			if fetchAll {
				return renderStreaming[netbox.VLAN](cmd, io, client.VLANsFetcher(opts), iterOpts, cols, kv)
			}
			page, err := client.ListVLANs(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols, kv)
		},
	}
}
