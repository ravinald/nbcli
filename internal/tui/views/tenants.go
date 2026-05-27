package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ravinald/nbcli/internal/netbox"
)

// TenantsView lists tenants in a scrollable table. Data loads on first Focus.
type TenantsView struct {
	client  *netbox.Client
	table   table.Model
	loaded  bool
	loading bool
	err     error
}

// tenantsLoadedMsg is private to this view so it can't collide with sibling views.
type tenantsLoadedMsg struct{ rows []netbox.Tenant }

// NewTenants constructs the view bound to client. No fetch happens until Focus.
func NewTenants(client *netbox.Client) *TenantsView {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 26},
		{Title: "Slug", Width: 26},
		{Title: "Group", Width: 22},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	t.SetStyles(s)
	return &TenantsView{client: client, table: t}
}

// Title is the human label rendered above the view body.
func (v *TenantsView) Title() string { return "Tenants" }

// Init is required by tea.Model; loading is driven by Focus.
func (v *TenantsView) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called.
func (v *TenantsView) Focus() tea.Cmd {
	if v.loaded || v.loading {
		return nil
	}
	v.loading = true
	return v.fetch()
}

func (v *TenantsView) fetch() tea.Cmd {
	return func() tea.Msg {
		rows, err := netbox.ListAll(context.Background(),
			v.client.TenantsFetcher(netbox.ListTenantsOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return tenantsLoadedMsg{rows: rows}
	}
}

// Update routes async load messages and forwards everything else to the table.
func (v *TenantsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tenantsLoadedMsg:
		v.loading = false
		v.loaded = true
		v.err = nil
		rows := make([]table.Row, 0, len(m.rows))
		for _, t := range m.rows {
			group := ""
			if t.Group != nil {
				group = t.Group.Name
			}
			rows = append(rows, table.Row{
				strconv.Itoa(t.ID),
				t.Name,
				t.Slug,
				group,
			})
		}
		v.table.SetRows(rows)
	case ErrMsg:
		v.loading = false
		v.err = m.Err
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

// View renders the title, then status (loading/error/empty) and the table.
func (v *TenantsView) View() string {
	body := Header(v.Title())
	switch {
	case v.loading:
		return body + "\nloading…\n" + Hint("first fetch can take a moment")
	case v.err != nil:
		return body + "\n" + ErrorBlock(v.err)
	case !v.loaded:
		return body + "\n" + Hint("(no data yet)")
	}
	return body + "\n" + v.table.View() + "\n" + Hint("↑/↓ row · q quit")
}
