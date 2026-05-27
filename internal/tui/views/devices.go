package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ravinald/nbcli/internal/netbox"
)

// DevicesView lists DCIM devices in a scrollable table.
type DevicesView struct {
	client  *netbox.Client
	table   table.Model
	loaded  bool
	loading bool
	err     error
}

type devicesLoadedMsg struct{ rows []netbox.Device }

// NewDevices constructs the view bound to client. No fetch until Focus.
func NewDevices(client *netbox.Client) *DevicesView {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 22},
		{Title: "Type", Width: 22},
		{Title: "Site", Width: 14},
		{Title: "Rack", Width: 12},
		{Title: "Status", Width: 12},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(defaultTableStyles())
	return &DevicesView{client: client, table: t}
}

// Title is the human label rendered above the view body.
func (v *DevicesView) Title() string { return "Devices" }

// Init is required by tea.Model; loading is driven by Focus.
func (v *DevicesView) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called.
func (v *DevicesView) Focus() tea.Cmd {
	if v.loaded || v.loading {
		return nil
	}
	v.loading = true
	return v.fetch()
}

func (v *DevicesView) fetch() tea.Cmd {
	return func() tea.Msg {
		rows, err := netbox.ListAll(context.Background(),
			v.client.DevicesFetcher(netbox.ListDevicesOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return devicesLoadedMsg{rows: rows}
	}
}

// Update routes async load messages and forwards everything else to the table.
func (v *DevicesView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case devicesLoadedMsg:
		v.loading = false
		v.loaded = true
		v.err = nil
		rows := make([]table.Row, 0, len(m.rows))
		for _, d := range m.rows {
			devType := ""
			if d.DeviceType != nil {
				if d.DeviceType.Manufacturer != nil {
					devType = d.DeviceType.Manufacturer.Name + " "
				}
				devType += d.DeviceType.Model
			}
			rows = append(rows, table.Row{
				strconv.Itoa(d.ID),
				d.Name,
				devType,
				nestedName(d.Site),
				nestedName(d.Rack),
				d.Status.Label,
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
func (v *DevicesView) View() string {
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

// nestedName returns the Name of a NestedRef, "" if nil. Shared across DCIM
// views; pulled out so we don't repeat the nil-guard inline.
func nestedName(n *netbox.NestedRef) string {
	if n == nil {
		return ""
	}
	return n.Name
}

// defaultTableStyles is the shared lipgloss table style used by every DCIM
// view. Pulled out so adding a new view doesn't redefine the look.
func defaultTableStyles() table.Styles {
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
	return s
}
