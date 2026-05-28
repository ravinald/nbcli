package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
)

// contactKeywords is the positional keyword set for `nbcli show contacts`.
var contactKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact contact name"},
	{Name: "email", Description: "email address"},
	{Name: "phone", Description: "phone number"},
	{Name: "group", Description: "contact group slug"},
}, cmdutils.PaginationKeywords()...)

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

			cols := []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.Contact).ID) }},
				{Header: "Name", Extract: func(r any) string { return r.(netbox.Contact).Name }},
				{Header: "Title", Extract: func(r any) string { return r.(netbox.Contact).Title }},
				{Header: "Email", Extract: func(r any) string { return r.(netbox.Contact).Email }},
				{Header: "Phone", Extract: func(r any) string { return r.(netbox.Contact).Phone }},
				{Header: "Group", Extract: func(r any) string {
					if r.(netbox.Contact).Group == nil {
						return ""
					}
					return r.(netbox.Contact).Group.Name
				}},
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
