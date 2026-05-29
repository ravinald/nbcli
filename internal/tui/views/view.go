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
	"net/url"
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

// NavMsg is fired by detail mode when the user presses a digit key over a
// foreign-key field. The root app.go switches active view to ViewName and
// asks that view to open detail for ID (via the OpenDetailByID method).
type NavMsg struct {
	ViewName string
	ID       int
}

// EscapeUpMsg is emitted by a view when Esc was pressed but the view had
// nothing internal to dismiss (no detail open, no search active, no committed
// filter). The shell uses this to return keyboard focus to the left viewport.
type EscapeUpMsg struct{}

// SizeMsg tells a view how much of the terminal its right-viewport pane
// covers (width and height in cells, after accounting for the left viewport,
// the status bar, and any chrome). Forwarded by the shell on tea.WindowSizeMsg.
type SizeMsg struct {
	Width  int
	Height int
}

// FetchOpts parameterizes a view's fetcher closure. baseView constructs one
// per load using its current pagination / search / drill-down state.
type FetchOpts struct {
	// Offset is the start index for paged fetches (0 = first page).
	Offset int
	// Limit is the page size. baseView sizes this to the viewport on first
	// SizeMsg. Zero means "fetcher's default".
	Limit int
	// Query is the free-text search term sent to Netbox as `?q=`. Empty
	// means no search.
	Query string
	// ID, when > 0, narrows the fetch to a single record by primary key.
	// Used by FK navigation so OpenDetailByID can find rows that aren't
	// in the currently-loaded page.
	ID int
}

// FetchResult is what a fetcher returns: the current page's rows plus the
// total count across all matching records (from Netbox's Page.Count).
type FetchResult[T any] struct {
	Rows  []T
	Total int
}

// FKRef is a parsed foreign-key reference. Built by DetailFKs when scanning
// a struct's fields. Detail mode uses the slice index + 1 as the user-facing
// digit key.
type FKRef struct {
	FieldKey string // e.g. "site"
	ViewName string // registered view name, e.g. "Sites"
	ID       int
	Name     string
}

// resourceToViewName maps Netbox API URL path segments to view names
// registered in tui.New(). Keep in sync with the views map there.
var resourceToViewName = map[string]string{
	"sites":            "Sites",
	"racks":            "Racks",
	"devices":          "Devices",
	"interfaces":       "Interfaces",
	"tenants":          "Tenants",
	"contacts":         "Contacts",
	"prefixes":         "Prefixes",
	"ip-addresses":     "IP Addresses",
	"vlans":            "VLANs",
	"vrfs":             "VRFs",
	"virtual-machines": "Virtual Machines",
	"clusters":         "Clusters",
}

// parseFKURL extracts the resource segment + numeric ID from a Netbox
// NestedRef URL: "https://nb.example.com/api/dcim/sites/42/" →
// ("sites", 42, true).
func parseFKURL(rawURL string) (resource string, id int, ok bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", 0, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	// Expected: api / <module> / <resource> / <id>
	if len(parts) < 4 || parts[0] != "api" {
		return "", 0, false
	}
	n, err := strconv.Atoi(parts[3])
	if err != nil {
		return "", 0, false
	}
	return parts[2], n, true
}

// DetailFKs returns the navigable foreign-key references in v's struct fields.
// Only NestedRef pointers with a parseable URL pointing at a known resource
// are returned, in field order.
func DetailFKs(v any) []FKRef {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	var out []FKRef
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		fv := rv.Field(i)
		if fv.IsZero() {
			continue
		}
		for fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				break
			}
			fv = fv.Elem()
		}
		nr, ok := fv.Interface().(netbox.NestedRef)
		if !ok {
			continue
		}
		resource, id, ok := parseFKURL(nr.URL)
		if !ok {
			continue
		}
		viewName, ok := resourceToViewName[resource]
		if !ok {
			continue
		}
		out = append(out, FKRef{
			FieldKey: detailFieldKey(f),
			ViewName: viewName,
			ID:       id,
			Name:     nr.Name,
		})
	}
	return out
}

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
	detailCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
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

// RenderDetail returns a key/value rendering of v with no cursor highlight.
// Equivalent to RenderDetailCursor(v, -1); kept as the zero-arg API.
func RenderDetail(v any) string {
	return RenderDetailCursor(v, -1)
}

// RenderDetailCursor returns a key/value rendering of v (any struct) using
// reflection. Non-zero fields only. NestedRef collapses to "Name (#id)";
// LabelValue to its Label. Pointer fields are deref'd. Navigable foreign
// keys are tagged with "[N]" markers (press the digit to jump) and the FK
// at fkCursor — when 0 ≤ fkCursor < len(DetailFKs(v)) — is highlighted
// with a ▸ prefix and accent color so the user sees what Enter will follow.
// Pass fkCursor=-1 for no highlight.
func RenderDetailCursor(v any, fkCursor int) string {
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
	// Build FK index by field key (0-indexed for cursor compare).
	fks := DetailFKs(v)
	fkByKey := make(map[string]int, len(fks))
	for i, fk := range fks {
		fkByKey[fk.FieldKey] = i
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
		fkIdx, isFK := fkByKey[key]
		if isFK {
			val = fmt.Sprintf("%s  %s", val, Hint(fmt.Sprintf("[%d]", fkIdx+1)))
		}
		keyText := fmt.Sprintf("%-*s:", width, key)
		if isFK && fkIdx == fkCursor {
			lines = append(lines, detailCursorStyle.Render("▸ "+keyText)+" "+val)
		} else {
			lines = append(lines, "  "+detailKeyStyle.Render(keyText)+" "+val)
		}
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
