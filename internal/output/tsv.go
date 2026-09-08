package output

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

type tsvRenderer struct{}

// Render writes tab-separated values with a header row. Awk-friendly:
// `nbcli show sites --format tsv | awk -F'\t' '{print $1}'`.
func (tsvRenderer) Render(w io.Writer, cols []Column, rows any) error {
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = strings.ToLower(c.Header)
	}
	if _, err := fmt.Fprintln(w, strings.Join(headers, "\t")); err != nil {
		return err
	}

	v := reflect.ValueOf(rows)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("tsv: rows must be a slice, got %T", rows)
	}
	for i := 0; i < v.Len(); i++ {
		row := v.Index(i).Interface()
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

// sanitizeTSV replaces embedded tabs/newlines so each row stays single-line.
func sanitizeTSV(s string) string {
	r := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")
	return r.Replace(s)
}
