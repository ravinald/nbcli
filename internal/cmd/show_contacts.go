package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/pager"
)

// contactKeywords is the positional keyword set for `nbcli show contacts`.
var contactKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact contact name"},
	{Name: "email", Description: "email address"},
	{Name: "phone", Description: "phone number"},
	{Name: "group", Description: "contact group slug"},
}, append(cmdutils.PaginationKeywords(), cmdutils.PagerKeyword())...)

func newShowContactsCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "contacts " + cmdutils.UsageLine(contactKeywords),
		Short: "List contacts",
		Long: "List contacts. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(contactKeywords) +
			"\nExamples:\n" +
			"  nbcli show contacts\n" +
			"  nbcli show contacts group ops\n" +
			"  nbcli show contacts limit 0\n",
		Args:              cmdutils.Validator(contactKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(contactKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, contactKeywords)

			opts := netbox.ListContactsOptions{
				Name:  kv["name"],
				Email: kv["email"],
				Phone: kv["phone"],
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

			cols := resolveColumns(cmd, "contacts")

			if kv["pager"] == "true" {
				return runPager(io, "Contacts", cols, func(ctx context.Context, po pager.FetchOpts) (pager.FetchResult, error) {
					listOpts := opts
					listOpts.Offset, listOpts.Limit = po.Offset, po.Limit
					applyPagerQuery(&listOpts.Extra, po.Query)
					page, err := client.ListContacts(ctx, listOpts)
					if err != nil {
						return pager.FetchResult{}, err
					}
					return pager.FetchResult{Rows: page.Results, Total: page.Count}, nil
				})
			}

			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}
			if fetchAll {
				return renderStreaming[netbox.Contact](cmd, io, client.ContactsFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListContacts(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
