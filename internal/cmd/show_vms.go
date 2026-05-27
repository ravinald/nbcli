package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
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
			var rows []netbox.VirtualMachine
			if fetchAll {
				rows, err = netbox.ListAll(cmd.Context(),
					client.VMsFetcher(opts),
					netbox.IterateOptions{PageSize: 100, MaxPages: 200})
			} else {
				var page netbox.Page[netbox.VirtualMachine]
				page, err = client.ListVMs(cmd.Context(), opts)
				rows = page.Results
			}
			if err != nil {
				return err
			}
			return renderRows(cmd, io, rows, []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.VirtualMachine).ID) }},
				{Header: "Name", Extract: func(r any) string { return r.(netbox.VirtualMachine).Name }},
				{Header: "Status", Extract: func(r any) string { return r.(netbox.VirtualMachine).Status.Label }},
				{Header: "Cluster", Extract: func(r any) string {
					if r.(netbox.VirtualMachine).Cluster == nil {
						return ""
					}
					return r.(netbox.VirtualMachine).Cluster.Name
				}},
				{Header: "Site", Extract: func(r any) string {
					if r.(netbox.VirtualMachine).Site == nil {
						return ""
					}
					return r.(netbox.VirtualMachine).Site.Name
				}},
				{Header: "vCPUs", Extract: func(r any) string {
					if c := r.(netbox.VirtualMachine).VCPUs; c != nil {
						return strconv.FormatFloat(*c, 'f', -1, 64)
					}
					return ""
				}},
				{Header: "MemMB", Extract: func(r any) string {
					if m := r.(netbox.VirtualMachine).Memory; m != nil {
						return strconv.Itoa(*m)
					}
					return ""
				}},
				{Header: "DiskGB", Extract: func(r any) string {
					if d := r.(netbox.VirtualMachine).Disk; d != nil {
						return strconv.Itoa(*d)
					}
					return ""
				}},
				{Header: "Tenant", Extract: func(r any) string {
					if r.(netbox.VirtualMachine).Tenant == nil {
						return ""
					}
					return r.(netbox.VirtualMachine).Tenant.Name
				}},
			})
		},
	}
}
