package output

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
)

type tableRenderer struct{}

// Render writes a left-aligned, padded table using text/tabwriter. We accept
// rows as any so the caller can pass []Site, []Device, etc. without converting.
func (tableRenderer) Render(w io.Writer, cols []Column, rows any) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()

	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = strings.ToUpper(c.Header)
	}
	if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
		return err
	}

	v := reflect.ValueOf(rows)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("table: rows must be a slice, got %T", rows)
	}
	for i := 0; i < v.Len(); i++ {
		row := v.Index(i).Interface()
		cells := make([]string, len(cols))
		for j, c := range cols {
			cells[j] = c.Extract(row)
		}
		if _, err := fmt.Fprintln(tw, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return nil
}
