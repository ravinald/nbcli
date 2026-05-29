package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
)

var clusterKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact cluster name"},
	{Name: "type", Description: "cluster type slug", Example: "vmware-vsphere"},
	{Name: "group", Description: "cluster group slug"},
	{Name: "site", Description: "site slug"},
	{Name: "status", Description: "status value",
		Values: []string{"active", "planned", "staging", "decommissioning", "offline"}},
}, cmdutils.PaginationKeywords()...)

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
