package cmd

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
)

// siteKeywords is the positional keyword set for `nbcli show sites`.
// Filters interact with the Netbox API → positional. Operational concerns
// (--format, --url, ...) stay as flags on root.
var siteKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact site name"},
	{Name: "slug", Description: "site slug"},
	{Name: "status", Description: "status value", Example: "active",
		Values: []string{"active", "planned", "staging", "decommissioning", "retired"}},
	{Name: "region", Description: "region slug"},
	{Name: "tenant", Description: "tenant slug"},
}, cmdutils.PaginationKeywords()...)

// newShowSitesCmd implements `nbcli show sites [keyword value]...`.
// The same shape extends to every other resource: declare the keyword set,
// hand it to cmdutils, map the parsed values onto the typed Options struct.
func newShowSitesCmd(io IO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sites " + cmdutils.UsageLine(siteKeywords),
		Short: "List DCIM sites",
		Long: "List DCIM sites. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(siteKeywords) +
			"\nExamples:\n" +
			"  nbcli show sites\n" +
			"  nbcli show sites status active\n" +
			"  nbcli show sites region us-west status active limit 100\n",
		Args:              cmdutils.Validator(siteKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(siteKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, siteKeywords)

			opts := netbox.ListSitesOptions{
				Name:   kv["name"],
				Slug:   kv["slug"],
				Status: kv["status"],
				Region: kv["region"],
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
			var rows []netbox.Site
			if fetchAll {
				rows, err = netbox.ListAll(cmd.Context(),
					netbox.PageFetcher[netbox.Site](func(ctx context.Context, offset, limit int) (netbox.Page[netbox.Site], error) {
						o := opts
						o.Offset, o.Limit = offset, limit
						return client.ListSites(ctx, o)
					}),
					netbox.IterateOptions{PageSize: 100, MaxPages: 200})
				if err != nil {
					return err
				}
			} else {
				page, err := client.ListSites(cmd.Context(), opts)
				if err != nil {
					return err
				}
				rows = page.Results
			}

			cfg := configFromCtx(cmd.Context())
			explicit, err := output.Parse(cfg.Format)
			if err != nil {
				return err
			}
			format := output.Resolve(explicit, io.Out)
			r, err := output.New(format)
			if err != nil {
				return err
			}

			cols := []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.Site).ID) }},
				{Header: "Name", Extract: func(r any) string { return r.(netbox.Site).Name }},
				{Header: "Slug", Extract: func(r any) string { return r.(netbox.Site).Slug }},
				{Header: "Status", Extract: func(r any) string { return r.(netbox.Site).Status.Label }},
				{Header: "Region", Extract: func(r any) string {
					if r.(netbox.Site).Region == nil {
						return ""
					}
					return r.(netbox.Site).Region.Name
				}},
				{Header: "Tenant", Extract: func(r any) string {
					if r.(netbox.Site).Tenant == nil {
						return ""
					}
					return r.(netbox.Site).Tenant.Name
				}},
			}
			return r.Render(io.Out, cols, rows)
		},
	}
	return cmd
}
