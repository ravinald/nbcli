package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/tui/views"
)

// Section is one entry in the sidebar — a top-level area of the Netbox web UI.
type Section struct {
	Title string
	Items []string
}

// DefaultSections mirrors the Netbox web nav.
func DefaultSections() []Section {
	return []Section{
		{Title: "Organization", Items: []string{"Sites", "Tenants", "Contacts"}},
		{Title: "DCIM", Items: []string{"Racks", "Devices", "Interfaces"}},
		{Title: "IPAM", Items: []string{"Prefixes", "IP Addresses", "VLANs", "VRFs"}},
		{Title: "Virtualization", Items: []string{"Virtual Machines", "Clusters"}},
		{Title: "Plugins", Items: []string{"Passthrough"}},
	}
}

// flatEntry maps a single integer cursor to (section, item) for keyboard nav.
type flatEntry struct {
	section int
	item    int
}

// Pane identifies which UI region owns the keyboard. Arrow keys, Enter, and
// most other input route to the focused pane.
type Pane int

// Pane values.
const (
	PaneSidebar Pane = iota
	PaneView
)

// Model is the root bubbletea model. It owns the sidebar selection and routes
// keyboard input / async messages to the currently active resource view.
type Model struct {
	client   *netbox.Client
	styles   Styles
	sections []Section

	flatIndex int
	flat      []flatEntry

	// views is the registry of resource views keyed by sidebar item name.
	// Missing entries fall back to the placeholder pane.
	views  map[string]views.View
	active views.View

	// pane is the currently-focused region (sidebar or active view).
	pane Pane

	showHelp bool

	width, height int
}

// New constructs the root model with the given Netbox client.
//
// v0.1 wires real views for Tenants and Contacts; everything else is a
// placeholder pane that says so.
func New(client *netbox.Client) Model {
	sections := DefaultSections()
	var flat []flatEntry
	for si, s := range sections {
		for ii := range s.Items {
			flat = append(flat, flatEntry{section: si, item: ii})
		}
	}
	m := Model{
		client:   client,
		styles:   DefaultStyles(),
		sections: sections,
		flat:     flat,
		views: map[string]views.View{
			"Tenants":          views.NewTenants(client),
			"Contacts":         views.NewContacts(client),
			"Racks":            views.NewRacks(client),
			"Devices":          views.NewDevices(client),
			"Interfaces":       views.NewInterfaces(client),
			"Prefixes":         views.NewPrefixes(client),
			"IP Addresses":     views.NewIPAddresses(client),
			"VLANs":            views.NewVLANs(client),
			"VRFs":             views.NewVRFs(client),
			"Virtual Machines": views.NewVMs(client),
			"Clusters":         views.NewClusters(client),
		},
		pane: PaneSidebar,
	}
	m.active = m.lookupView()
	return m
}

// currentItem returns the sidebar item name under the cursor.
func (m Model) currentItem() string {
	if len(m.flat) == 0 {
		return ""
	}
	cur := m.flat[m.flatIndex]
	return m.sections[cur.section].Items[cur.item]
}

// lookupView returns the view registered for the current sidebar item, or nil.
func (m Model) lookupView() views.View {
	return m.views[m.currentItem()]
}

// Init kicks off the first view's data fetch if one is registered.
func (m Model) Init() tea.Cmd {
	if m.active != nil {
		return m.active.Focus()
	}
	return nil
}

// Update routes keyboard navigation, FK-nav messages from views, and async
// data-load results. Key routing is focus-aware: when the sidebar is active,
// arrow keys move section items; when a view is active, they reach the
// view's table widget.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case views.NavMsg:
		m.pane = PaneView
		return m.navigateTo(msg.ViewName, msg.ID)
	case views.EscapeUpMsg:
		m.pane = PaneSidebar
		return m, nil
	case tea.KeyMsg:
		// Global keys, regardless of pane focus.
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		}
		if m.showHelp {
			return m, nil
		}
		if m.pane == PaneSidebar {
			return m.updateSidebar(msg)
		}
		// PaneView: fall through to the forward-to-active block below.
	}
	if m.active != nil {
		next, cmd := m.active.Update(msg)
		if v, ok := next.(views.View); ok {
			m.active = v
		}
		return m, cmd
	}
	return m, nil
}

