package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/version"
)

// newVersionCmd reports build metadata. JSON output is supported so CI can
// parse the version of a built binary without scraping text.
func newVersionCmd(io IO) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print nbcli version and build info",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			info := version.Get()
			if asJSON {
				enc := json.NewEncoder(io.Out)
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			_, err := fmt.Fprintln(io.Out, info.String())
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit version info as JSON")
	return cmd
}
