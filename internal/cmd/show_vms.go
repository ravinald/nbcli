package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
)

var vmKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact VM name"},
	{Name: "status", Description: "status value", Example: "active",
		Values: []string{"offline", "active", "planned", "staged", "failed", "decommissioning"}},
	{Name: "site", Description: "site slug"},
	{Name: "cluster", Description: "cluster name"},
	{Name: "tenant", Description: "tenant slug"},
	{Name: "role", Description: "VM role slug"},
}, cmdutils.PaginationKeywords()...)

func newShowVMsCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:     "vms " + cmdutils.UsageLine(vmKeywords),
		Short:   "List virtual machines",
		Aliases: []string{"virtual-machines", "vm"},
		Long: "List virtual machines. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(vmKeywords) +
			"\nExamples:\n" +
			"  nbcli show vms cluster prod-vm-1\n" +
			"  nbcli show vms status active site hq\n",
		Args:              cmdutils.Validator(vmKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(vmKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, vmKeywords)
			opts := netbox.ListVMsOptions{
				Name:    kv["name"],
				Status:  kv["status"],
				Site:    kv["site"],
				Cluster: kv["cluster"],
				Tenant:  kv["tenant"],
				Role:    kv["role"],
				Limit:   50,
			}
			fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
			if err != nil {
				return err
			}
			client, err := clientFromCtx(cmd)
			if err != nil {
				return err
			}
			cols := resolveColumns(cmd, "virtual-machines")
			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}

			if fetchAll {
				return renderStreaming[netbox.VirtualMachine](cmd, io, client.VMsFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListVMs(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
