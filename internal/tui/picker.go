package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ravinald/nbcli/internal/columns"
)

// pickerSavedMsg is fired when the user confirms the column picker. The
// root model handles it by persisting config and asking the active view
// to RefreshColumns.
type pickerSavedMsg struct {
	resource string
	columns  []string
}

// pickerCancelledMsg signals the user dismissed the picker without saving.
// Mostly a marker for the root model to redraw status appropriately.
type pickerCancelledMsg struct{}

// pickerModel is a modal popup for picking which columns to show. It owns:
//
//   - The list of available columns for a resource (from the registry)
//   - A selection map (column name → visible?)
//   - An ordering slice (the order the user will see — currently visible
//     first in their order, then unselected after)
//   - A cursor index into that ordering
//
// When the user presses Enter, the picker emits pickerSavedMsg with the
// final ordered list of selected names. Esc fires pickerCancelledMsg.
type pickerModel struct {
	open     bool
	resource string
	set      columns.Set
	order    []string
	selected map[string]bool
	cursor   int
}

func newPickerModel() pickerModel { return pickerModel{} }

// Active reports whether the picker is currently open and owns the keyboard.
func (p pickerModel) Active() bool { return p.open }

// Open initializes the picker for resource, seeded with the currently-visible
// names (in their order). Unknown / no-longer-existing names are skipped;
// columns not in current are appended in registry order so the user can
// turn them on.
func (p *pickerModel) Open(resource string, current []string) {
	set, ok := columns.Registry()[resource]
	if !ok {
		return
	}
	p.open = true
	p.resource = resource
	p.set = set
	p.selected = make(map[string]bool, len(set.Columns))
	p.order = make([]string, 0, len(set.Columns))
	known := make(map[string]bool, len(set.Columns))
	for _, c := range set.Columns {
		known[c.Name] = true
	}
	seen := make(map[string]bool, len(set.Columns))
	for _, n := range current {
		if known[n] && !seen[n] {
			p.order = append(p.order, n)
			p.selected[n] = true
			seen[n] = true
		}
	}
	for _, c := range set.Columns {
		if !seen[c.Name] {
			p.order = append(p.order, c.Name)
			p.selected[c.Name] = false
		}
	}
	p.cursor = 0
}

// Close dismisses the picker without firing any message.
func (p *pickerModel) Close() { p.open = false }

// Update handles a key event while the picker is open. Returns the
// possibly-updated picker and an optional Cmd that fires a saved/cancelled
// message.
func (p pickerModel) Update(msg tea.KeyMsg) (pickerModel, tea.Cmd) {
	if !p.open {
		return p, nil
	}
	switch msg.String() {
	case "up", "k":
		if p.cursor > 0 {
			p.cursor--
		}
	case "down", "j":
		if p.cursor < len(p.order)-1 {
			p.cursor++
		}
	case " ", "x":
		if p.cursor >= 0 && p.cursor < len(p.order) {
			n := p.order[p.cursor]
			p.selected[n] = !p.selected[n]
		}
	case "K", "ctrl+up":
		// Move current item up in display order.
		if p.cursor > 0 {
			p.order[p.cursor-1], p.order[p.cursor] = p.order[p.cursor], p.order[p.cursor-1]
			p.cursor--
		}
	case "J", "ctrl+down":
		if p.cursor < len(p.order)-1 {
			p.order[p.cursor+1], p.order[p.cursor] = p.order[p.cursor], p.order[p.cursor+1]
			p.cursor++
		}
	case "enter":
		var visible []string
		for _, n := range p.order {
			if p.selected[n] {
				visible = append(visible, n)
			}
		}
		resource := p.resource
		p.Close()
		return p, func() tea.Msg {
			return pickerSavedMsg{resource: resource, columns: visible}
		}
	case "esc":
		p.Close()
		return p, func() tea.Msg { return pickerCancelledMsg{} }
	}
	return p, nil
}

// pickerStyles holds the lipgloss styles for the picker. Tuned to feel like
// the rest of the TUI (rounded border, accent purple for cursor).
var (
	pickerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7D56F4")).
				MarginBottom(1)
	pickerCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)
	pickerDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888")).
			Italic(true)
)

// View renders the picker as a vertical block. The shell embeds it in the
// right-viewport pane (replacing the active view's render while open).
func (p pickerModel) View() string {
	if !p.open {
		return ""
	}
	var b strings.Builder
	b.WriteString(pickerTitleStyle.Render("Configure columns — " + p.resource))
	b.WriteString("\n")

	headersByName := make(map[string]string, len(p.set.Columns))
	defaultsByName := make(map[string]bool, len(p.set.Columns))
	for _, c := range p.set.Columns {
		headersByName[c.Name] = c.Header
		defaultsByName[c.Name] = c.Default
	}

	for i, n := range p.order {
		mark := "[ ]"
		if p.selected[n] {
			mark = "[x]"
		}
		def := ""
		if defaultsByName[n] {
			def = pickerDimStyle.Render(" (default)")
		}
		line := fmt.Sprintf("  %s %-18s %s%s", mark, n, headersByName[n], def)
		if i == p.cursor {
			line = pickerCursorStyle.Render("▸ "+mark+" "+padRight(n, 18)+" "+headersByName[n]) + def
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(pickerDimStyle.Render("space toggle · K/J reorder · enter save · esc cancel"))
	return b.String()
}

// padRight pads s to width n with trailing spaces. Used in View() so the
// cursor-highlighted line aligns with the un-highlighted lines.
func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
