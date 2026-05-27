package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ravinald/nbcli/internal/netbox"
)

// ContactsView lists contacts in a scrollable table.
type ContactsView struct {
	client  *netbox.Client
	table   table.Model
	loaded  bool
	loading bool
	err     error
}

type contactsLoadedMsg struct{ rows []netbox.Contact }

// NewContacts constructs the view bound to client. No fetch happens until Focus.
func NewContacts(client *netbox.Client) *ContactsView {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 24},
		{Title: "Title", Width: 18},
		{Title: "Email", Width: 28},
		{Title: "Phone", Width: 16},
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
	return &ContactsView{client: client, table: t}
}

// Title is the human label rendered above the view body.
func (v *ContactsView) Title() string { return "Contacts" }

// Init is required by tea.Model; loading is driven by Focus.
func (v *ContactsView) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called.
func (v *ContactsView) Focus() tea.Cmd {
	if v.loaded || v.loading {
		return nil
	}
	v.loading = true
	return v.fetch()
}

func (v *ContactsView) fetch() tea.Cmd {
	return func() tea.Msg {
		rows, err := netbox.ListAll(context.Background(),
			v.client.ContactsFetcher(netbox.ListContactsOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return contactsLoadedMsg{rows: rows}
	}
}

// Update routes async load messages and forwards everything else to the table.
func (v *ContactsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case contactsLoadedMsg:
		v.loading = false
		v.loaded = true
		v.err = nil
		rows := make([]table.Row, 0, len(m.rows))
		for _, c := range m.rows {
			rows = append(rows, table.Row{
				strconv.Itoa(c.ID),
				c.Name,
				c.Title,
				c.Email,
				c.Phone,
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

// View renders the title, status, and table.
func (v *ContactsView) View() string {
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
