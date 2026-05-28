package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ravinald/nbcli/internal/netbox"
)

// IPAddressesView lists IPAM IP addresses.
type IPAddressesView struct {
	client      *netbox.Client
	table       table.Model
	rows        []netbox.IPAddress
	selectedIdx int
	inDetail    bool
	loaded      bool
	loading     bool
	err         error
}

type ipAddressesLoadedMsg struct{ rows []netbox.IPAddress }

// NewIPAddresses constructs the view bound to client. No fetch until Focus.
func NewIPAddresses(client *netbox.Client) *IPAddressesView {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Address", Width: 22},
		{Title: "Family", Width: 7},
		{Title: "VRF", Width: 14},
		{Title: "Status", Width: 12},
		{Title: "DNS", Width: 24},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(defaultTableStyles())
	return &IPAddressesView{client: client, table: t}
}

// Title is the human label rendered above the view body.
func (v *IPAddressesView) Title() string { return "IP Addresses" }

// Init is required by tea.Model; loading is driven by Focus.
func (v *IPAddressesView) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called.
func (v *IPAddressesView) Focus() tea.Cmd {
	if v.loaded || v.loading {
		return nil
	}
	v.loading = true
	return v.fetch()
}

func (v *IPAddressesView) fetch() tea.Cmd {
	return func() tea.Msg {
		rows, err := netbox.ListAll(context.Background(),
			v.client.IPAddressesFetcher(netbox.ListIPAddressesOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return ipAddressesLoadedMsg{rows: rows}
	}
}

// Update routes async load messages and forwards everything else to the table.
func (v *IPAddressesView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case ipAddressesLoadedMsg:
		v.loading = false
		v.loaded = true
		v.err = nil
		v.rows = m.rows
		rows := make([]table.Row, 0, len(m.rows))
		for _, ip := range m.rows {
			rows = append(rows, table.Row{
				strconv.Itoa(ip.ID),
				ip.Address,
				ip.Family.Label,
				nestedName(ip.VRF),
				ip.Status.Label,
				ip.DNSName,
			})
		}
		v.table.SetRows(rows)
	case ErrMsg:
		v.loading = false
		v.err = m.Err
	case tea.KeyMsg:
		switch m.String() {
		case "enter":
			if !v.inDetail && v.loaded && len(v.rows) > 0 {
				v.selectedIdx = v.table.Cursor()
				v.inDetail = true
				return v, nil
			}
		case "esc":
			if v.inDetail {
				v.inDetail = false
				return v, nil
			}
		}
	}
	if v.inDetail {
		return v, nil
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

// View renders the title, then status/table — or detail of the selected row.
func (v *IPAddressesView) View() string {
	body := Header(v.Title())
	if v.inDetail && v.selectedIdx < len(v.rows) {
		return body + " · " + Hint("detail · esc back") + "\n\n" + RenderDetail(v.rows[v.selectedIdx])
	}
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
