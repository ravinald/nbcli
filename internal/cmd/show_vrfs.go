package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
)

var vrfKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact VRF name"},
	{Name: "rd", Description: "route distinguisher"},
	{Name: "tenant", Description: "tenant slug"},
}, cmdutils.PaginationKeywords()...)

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
			var rows []netbox.VRF
			if fetchAll {
				rows, err = netbox.ListAll(cmd.Context(),
					client.VRFsFetcher(opts),
					netbox.IterateOptions{PageSize: 100, MaxPages: 200})
			} else {
				var page netbox.Page[netbox.VRF]
				page, err = client.ListVRFs(cmd.Context(), opts)
				rows = page.Results
			}
			if err != nil {
				return err
			}
			return renderRows(cmd, io, rows, []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.VRF).ID) }},
				{Header: "Name", Extract: func(r any) string { return r.(netbox.VRF).Name }},
				{Header: "RD", Extract: func(r any) string { return r.(netbox.VRF).RD }},
				{Header: "Tenant", Extract: func(r any) string {
					if r.(netbox.VRF).Tenant == nil {
						return ""
					}
					return r.(netbox.VRF).Tenant.Name
				}},
				{Header: "Description", Extract: func(r any) string { return r.(netbox.VRF).Description }},
			})
		},
	}
}
