package output

import (
	"io"

	"gopkg.in/yaml.v3"
)

type yamlRenderer struct{}

// Render emits the rows slice as YAML. Columns are ignored — YAML callers get
// every field on the underlying struct.
func (yamlRenderer) Render(w io.Writer, _ []Column, rows any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer func() { _ = enc.Close() }()
	return enc.Encode(rows)
}
