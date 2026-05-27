package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ravinald/nbcli/internal/netbox"
)

// InterfacesView lists DCIM interfaces. Without a device filter the result
// set can be very large; the view caps at 20 pages × 100 = 2000 rows as a
// safety belt. For targeted queries, use `nbcli show interfaces device ...`.
type InterfacesView struct {
	client  *netbox.Client
	table   table.Model
	loaded  bool
	loading bool
	err     error
}

type interfacesLoadedMsg struct{ rows []netbox.Interface }

// NewInterfaces constructs the view bound to client. No fetch until Focus.
func NewInterfaces(client *netbox.Client) *InterfacesView {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Device", Width: 20},
		{Title: "Name", Width: 18},
		{Title: "Type", Width: 18},
		{Title: "Enabled", Width: 8},
		{Title: "MAC", Width: 18},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(defaultTableStyles())
	return &InterfacesView{client: client, table: t}
}

// Title is the human label rendered above the view body.
func (v *InterfacesView) Title() string { return "Interfaces" }

// Init is required by tea.Model; loading is driven by Focus.
func (v *InterfacesView) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called.
func (v *InterfacesView) Focus() tea.Cmd {
	if v.loaded || v.loading {
		return nil
	}
	v.loading = true
	return v.fetch()
}

func (v *InterfacesView) fetch() tea.Cmd {
	return func() tea.Msg {
		rows, err := netbox.ListAll(context.Background(),
			v.client.InterfacesFetcher(netbox.ListInterfacesOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 20})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return interfacesLoadedMsg{rows: rows}
	}
}

// Update routes async load messages and forwards everything else to the table.
func (v *InterfacesView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case interfacesLoadedMsg:
		v.loading = false
		v.loaded = true
		v.err = nil
		rows := make([]table.Row, 0, len(m.rows))
		for _, i := range m.rows {
			rows = append(rows, table.Row{
				strconv.Itoa(i.ID),
				nestedName(i.Device),
				i.Name,
				i.Type.Label,
				strconv.FormatBool(i.Enabled),
				i.MACAddress,
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
func (v *InterfacesView) View() string {
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
