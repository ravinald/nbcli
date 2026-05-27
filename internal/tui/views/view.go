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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
)

// Header renders a view's top-line title in the shared accent color.
func Header(title string) string { return titleStyle.Render(title) }

// ErrorBlock formats an error nicely for the right pane.
func ErrorBlock(err error) string {
	return errorStyle.Render("error: ") + err.Error()
}

// Hint renders muted help text (keybind hints, "first fetch can take...", etc.).
func Hint(s string) string { return hintStyle.Render(s) }
