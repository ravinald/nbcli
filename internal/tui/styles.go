// Package tui hosts the bubbletea models that mirror the Netbox web UI.
//
// Layout:
//
//	+--------- nbcli ------------------------+
//	| Organization | Sites          ...      |
//	| DCIM         | --------------------    |
//	| > Sites      | NAME    SLUG    STATUS  |
//	|   Racks      | hq      hq      active  |
//	|   Devices    | ...                     |
//	| IPAM         |                         |
//	| Virtualization|                        |
//	| Tenancy      |                         |
//	| Plugins      |                         |
//	+----------------------------------------+
//
// The shell is a Model that owns the sidebar and routes input to the active
// view (a tea.Model). Each view is implemented next to its resource (sites.go,
// devices.go, ...) so the package stays navigable.
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

// Styles bundles the shared lipgloss styles. New a copy and tweak as needed.
type Styles struct {
	App           lipgloss.Style
	Sidebar       lipgloss.Style
	SidebarItem   lipgloss.Style
	SidebarActive lipgloss.Style
	Main          lipgloss.Style
	Title         lipgloss.Style
	StatusBar     lipgloss.Style
}

// DefaultStyles returns the base look. Views are free to derive from it.
func DefaultStyles() Styles {
	s := Styles{}
	s.App = lipgloss.NewStyle()
	s.Sidebar = lipgloss.NewStyle().
		Width(22).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(Palette.Subtle)
	s.SidebarItem = lipgloss.NewStyle().Foreground(Palette.Faint)
	s.SidebarActive = lipgloss.NewStyle().
		Foreground(Palette.Primary).
		Bold(true)
	s.Main = lipgloss.NewStyle().Padding(1, 2)
	s.Title = lipgloss.NewStyle().
		Foreground(Palette.Primary).
		Bold(true).
		Padding(0, 0, 1, 0)
	s.StatusBar = lipgloss.NewStyle().
		Foreground(Palette.Faint).
		Padding(0, 1)
	return s
}
