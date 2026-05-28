package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
)

// tenantKeywords is the positional keyword set for `nbcli show tenants`.
var tenantKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact tenant name"},
	{Name: "slug", Description: "tenant slug"},
	{Name: "group", Description: "tenant group slug"},
}, cmdutils.PaginationKeywords()...)

func newShowTenantsCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "tenants " + cmdutils.UsageLine(tenantKeywords),
		Short: "List tenants",
		Long: "List tenants. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(tenantKeywords) +
			"\nExamples:\n" +
			"  nbcli show tenants\n" +
			"  nbcli show tenants group engineering\n" +
			"  nbcli show tenants limit 0\n",
		Args:              cmdutils.Validator(tenantKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(tenantKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, tenantKeywords)

			opts := netbox.ListTenantsOptions{
				Name:  kv["name"],
				Slug:  kv["slug"],
				Group: kv["group"],
				Limit: 50,
			}
			fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
			if err != nil {
				return err
			}

			client, err := clientFromCtx(cmd)
			if err != nil {
				return err
			}

			cols := []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.Tenant).ID) }},
				{Header: "Name", Extract: func(r any) string { return r.(netbox.Tenant).Name }},
				{Header: "Slug", Extract: func(r any) string { return r.(netbox.Tenant).Slug }},
				{Header: "Group", Extract: func(r any) string {
					if r.(netbox.Tenant).Group == nil {
						return ""
					}
					return r.(netbox.Tenant).Group.Name
				}},
				{Header: "Description", Extract: func(r any) string { return r.(netbox.Tenant).Description }},
			}
			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}

			if fetchAll {
				return renderStreaming[netbox.Tenant](cmd, io, client.TenantsFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListTenants(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
