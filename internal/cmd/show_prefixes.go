package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
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
			var rows []netbox.Prefix
			if fetchAll {
				rows, err = netbox.ListAll(cmd.Context(),
					client.PrefixesFetcher(opts),
					netbox.IterateOptions{PageSize: 100, MaxPages: 200})
			} else {
				var page netbox.Page[netbox.Prefix]
				page, err = client.ListPrefixes(cmd.Context(), opts)
				rows = page.Results
			}
			if err != nil {
				return err
			}
			return renderRows(cmd, io, rows, []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.Prefix).ID) }},
				{Header: "Prefix", Extract: func(r any) string { return r.(netbox.Prefix).Prefix }},
				{Header: "Family", Extract: func(r any) string { return r.(netbox.Prefix).Family.Label }},
				{Header: "VRF", Extract: func(r any) string {
					if r.(netbox.Prefix).VRF == nil {
						return ""
					}
					return r.(netbox.Prefix).VRF.Name
				}},
				{Header: "Site", Extract: func(r any) string {
					if r.(netbox.Prefix).Site == nil {
						return ""
					}
					return r.(netbox.Prefix).Site.Name
				}},
				{Header: "Role", Extract: func(r any) string {
					if r.(netbox.Prefix).Role == nil {
						return ""
					}
					return r.(netbox.Prefix).Role.Name
				}},
				{Header: "Status", Extract: func(r any) string { return r.(netbox.Prefix).Status.Label }},
				{Header: "Tenant", Extract: func(r any) string {
					if r.(netbox.Prefix).Tenant == nil {
						return ""
					}
					return r.(netbox.Prefix).Tenant.Name
				}},
			})
		},
	}
}