// updateSidebar handles keys while the sidebar holds focus.
func (m Model) updateSidebar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.flatIndex > 0 {
			m.flatIndex--
			return m.swapActive()
		}
	case "down", "j":
		if m.flatIndex < len(m.flat)-1 {
			m.flatIndex++
			return m.swapActive()
		}
	case "tab", "ctrl+n", "]":
		if m.flatIndex < len(m.flat)-1 {
			m.flatIndex++
			return m.swapActive()
		}
	case "shift+tab", "ctrl+p", "[":
		if m.flatIndex > 0 {
			m.flatIndex--
			return m.swapActive()
		}
	case "enter", "right", "l":
		// Hand keyboard focus to the active view.
		m.pane = PaneView
		if m.active != nil {
			return m, m.active.Focus()
		}
	}
	return m, nil
}

// swapActive points m.active at the view for the newly-selected sidebar item
// and returns its Focus cmd (which fetches data on first activation).
func (m Model) swapActive() (tea.Model, tea.Cmd) {
	m.active = m.lookupView()
	if m.active == nil {
		return m, nil
	}
	return m, m.active.Focus()
}

// navigateTo switches the sidebar selection and active view to the named
// view, then asks the target (if it supports it) to open detail for id.
// Used by FK-navigation: pressing a digit in detail mode emits a
// views.NavMsg that lands here.
func (m Model) navigateTo(viewName string, id int) (tea.Model, tea.Cmd) {
	for i, fe := range m.flat {
		if m.sections[fe.section].Items[fe.item] != viewName {
			continue
		}
		m.flatIndex = i
		m.active = m.views[viewName]
		if m.active == nil {
			return m, nil
		}
		cmds := []tea.Cmd{m.active.Focus()}
		if opener, ok := m.active.(interface{ OpenDetailByID(int) tea.Cmd }); ok {
			if c := opener.OpenDetailByID(id); c != nil {
				cmds = append(cmds, c)
			}
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// View renders the two-pane shell with the active view's body on the right.
// When the help pane is up, it overlays the main pane.
func (m Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	sidebar := m.renderSidebar()
	var main string
	if m.showHelp {
		main = m.styles.Main.Render(m.styles.Title.Render("Help") + "\n\n" + helpText())
	} else {
		main = m.renderMain()
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	status := m.styles.StatusBar.Render(m.statusLine())
	return lipgloss.JoinVertical(lipgloss.Left, body, status)
}

// statusLine builds the bottom hint based on which pane has focus.
func (m Model) statusLine() string {
	if m.showHelp {
		return "? close help · q quit"
	}
	if m.pane == PaneSidebar {
		return "↑/↓ section · → / enter focus view · ? help · q quit"
	}
	return "↑/↓ rows · enter detail · / search · esc back to sidebar · ? help · q quit"
}

// helpText is the canonical keybind reference shown by `?`.
func helpText() string {
	return "Sidebar (focus by default)\n" +
		"  ↑ / ↓ · k / j      next / previous section item\n" +
		"  Tab / Shift+Tab    same; ] / [ are aliases\n" +
		"  Enter / → / l      focus into the active view\n" +
		"\n" +
		"View (when focused)\n" +
		"  ↑ / ↓ · k / j      move between table rows\n" +
		"  Enter              show detail of selected row\n" +
		"  /                  open search input\n" +
		"  Esc                close detail, clear filter, or return to sidebar\n" +
		"\n" +
		"Detail view\n" +
		"  1 - 9              follow the FK marked [N] (jump to that resource)\n" +
		"  Esc                back to list\n" +
		"\n" +
		"Search\n" +
		"  Enter              commit filter (keep visible, exit input)\n" +
		"  Esc                cancel (clear filter, exit input)\n" +
		"\n" +
		"Global\n" +
		"  ?                  toggle this help\n" +
		"  q / Ctrl+C         quit"
}

func (m Model) renderSidebar() string {
	var b strings.Builder
	cursor := 0
	for si, s := range m.sections {
		b.WriteString(m.styles.Title.Render(s.Title))
		b.WriteString("\n")
		for _, item := range s.Items {
			line := "  " + item
			style := m.styles.SidebarItem
			if cursor == m.flatIndex {
				line = "▸ " + item
				style = m.styles.SidebarActive
			}
			b.WriteString(style.Render(line))
			b.WriteString("\n")
			cursor++
		}
		if si < len(m.sections)-1 {
			b.WriteString("\n")
		}
	}
	return m.styles.Sidebar.Render(b.String())
}

func (m Model) renderMain() string {
	if m.active != nil {
		return m.styles.Main.Render(m.active.View())
	}
	name := m.currentItem()
	placeholder := m.styles.Title.Render(name) + "\n\n" +
		"(view not yet implemented in v0.1)\n\nTarget: " + m.client.BaseURL()
	return m.styles.Main.Render(placeholder)
}

// Run starts the bubbletea program until quit. Called from `nbcli tui`.
func Run(client *netbox.Client) error {
	p := tea.NewProgram(New(client), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
