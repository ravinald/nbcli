// Package output renders typed slices as table/json/yaml/tsv.
//
// Resolution rules (highest wins):
//
//  1. --format flag
//  2. NBCLI_FORMAT env / config file value
//  3. Implicit: "table" when stdout is a TTY, "json" when piped/redirected
//
// (1) and (2) are passed in as the explicit argument to Resolve; (3) only
// fires when explicit == "".
package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Format names the supported renderers.
type Format string

// All supported output formats.
const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
	FormatTSV   Format = "tsv"
)

// All returns every format string for help text and flag completion.
func All() []string {
	return []string{string(FormatTable), string(FormatJSON), string(FormatYAML), string(FormatTSV)}
}

// Parse normalizes user input ("JSON", " yaml ", "tab") into a Format.
// Empty input is preserved as "" so Resolve can apply the implicit default.
func Parse(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return "", nil
	case "table", "tbl":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	case "tsv", "plain", "tab":
		return FormatTSV, nil
	default:
		return "", fmt.Errorf("output: unknown format %q (want one of %s)", s, strings.Join(All(), ","))
	}
}

// Resolve picks the final Format. explicit comes from flag/env/config (in that
// precedence, already merged by viper). When explicit is empty, Resolve checks
// whether out is a TTY: TTY → table, non-TTY → json.
func Resolve(explicit Format, out io.Writer) Format {
	if explicit != "" {
		return explicit
	}
	if isTerminal(out) {
		return FormatTable
	}
	return FormatJSON
}

// Renderer writes a slice of rows of any concrete type to w.
//
// We accept any here rather than a generic parameter because cobra commands
// dispatch on Format at runtime; the typed call site is one frame up.
type Renderer interface {
	Render(w io.Writer, columns []Column, rows any) error
}

// Column describes one column in tabular outputs. Extract pulls the cell value
// from one row — the function form keeps the package free of reflection.
type Column struct {
	Header  string
	Extract func(row any) string
}

// New returns the Renderer for the given Format. Unknown formats return an error.
func New(f Format) (Renderer, error) {
	switch f {
	case FormatTable:
		return tableRenderer{}, nil
	case FormatJSON:
		return jsonRenderer{}, nil
	case FormatYAML:
		return yamlRenderer{}, nil
	case FormatTSV:
		return tsvRenderer{}, nil
	default:
		return nil, fmt.Errorf("output: no renderer for %q", f)
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
