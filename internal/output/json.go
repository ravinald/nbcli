package output

import (
	"encoding/json"
	"io"
)

type jsonRenderer struct{}

// Render emits the rows slice as indented JSON. Columns are ignored — JSON
// callers get every field on the underlying struct.
func (jsonRenderer) Render(w io.Writer, _ []Column, rows any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(rows)
}
