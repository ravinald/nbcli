package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
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
}, cmdutils.PaginationKeywords()...)

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
			var rows []netbox.VLAN
			if fetchAll {
				rows, err = netbox.ListAll(cmd.Context(),
					client.VLANsFetcher(opts),
					netbox.IterateOptions{PageSize: 100, MaxPages: 200})
			} else {
				var page netbox.Page[netbox.VLAN]
				page, err = client.ListVLANs(cmd.Context(), opts)
				rows = page.Results
			}
			if err != nil {
				return err
			}
			return renderRows(cmd, io, rows, []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.VLAN).ID) }},
				{Header: "VID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.VLAN).VID) }},
				{Header: "Name", Extract: func(r any) string { return r.(netbox.VLAN).Name }},
				{Header: "Site", Extract: func(r any) string {
					if r.(netbox.VLAN).Site == nil {
						return ""
					}
					return r.(netbox.VLAN).Site.Name
				}},
				{Header: "Group", Extract: func(r any) string {
					if r.(netbox.VLAN).Group == nil {
						return ""
					}
					return r.(netbox.VLAN).Group.Name
				}},
				{Header: "Status", Extract: func(r any) string { return r.(netbox.VLAN).Status.Label }},
				{Header: "Role", Extract: func(r any) string {
					if r.(netbox.VLAN).Role == nil {
						return ""
					}
					return r.(netbox.VLAN).Role.Name
				}},
				{Header: "Tenant", Extract: func(r any) string {
					if r.(netbox.VLAN).Tenant == nil {
						return ""
					}
					return r.(netbox.VLAN).Tenant.Name
				}},
			})
		},
	}
}
