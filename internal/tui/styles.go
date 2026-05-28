// Package tui hosts the bubbletea models that mirror the Netbox web UI.
//
// Layout (rendered by app.go.View):
//
//	╭─ Navigation ─────╮╭─ Results ────────────────────────╮
//	│ Organization     ││ Interfaces                       │
//	│  ▸ Sites         ││ ─────────────────────────────    │
//	│    Tenants       ││ ID  DEVICE     NAME  TYPE …      │
//	│ DCIM             ││ 330 aloha-dev  tail  virtual …   │
//	│    Racks         ││ …                                │
//	╰──────────────────╯╰──────────────────────────────────╯
//	 ↑/↓ section · tab focus right · ? help · q quit
//
// Each viewport gets a bordered box; the focused one is bright white, the
// inactive one dim grey (same pattern as bodega's TUI).
package tui

import "github.com/charmbracelet/lipgloss"

// Palette is the shared color set. Pulled into one place so theming is easy.
var Palette = struct {
	Primary  lipgloss.Color
	Subtle   lipgloss.Color
	Accent   lipgloss.Color
	Danger   lipgloss.Color
	Faint    lipgloss.Color
	BG       lipgloss.Color
	BGSubtle lipgloss.Color
}{
	Primary:  lipgloss.Color("#7D56F4"),
	Subtle:   lipgloss.Color("#555"),
	Accent:   lipgloss.Color("#04B575"),
	Danger:   lipgloss.Color("#E45757"),
	Faint:    lipgloss.Color("#888"),
	BG:       lipgloss.Color("#1A1A1A"),
	BGSubtle: lipgloss.Color("#222"),
}

// Viewport border colors. White = focused, grey = inactive.
var (
	colorFocusedBorder   = lipgloss.Color("15")  // bright white
	colorUnfocusedBorder = lipgloss.Color("240") // dim grey
)

// PaneStyle returns the viewport border style for a given focus state.
// Apply Width() and Height() to it at render time to size each viewport.
func PaneStyle(focused bool) lipgloss.Style {
	border := colorUnfocusedBorder
	if focused {
		border = colorFocusedBorder
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
}

// Styles bundles the shared lipgloss styles that aren't pane borders.
type Styles struct {
	SidebarItem   lipgloss.Style
	SidebarActive lipgloss.Style
	Title         lipgloss.Style
	StatusBar     lipgloss.Style
}

// DefaultStyles returns the base look. Views are free to derive from it.
func DefaultStyles() Styles {
	return Styles{
		SidebarItem: lipgloss.NewStyle().Foreground(Palette.Faint),
		SidebarActive: lipgloss.NewStyle().
			Foreground(Palette.Primary).
			Bold(true),
		Title: lipgloss.NewStyle().
			Foreground(Palette.Primary).
			Bold(true).
			Padding(0, 0, 1, 0),
		StatusBar: lipgloss.NewStyle().
			Foreground(Palette.Faint).
			Padding(0, 1),
	}
}
