package cmd

import (
	"github.com/spf13/cobra"
)

// newShowCmd is the parent of all read-only list/detail commands.
// Convention: `nbcli show <resource> [flags]`. Output format flows from the
// persistent --format flag on root.
func newShowCmd(io IO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show Netbox resources",
		Long:  "Read-only queries against Netbox. Combine with --format json|yaml|tsv for machine output.",
	}
	cmd.AddCommand(
		newShowSitesCmd(io),
		newShowTenantsCmd(io),
		newShowContactsCmd(io),
	)
	return cmd
}
