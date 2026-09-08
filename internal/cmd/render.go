package cmd

import (
	"errors"
	"iter"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/columns"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
)

// resolveColumns picks the column set for resource. Precedence (highest first):
//
//  1. positional kv["columns"] — `nbcli show sites columns id,name,status`
//  2. cfg.Columns[resource] from config.yaml (and NBCLI_FORMAT-like env via viper)
//  3. the registry's Default-flagged columns
//
// Adapts columns.Column to output.Column so the existing renderers just work.
func resolveColumns(cmd *cobra.Command, resource string, kv map[string]string) []output.Column {
	var override []string
	if v := kv["columns"]; v != "" {
		override = splitColumns(v)
	} else if cfg := configFromCtx(cmd.Context()); cfg.Columns != nil {
		override = cfg.Columns[resource]
	}
	visible := columns.Resolve(resource, override)
	out := make([]output.Column, len(visible))
	for i, c := range visible {
		out[i] = output.Column{Header: c.Header, Extract: c.Extract}
	}
	return out
}

// splitColumns parses a comma-separated column list, trimming whitespace and
// dropping empties. Tolerant of trailing commas and surrounding spaces.
func splitColumns(raw string) []string {
	var out []string
	for _, n := range strings.Split(raw, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// resolveFormat picks the output format. Same precedence as resolveColumns:
//
//  1. positional kv["format"]
//  2. cfg.Format / NBCLI_FORMAT env
//  3. auto-detect from stdout (TTY → table, else json)
func resolveFormat(cmd *cobra.Command, io IO, kv map[string]string) (output.Format, error) {
	cfg := configFromCtx(cmd.Context())
	raw := cfg.Format
	if v := kv["format"]; v != "" {
		raw = v
	}
	explicit, err := output.Parse(raw)
	if err != nil {
		return "", err
	}
	return output.Resolve(explicit, io.Out), nil
}

// renderRows centralizes the boilerplate at the tail of every `show <resource>`
// command: pick the format, build the renderer, write to io.Out. kv carries
// any positional presentation overrides (format / columns).
func renderRows(cmd *cobra.Command, io IO, rows any, cols []output.Column, kv map[string]string) error {
	format, err := resolveFormat(cmd, io, kv)
	if err != nil {
		return err
	}
	r, err := output.New(format)
	if err != nil {
		return err
	}
	return r.Render(io.Out, cols, rows)
}

// errStopIteration is a sentinel used inside renderStreaming to signal the
// renderer told us to stop. Never propagated up to the user.
var errStopIteration = errors.New("render: stream consumer halted")

// renderStreaming is the fetchAll counterpart to renderRows. When the resolved
// format implements output.StreamingRenderer (json, yaml, tsv) and isn't
// table, rows write as they arrive from netbox.Iterate — memory stays O(1)
// and the user sees output incrementally. Otherwise falls back to the batch
// path (table needs all rows to align columns).
func renderStreaming[T any](
	cmd *cobra.Command,
	io IO,
	fetcher netbox.PageFetcher[T],
	iterOpts netbox.IterateOptions,
	cols []output.Column,
	kv map[string]string,
) error {
	format, err := resolveFormat(cmd, io, kv)
	if err != nil {
		return err
	}
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
