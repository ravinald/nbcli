package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/tui"
)

// newTUICmd launches the bubbletea interface mirroring the Netbox web UI.
// CLI subcommands (`show`, `plugin`) work without ever entering the TUI.
func newTUICmd(_ IO) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive TUI",
		Long:  "Open the full-screen Netbox-style browser. Quit with q or Ctrl+C.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientFromCtx(cmd)
			if err != nil {
				return err
			}
			return tui.Run(client)
		},
	}
}
