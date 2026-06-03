package cmd

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/columns"
)

// newColumnsCmd is `nbcli columns [resource]`. Without an argument it lists
// every resource the registry knows about; with one, it lists that
// resource's available column names + headers + default-visibility flag —
// the menu to choose from when writing config.yaml or using the `columns`
// positional on a show command.
func newColumnsCmd(io IO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "columns [resource]",
		Short: "List available columns for a resource",
		Long: "Lists the column menu for a resource (or all resources when called " +
			"without an argument). Use the names in config.yaml's `columns:` map " +
			"or as the value of the `columns` positional on a show/search command " +
			"(e.g. `nbcli show sites columns id,name,status`).\n\n" +
			"Examples:\n" +
			"  nbcli columns                # list all resources\n" +
			"  nbcli columns sites          # show available columns for sites\n" +
			"  nbcli columns devices        # show available columns for devices\n",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			reg := columns.Registry()
			if len(args) == 0 {
				names := make([]string, 0, len(reg))
				for n := range reg {
					names = append(names, n)
				}
				sort.Strings(names)
				if _, err := fmt.Fprintln(io.Out, "Resources (run `nbcli columns <resource>` for details):"); err != nil {
					return err
				}
				for _, n := range names {
					if _, err := fmt.Fprintf(io.Out, "  %s\n", n); err != nil {
						return err
					}
				}
				return nil
			}
			resource := args[0]
			set, ok := reg[resource]
			if !ok {
				available := make([]string, 0, len(reg))
				for n := range reg {
					available = append(available, n)
				}
				sort.Strings(available)
				return fmt.Errorf("unknown resource %q (one of: %v)", resource, available)
			}
			if _, err := fmt.Fprintf(io.Out, "Columns for %s:\n\n", resource); err != nil {
				return err
			}
			tw := tabwriter.NewWriter(io.Out, 0, 0, 2, ' ', 0)
			defer func() { _ = tw.Flush() }()
			if _, err := fmt.Fprintln(tw, "NAME\tHEADER\tDEFAULT"); err != nil {
				return err
			}
			for _, c := range set.Columns {
				flag := ""
				if c.Default {
					flag = "yes"
				}
				if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", c.Name, c.Header, flag); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
