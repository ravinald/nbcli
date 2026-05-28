// Package cmd builds the cobra command tree. Every command is constructed by
// a small factory (newFooCmd) so the tree is composable and unit-testable.
//
// The flag/env/config layering is set up in PersistentPreRunE on the root —
// every subcommand can read the resolved *config.Config from cmd.Context().
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/config"
	"github.com/ravinald/nbcli/internal/logging"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/version"
)

// contextKey is unexported to avoid collisions with any keys other packages
// might stash in cmd.Context().
type contextKey string

const (
	ctxKeyConfig contextKey = "nbcli.config"
	ctxKeyClient contextKey = "nbcli.client"
)

// IO bundles the streams a command writes to. Lifted out so tests can capture
// stdout/stderr without touching os.Stdout.
type IO struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// StdIO returns IO bound to the real process streams.
func StdIO() IO { return IO{In: os.Stdin, Out: os.Stdout, Err: os.Stderr} }

// rootOptions holds the persistent flags resolved by viper.
type rootOptions struct {
	configFile string
	envFile    string
	url        string
	format     string
	timeout    time.Duration
	insecure   bool
	verbose    bool
}

// NewRootCmd returns the top-level `nbcli` command with all subcommands wired in.
// io is the stream set commands write to. Pass StdIO() for normal use.
func NewRootCmd(io IO) *cobra.Command {
	opts := &rootOptions{}

	root := &cobra.Command{
		Use:           "nbcli",
		Short:         "Modern CLI + TUI for Netbox",
		Long:          "nbcli queries Netbox via cobra subcommands or a bubbletea TUI that mirrors the web UI.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Get().String(),
	}

	root.PersistentFlags().StringVar(&opts.configFile, "config", "", "path to config file (default: $XDG_CONFIG_HOME/nbcli/config.yaml)")
	root.PersistentFlags().StringVar(&opts.envFile, "env-file", "", "additional env file (overlays ~/.config/nbcli/secrets.env and ~/.env.netbox)")
	root.PersistentFlags().StringVar(&opts.url, "url", "", "Netbox base URL (env NBCLI_URL)")
	root.PersistentFlags().StringVar(&opts.format, "format", "", "output format: table|json|yaml|tsv (env NBCLI_FORMAT)")
	root.PersistentFlags().DurationVar(&opts.timeout, "timeout", 0, "HTTP request timeout (e.g. 10s)")
	root.PersistentFlags().BoolVar(&opts.insecure, "insecure", false, "skip TLS cert verification (dangerous; dev only)")
	root.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose logging to stderr")

	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		logging.Setup(opts.verbose, io.Err)
		cfg, err := config.Load(cmd.Flags(), opts.configFile, opts.envFile)
		if err != nil {
			return err
		}
		ctx := context.WithValue(cmd.Context(), ctxKeyConfig, &cfg)
		cmd.SetContext(ctx)
		return nil
	}

	root.AddCommand(
		newVersionCmd(io),
		newShowCmd(io),
		newTUICmd(io),
		newPluginCmd(io),
	)

	return root
}

// Execute is the single entry point used by main(). It returns an exit code
// so main can pass it to os.Exit without bringing cobra into scope there.
func Execute(args []string, io IO) int {
	root := NewRootCmd(io)
	root.SetArgs(args)
	root.SetOut(io.Out)
	root.SetErr(io.Err)
	root.SetIn(io.In)
	if err := root.ExecuteContext(context.Background()); err != nil {
		_, _ = fmt.Fprintf(io.Err, "Error: %v\n", err)
		return 1
	}
	return 0
}

// configFromCtx returns the *config.Config stashed by PersistentPreRunE.
// Panics if missing — that's a programming error in the command tree.
func configFromCtx(ctx context.Context) *config.Config {
	c, ok := ctx.Value(ctxKeyConfig).(*config.Config)
	if !ok {
		panic("cmd: no *config.Config in context — PersistentPreRunE didn't run")
	}
	return c
}

// clientFromCtx returns a cached *netbox.Client for this command tree, building
// one on first use. Token + URL are required; this is also the call site that
// surfaces "missing token / url" errors to the user.
func clientFromCtx(cmd *cobra.Command) (*netbox.Client, error) {
	ctx := cmd.Context()
	if c, ok := ctx.Value(ctxKeyClient).(*netbox.Client); ok && c != nil {
		return c, nil
	}
	cfg := configFromCtx(ctx)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.RequireToken(); err != nil {
		return nil, err
	}
	c, err := netbox.New(netbox.Options{
		BaseURL:            cfg.URL,
		Token:              cfg.Token,
		Timeout:            time.Duration(cfg.TimeoutSeconds) * time.Second,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	})
	if err != nil {
		return nil, err
	}
	cmd.SetContext(context.WithValue(ctx, ctxKeyClient, c))
	return c, nil
}
