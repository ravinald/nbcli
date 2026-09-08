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
		Long:  "Read-only queries against Netbox. Append `format json|yaml|tsv` to any subcommand for machine output.",
	}
	cmd.AddCommand(
		newShowSitesCmd(io),
		newShowRacksCmd(io),
		newShowDevicesCmd(io),
		newShowInterfacesCmd(io),
		newShowPrefixesCmd(io),
		newShowIPAddressesCmd(io),
		newShowVLANsCmd(io),
		newShowVRFsCmd(io),
		newShowVMsCmd(io),
		newShowClustersCmd(io),
		newShowTenantsCmd(io),
		newShowContactsCmd(io),
	)
	return cmd
}
