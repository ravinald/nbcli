// Package views holds the bubbletea models for each Netbox resource pane.
//
// Lifecycle:
//
//   - The root tui.Model registers one View per sidebar item it knows how to render.
//   - When a view becomes active, the shell calls Focus(). The view returns a
//     tea.Cmd that fetches data on first activation, then no-ops on re-focus.
//   - Each view is a tea.Model: Update returns the next model + optional Cmd.
//
// Views deliberately use *background* contexts when fetching. The TUI's
// "cancel current operation" is bubbletea quitting the program; we don't
// thread cancellation per-view in v0.1.
package views

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ravinald/nbcli/internal/netbox"
)

// View is what every resource view in the TUI implements. It's tea.Model with
// a Title used by the shell for breadcrumbs and a Focus hook that fires when
// the view becomes active.
type View interface {
	tea.Model
	// Title is the human label rendered above the view's body.
	Title() string
	// Focus is called when the view becomes active. Returns a tea.Cmd that
	// typically kicks off (or refreshes) data fetching. Idempotent.
	Focus() tea.Cmd
}

// ErrMsg wraps an error so it flows through bubbletea's Cmd channel.
type ErrMsg struct{ Err error }

// Style tokens shared across views. Keep in one place so the right pane
// stays visually consistent regardless of which resource is on screen.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1)
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E45757")).
			Bold(true)
	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888")).
			Italic(true)
	detailKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)
)

// Header renders a view's top-line title in the shared accent color.
func Header(title string) string { return titleStyle.Render(title) }

// ErrorBlock formats an error nicely for the right pane.
func ErrorBlock(err error) string {
	return errorStyle.Render("error: ") + err.Error()
}

// Hint renders muted help text (keybind hints, "first fetch can take...", etc.).
func Hint(s string) string { return hintStyle.Render(s) }

// nestedName returns the Name of a NestedRef, "" if nil. Used pervasively by
// row mappers to flatten foreign-key references into table cells.
func nestedName(n *netbox.NestedRef) string {
	if n == nil {
		return ""
	}
	return n.Name
}

// RenderDetail returns a key/value rendering of v (any struct) using reflection.
// Non-zero fields only. NestedRef collapses to "Name (#id)"; LabelValue to its
// Label. Pointer fields are deref'd. Used by every view's detail pane so we
// don't write a hand-rolled detail layout per resource.
func RenderDetail(v any) string {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "(nil)"
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Sprintf("%v", v)
	}
	width := 0
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() || rv.Field(i).IsZero() {
			continue
		}
		if k := detailFieldKey(f); len(k) > width {
			width = len(k)
		}
	}
	var lines []string
	for i := 0; i < rv.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		fv := rv.Field(i)
		if fv.IsZero() {
			continue
		}
		key := detailFieldKey(f)
		val := detailFieldValue(fv)
		lines = append(lines, fmt.Sprintf("%s %s",
			detailKeyStyle.Render(fmt.Sprintf("%-*s", width, key)+":"), val))
	}
	return strings.Join(lines, "\n")
}

// detailFieldKey returns the JSON tag name if present, else the field name.
// Strips ",omitempty" / ",string" suffixes.
func detailFieldKey(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" || tag == "-" {
		return f.Name
	}
	if i := strings.IndexByte(tag, ','); i > 0 {
		return tag[:i]
	}
	return tag
}

// detailFieldValue stringifies a reflect.Value for the detail pane. Pulled out
// so RenderDetail stays readable.
func detailFieldValue(v reflect.Value) string {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		if nr, ok := v.Interface().(netbox.NestedRef); ok {
			if nr.Name != "" {
				return fmt.Sprintf("%s (#%d)", nr.Name, nr.ID)
			}
			return fmt.Sprintf("#%d", nr.ID)
		}
		if lv, ok := v.Interface().(netbox.LabelValue); ok {
			if lv.Label != "" {
				return lv.Label
			}
			return lv.Value
		}
		b, _ := json.Marshal(v.Interface())
		return string(b)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.String:
		return v.String()
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}
