package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/output"
	"github.com/ravinald/nbcli/internal/plugins"
)

// newPluginCmd is the parent for Netbox plugin commands. Two paths:
//
//  1. Named plugins registered via plugins.Register() show up as subcommands.
//  2. `nbcli plugin passthrough` forwards any call to /api/plugins/<name>/...
//     for plugins that don't have a typed wrapper yet.
func newPluginCmd(io IO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Access Netbox plugin endpoints",
	}
	cmd.AddCommand(newPassthroughCmd(io))
	cmd.AddCommand(newPluginListCmd(io))
	for _, p := range plugins.Default().List() {
		sub := &cobra.Command{Use: p.Name(), Short: p.Title()}
		sub.AddCommand(p.Commands()...)
		cmd.AddCommand(sub)
	}
	return cmd
}

// newPluginListCmd prints the registered plugins. Helpful for confirming a
// compile-time integration was wired in.
func newPluginListCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List plugins that nbcli knows about (compiled-in)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			list := plugins.Default().List()
			if len(list) == 0 {
				_, err := fmt.Fprintln(io.Out, "(no named plugins compiled in; use `nbcli plugin passthrough`)")
				return err
			}
			for _, p := range list {
				if _, err := fmt.Fprintf(io.Out, "%-30s %s\n", p.Name(), p.Title()); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// newPassthroughCmd forwards an HTTP call to /api/plugins/<name>/<subpath>.
//
// Junos-style shape:
//
//	nbcli plugin passthrough <name> <subpath> [key value ...]
//
// The trailing pairs become URL query parameters. --method stays as a flag
// because it controls how nbcli speaks HTTP, not what filter it asks for.
func newPassthroughCmd(io IO) *cobra.Command {
	var method string

	cmd := &cobra.Command{
		Use:   "passthrough <plugin> <subpath> [key value ...]",
		Short: "Forward a raw request to a Netbox plugin endpoint",
		Long: "Forward a raw request to /api/plugins/<plugin>/<subpath>. Any trailing\n" +
			"positional key/value pairs become URL query parameters.\n\n" +
			"Examples:\n" +
			"  nbcli plugin passthrough wireless-controllers controllers/\n" +
			"  nbcli plugin passthrough my-plugin some/endpoint/ site hq limit 10\n",
		Args: passthroughArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromCtx(cmd)
			if err != nil {
				return err
			}
			pluginName, subpath := args[0], args[1]
			q := url.Values{}
			for i := 2; i < len(args); i += 2 {
				q.Add(args[i], args[i+1])
			}

			res, err := plugins.Passthrough(cmd.Context(), client, pluginName, strings.ToUpper(method), subpath, q, nil)
			if err != nil {
				return err
			}

			cfg := configFromCtx(cmd.Context())
			explicit, err := output.Parse(cfg.Format)
			if err != nil {
				return err
			}
			format := output.Resolve(explicit, io.Out)
			// Passthrough has no fixed schema → table is meaningless. Coerce to JSON.
			if format == output.FormatTable {
				format = output.FormatJSON
			}
			r, err := output.New(format)
			if err != nil {
				return err
			}
			return r.Render(io.Out, nil, []any{res.Body})
		},
	}
	cmd.Flags().StringVar(&method, "method", "GET", "HTTP method (operational flag)")
	return cmd
}

// passthroughArgs enforces: <plugin> <subpath> [key value]*.
func passthroughArgs(_ *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("expected <plugin> <subpath> [key value ...], got %d args", len(args))
	}
	if (len(args)-2)%2 != 0 {
		return fmt.Errorf("trailing args must be key/value pairs (got %d extra)", len(args)-2)
	}
	return nil
}
