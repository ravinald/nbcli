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

// Viewport identifies which UI region owns the keyboard.
//
//   - LeftViewport   — navigation (the section/item list)
//   - RightViewport  — results and details (the active resource view)
//
// Tab toggles between them; arrow keys move within the focused viewport.
type Viewport int

// Viewport values.
const (
	LeftViewport Viewport = iota
	RightViewport
)

// leftPaneTotalWidth is the total horizontal cells the left viewport consumes
// (content + border + padding). Tuned so "Virtual Machines" + cursor fits
// without truncation.
const leftPaneTotalWidth = 26

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

	// focused is the currently-active viewport (left or right).
	focused Viewport

	showHelp bool

	width, height int

	// rightContentW/H cache the right viewport's content dimensions. Computed
	// by resizeChildren on tea.WindowSizeMsg; reused when swapActive or
	// navigateTo activates a different view so the new view's table sizes
	// correctly without waiting for another window resize.
	rightContentW, rightContentH int
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
			"Sites":            views.NewSites(client),
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
		focused: LeftViewport,
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
		nm, cmd := m.resizeChildren()
		return nm, cmd
	case views.NavMsg:
		m.focused = RightViewport
		return m.navigateTo(msg.ViewName, msg.ID)
	case views.EscapeUpMsg:
		m.focused = LeftViewport
		return m, nil
	case tea.KeyMsg:
		// Global keys, regardless of focus.
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "tab", "shift+tab":
			// Toggle which viewport owns the keyboard.
			if m.focused == LeftViewport {
				m.focused = RightViewport
				if m.active != nil {
					return m, m.active.Focus()
				}
			} else {
				m.focused = LeftViewport
			}
			return m, nil
		}
		if m.showHelp {
			return m, nil
		}
		if m.focused == LeftViewport {
			return m.updateLeftViewport(msg)
		}
		// RightViewport: fall through to the forward-to-active block below.
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

// updateLeftViewport handles keys while the left (navigation) viewport is focused.
func (m Model) updateLeftViewport(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "enter", "right", "l":
		// Equivalent to Tab — hand focus to the right viewport.
		m.focused = RightViewport
		if m.active != nil {
			return m, m.active.Focus()
		}
	}
	return m, nil
}

// resizeChildren recomputes the right viewport's content dimensions and
// forwards them to the active view. Fires on tea.WindowSizeMsg.
func (m Model) resizeChildren() (Model, tea.Cmd) {
	frameW, frameH := PaneStyle(false).GetFrameSize()
	w := (m.width - leftPaneTotalWidth) - frameW
	h := (m.height - 1) - frameH // -1 for the status bar line
	if w < 20 {
		w = 20
	}
	if h < 5 {
		h = 5
	}
	m.rightContentW = w
	m.rightContentH = h
	return m.sizeActive()
}

// sizeActive forwards the cached right-viewport dimensions to m.active.
// Called both from resizeChildren (on window resize) and from swapActive /
// navigateTo (on view change). No-op until at least one WindowSizeMsg has
// arrived.
func (m Model) sizeActive() (Model, tea.Cmd) {
	if m.active == nil || m.rightContentW <= 0 {
		return m, nil
	}
	next, cmd := m.active.Update(views.SizeMsg{
		Width:  m.rightContentW,
		Height: m.rightContentH,
	})
	if v, ok := next.(views.View); ok {
		m.active = v
	}
	return m, cmd
}

// swapActive points m.active at the view for the newly-selected sidebar item.
// Returns Focus() (first call kicks off the data fetch) batched with a
// SizeMsg so the new view's table is sized correctly without waiting for the
// next terminal resize.
func (m Model) swapActive() (tea.Model, tea.Cmd) {
	m.active = m.lookupView()
	if m.active == nil {
		return m, nil
	}
	cmds := []tea.Cmd{m.active.Focus()}
	nm, sizeCmd := m.sizeActive()
	m = nm
	if sizeCmd != nil {
		cmds = append(cmds, sizeCmd)
	}
	return m, tea.Batch(cmds...)
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
		nm, sizeCmd := m.sizeActive()
		m = nm
		if sizeCmd != nil {
			cmds = append(cmds, sizeCmd)
		}
		if opener, ok := m.active.(interface{ OpenDetailByID(int) tea.Cmd }); ok {
			if c := opener.OpenDetailByID(id); c != nil {
				cmds = append(cmds, c)
			}
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// View renders the two-viewport shell. Each viewport is wrapped in a bordered
// box (white when focused, grey when not), sized to fill the terminal.
func (m Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	frameW, frameH := PaneStyle(false).GetFrameSize()
	leftContentW := leftPaneTotalWidth - frameW
	rightContentW := m.width - leftPaneTotalWidth - frameW
	contentH := (m.height - 1) - frameH // status bar at the bottom

	leftPane := PaneStyle(m.focused == LeftViewport).
		Width(leftContentW).
		Height(contentH).
		Render(m.renderSidebar())

	var rightContent string
	if m.showHelp {
		rightContent = m.styles.Title.Render("Help") + "\n\n" + helpText()
	} else {
		rightContent = m.renderMain()
	}
	rightPane := PaneStyle(m.focused == RightViewport).
		Width(rightContentW).
		Height(contentH).
		Render(rightContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	status := m.styles.StatusBar.Render(m.statusLine())
	return lipgloss.JoinVertical(lipgloss.Left, body, status)
}

// statusLine builds the bottom hint based on which viewport has focus.
func (m Model) statusLine() string {
	if m.showHelp {
		return "? close help · q quit"
	}
	if m.focused == LeftViewport {
		return "↑/↓ section · tab focus right · ? help · q quit"
	}
	return "↑/↓ rows · enter detail · / search · tab focus left · ? help · q quit"
}

// helpText is the canonical keybind reference shown by `?`.
func helpText() string {
	return "Viewport focus\n" +
		"  Tab / Shift+Tab    toggle between left (nav) and right (results)\n" +
		"  Enter / → / l      from left → focus right\n" +
		"  Esc                from right → focus left (when nothing to dismiss)\n" +
		"\n" +
		"Left viewport (navigation)\n" +
		"  ↑ / ↓ · k / j      select a section item\n" +
		"\n" +
		"Right viewport (results)\n" +
		"  ↑ / ↓ · k / j      move between table rows\n" +
		"  Enter              show detail of selected row\n" +
		"  /                  open search input\n" +
		"  Esc                close detail, or clear filter\n" +
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

// renderSidebar builds the sidebar content. PaneStyle wraps it with a border
// at View() time so styling stays in one place.
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
	return b.String()
}

// renderMain builds the right viewport content. PaneStyle wraps it at View() time.
func (m Model) renderMain() string {
	if m.active != nil {
		return m.active.View()
	}
	name := m.currentItem()
	return m.styles.Title.Render(name) + "\n\n" +
		"(view not yet implemented in v0.1)\n\nTarget: " + m.client.BaseURL()
}

// Run starts the bubbletea program until quit. Called from `nbcli tui`.
func Run(client *netbox.Client) error {
	p := tea.NewProgram(New(client), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
