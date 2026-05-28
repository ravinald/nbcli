package cmd

import (
	"errors"
	"iter"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/netbox"
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

// errStopIteration is a sentinel used inside renderStreaming to signal the
// renderer told us to stop. Never propagated up to the user.
var errStopIteration = errors.New("render: stream consumer halted")

// renderStreaming is the fetchAll counterpart to renderRows. When the
// resolved format implements output.StreamingRenderer (json, yaml, tsv) and
// isn't table, rows are written as they arrive from netbox.Iterate — memory
// stays O(1) and the user sees output incrementally. Otherwise we fall back
// to the batch path (table needs all rows to align columns).
func renderStreaming[T any](
	cmd *cobra.Command,
	io IO,
	fetcher netbox.PageFetcher[T],
	iterOpts netbox.IterateOptions,
	cols []output.Column,
) error {
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

	sr, streamable := r.(output.StreamingRenderer)
	if !streamable || format == output.FormatTable {
		rows, err := netbox.ListAll(cmd.Context(), fetcher, iterOpts)
		if err != nil {
			return err
		}
		return r.Render(io.Out, cols, rows)
	}

	var iterErr error
	seq := iter.Seq[any](func(yield func(any) bool) {
		iterErr = netbox.Iterate(cmd.Context(), fetcher, iterOpts, func(row T) error {
			if !yield(row) {
				return errStopIteration
			}
			return nil
		})
	})
	if err := sr.Stream(io.Out, cols, seq); err != nil {
		return err
	}
	if iterErr != nil && !errors.Is(iterErr, errStopIteration) {
		return iterErr
	}
	return nil
}
