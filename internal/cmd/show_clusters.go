package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/pager"
)

var clusterKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact cluster name"},
	{Name: "type", Description: "cluster type slug", Example: "vmware-vsphere"},
	{Name: "group", Description: "cluster group slug"},
	{Name: "site", Description: "site slug"},
	{Name: "status", Description: "status value",
		Values: []string{"active", "planned", "staging", "decommissioning", "offline"}},
}, append(cmdutils.PaginationKeywords(), cmdutils.PagerKeyword())...)

func newShowClustersCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "clusters " + cmdutils.UsageLine(clusterKeywords),
		Short: "List virtualization clusters",
		Long: "List virtualization clusters. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(clusterKeywords) +
			"\nExamples:\n" +
			"  nbcli show clusters type vmware-vsphere\n" +
			"  nbcli show clusters site hq\n",
		Args:              cmdutils.Validator(clusterKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(clusterKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, clusterKeywords)
			opts := netbox.ListClustersOptions{
				Name:   kv["name"],
				Type:   kv["type"],
				Group:  kv["group"],
				Site:   kv["site"],
				Status: kv["status"],
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
			cols := resolveColumns(cmd, "clusters")

			if kv["pager"] == "true" {
				return runPager(io, "Clusters", cols, func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
					listOpts := opts
					listOpts.Offset, listOpts.Limit = po.Offset, po.Limit
					applyPagerQuery(&listOpts.Extra, po.Query)
					page, err := client.ListClusters(ctx, listOpts)
					if err != nil {
						return pager.FetchResult{}, err
					}
					return pager.FetchResult{Rows: page.Results, Total: page.Count}, nil
				})
			}

			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}
			if fetchAll {
				return renderStreaming[netbox.Cluster](cmd, io, client.ClustersFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListClusters(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
