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

	width, height int
	status        string
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
		status: "tab/⇧tab sidebar · ↑/↓ rows · enter detail · esc back · q quit",
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

// Update routes keyboard navigation, then forwards every other message
// (including async data-load results) to the active view.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "ctrl+n", "]":
			if m.flatIndex < len(m.flat)-1 {
				m.flatIndex++
				return m.swapActive()
			}
			return m, nil
		case "shift+tab", "ctrl+p", "[":
			if m.flatIndex > 0 {
				m.flatIndex--
				return m.swapActive()
			}
			return m, nil
		}
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

// swapActive points m.active at the view for the newly-selected sidebar item
// and returns its Focus cmd (which fetches data on first activation).
func (m Model) swapActive() (tea.Model, tea.Cmd) {
	m.active = m.lookupView()
	if m.active == nil {
		return m, nil
	}
	return m, m.active.Focus()
}

// View renders the two-pane shell with the active view's body on the right.
func (m Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	sidebar := m.renderSidebar()
	main := m.renderMain()
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	status := m.styles.StatusBar.Render(m.status)
	return lipgloss.JoinVertical(lipgloss.Left, body, status)
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
