package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
)

// rackKeywords is the positional keyword set for `nbcli show racks`.
var rackKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact rack name"},
	{Name: "site", Description: "site slug"},
	{Name: "status", Description: "status value", Example: "active",
		Values: []string{"reserved", "available", "planned", "active", "deprecated"}},
	{Name: "role", Description: "rack role slug"},
	{Name: "location", Description: "location slug"},
	{Name: "tenant", Description: "tenant slug"},
}, cmdutils.PaginationKeywords()...)

func newShowRacksCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "racks " + cmdutils.UsageLine(rackKeywords),
		Short: "List DCIM racks",
		Long: "List DCIM racks. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(rackKeywords) +
			"\nExamples:\n" +
			"  nbcli show racks site hq\n" +
			"  nbcli show racks status active location row-a\n",
		Args:              cmdutils.Validator(rackKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(rackKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, rackKeywords)

			opts := netbox.ListRacksOptions{
				Name:     kv["name"],
				Site:     kv["site"],
				Status:   kv["status"],
				Role:     kv["role"],
				Location: kv["location"],
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

			cols := []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.Rack).ID) }},
				{Header: "Name", Extract: func(r any) string { return r.(netbox.Rack).Name }},
				{Header: "Site", Extract: func(r any) string {
					if r.(netbox.Rack).Site == nil {
						return ""
					}
					return r.(netbox.Rack).Site.Name
				}},
				{Header: "Location", Extract: func(r any) string {
					if r.(netbox.Rack).Location == nil {
						return ""
					}
					return r.(netbox.Rack).Location.Name
				}},
				{Header: "Role", Extract: func(r any) string {
					if r.(netbox.Rack).Role == nil {
						return ""
					}
					return r.(netbox.Rack).Role.Name
				}},
				{Header: "Status", Extract: func(r any) string { return r.(netbox.Rack).Status.Label }},
				{Header: "U", Extract: func(r any) string { return strconv.Itoa(r.(netbox.Rack).UHeight) }},
				{Header: "Tenant", Extract: func(r any) string {
					if r.(netbox.Rack).Tenant == nil {
						return ""
					}
					return r.(netbox.Rack).Tenant.Name
				}},
			}
			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}

			if fetchAll {
				return renderStreaming[netbox.Rack](cmd, io, client.RacksFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListRacks(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
