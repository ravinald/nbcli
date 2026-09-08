// stream.go adds row-at-a-time rendering for output formats that don't need
// the entire input materialized. Today: JSON, YAML, TSV. Table needs full
// data to align columns and falls back to the batch Renderer.Render path.
//
// Streaming matters for `nbcli show <resource> limit 0` against deployments
// with very large result sets: the user sees rows as they arrive, and memory
// stays bounded to one row at a time rather than the full Netbox response.

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strings"

	"gopkg.in/yaml.v3"
)

// StreamingRenderer is the optional capability for formats that can write
// rows incrementally. Callers type-assert from Renderer; when the assertion
// fails (e.g. table) they collect the full slice and call Render instead.
type StreamingRenderer interface {
	Renderer
	// Stream consumes rows from seq one at a time and writes immediately.
	// Returning false from yield stops iteration; the renderer should treat
	// that as a normal end-of-input.
	Stream(w io.Writer, cols []Column, seq iter.Seq[any]) error
}

// Stream emits a JSON array element-by-element. Output is byte-identical
// to Render for the same input, so consumers can't tell the difference.
func (jsonRenderer) Stream(w io.Writer, _ []Column, seq iter.Seq[any]) error {
	if _, err := io.WriteString(w, "[\n"); err != nil {
		return err
	}
	first := true
	for row := range seq {
		if !first {
			if _, err := io.WriteString(w, ",\n"); err != nil {
				return err
			}
		}
		first = false
		if _, err := io.WriteString(w, "  "); err != nil {
			return err
		}
		b, err := json.MarshalIndent(row, "  ", "  ")
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n]\n")
	return err
}

// Stream emits a YAML sequence element-by-element. Each row is marshaled
// independently and prefixed with "- "; subsequent lines indent by 2 spaces.
// Output matches the batch yaml.Encoder output for the same input.
func (yamlRenderer) Stream(w io.Writer, _ []Column, seq iter.Seq[any]) error {
	for row := range seq {
		b, err := yaml.Marshal(row)
		if err != nil {
			return err
		}
		lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		for i, line := range lines {
			prefix := "  "
			if i == 0 {
				prefix = "- "
			}
			if _, err := fmt.Fprintln(w, prefix+line); err != nil {
				return err
			}
		}
	}
	return nil
}

// Stream emits TSV row-by-row after the header line. Naturally streaming;
// each row is independent.
func (tsvRenderer) Stream(w io.Writer, cols []Column, seq iter.Seq[any]) error {
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = strings.ToLower(c.Header)
	}
	if _, err := fmt.Fprintln(w, strings.Join(headers, "\t")); err != nil {
		return err
	}
	for row := range seq {
		cells := make([]string, len(cols))
		for j, c := range cols {
			cells[j] = sanitizeTSV(c.Extract(row))
		}
		if _, err := fmt.Fprintln(w, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return nil
}
