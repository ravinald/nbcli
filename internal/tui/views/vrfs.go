package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ravinald/nbcli/internal/netbox"
)

// VRFsView lists IPAM VRFs.
type VRFsView struct {
	client      *netbox.Client
	table       table.Model
	rows        []netbox.VRF
	selectedIdx int
	inDetail    bool
	loaded      bool
	loading     bool
	err         error
}

type vrfsLoadedMsg struct{ rows []netbox.VRF }

// NewVRFs constructs the view bound to client. No fetch until Focus.
func NewVRFs(client *netbox.Client) *VRFsView {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 20},
		{Title: "RD", Width: 18},
		{Title: "Tenant", Width: 16},
		{Title: "Description", Width: 30},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(defaultTableStyles())
	return &VRFsView{client: client, table: t}
}

// Title is the human label rendered above the view body.
func (v *VRFsView) Title() string { return "VRFs" }

// Init is required by tea.Model; loading is driven by Focus.
func (v *VRFsView) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called.
func (v *VRFsView) Focus() tea.Cmd {
	if v.loaded || v.loading {
		return nil
	}
	v.loading = true
	return v.fetch()
}

func (v *VRFsView) fetch() tea.Cmd {
	return func() tea.Msg {
		rows, err := netbox.ListAll(context.Background(),
			v.client.VRFsFetcher(netbox.ListVRFsOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return vrfsLoadedMsg{rows: rows}
	}
}

// Update routes async load messages and forwards everything else to the table.
func (v *VRFsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case vrfsLoadedMsg:
		v.loading = false
		v.loaded = true
		v.err = nil
		v.rows = m.rows
		rows := make([]table.Row, 0, len(m.rows))
		for _, r := range m.rows {
			rows = append(rows, table.Row{
				strconv.Itoa(r.ID),
				r.Name,
				r.RD,
				nestedName(r.Tenant),
				r.Description,
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
func (v *VRFsView) View() string {
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
