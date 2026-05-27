package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/output"
)

// renderRows centralizes the boilerplate at the tail of every `show <resource>`
// command: read the resolved format off config, build the renderer, write to
// io.Out. Pulled out so adding a new resource is just columns + a slice.
//
// Two non-trivial responsibilities: format resolution (explicit flag / env /
// config / TTY-implicit) and renderer dispatch. The previous per-command
// inline version was identical ~25 lines, so this is a pure dedup.
func renderRows(cmd *cobra.Command, io IO, rows any, cols []output.Column) error {
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
	return r.Render(io.Out, cols, rows)
}
