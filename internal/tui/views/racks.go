package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ravinald/nbcli/internal/netbox"
)

// RacksView lists DCIM racks in a scrollable table.
type RacksView struct {
	client      *netbox.Client
	table       table.Model
	rows        []netbox.Rack
	selectedIdx int
	inDetail    bool
	loaded      bool
	loading     bool
	err         error
}

type racksLoadedMsg struct{ rows []netbox.Rack }

// NewRacks constructs the view bound to client. No fetch until Focus.
func NewRacks(client *netbox.Client) *RacksView {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 18},
		{Title: "Site", Width: 14},
		{Title: "Location", Width: 18},
		{Title: "Role", Width: 14},
		{Title: "Status", Width: 12},
		{Title: "U", Width: 4},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(defaultTableStyles())
	return &RacksView{client: client, table: t}
}

// Title is the human label rendered above the view body.
func (v *RacksView) Title() string { return "Racks" }

// Init is required by tea.Model; loading is driven by Focus.
func (v *RacksView) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called.
func (v *RacksView) Focus() tea.Cmd {
	if v.loaded || v.loading {
		return nil
	}
	v.loading = true
	return v.fetch()
}

func (v *RacksView) fetch() tea.Cmd {
	return func() tea.Msg {
		rows, err := netbox.ListAll(context.Background(),
			v.client.RacksFetcher(netbox.ListRacksOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return racksLoadedMsg{rows: rows}
	}
}

// Update routes async load messages and forwards everything else to the table.
func (v *RacksView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case racksLoadedMsg:
		v.loading = false
		v.loaded = true
		v.err = nil
		v.rows = m.rows
		rows := make([]table.Row, 0, len(m.rows))
		for _, r := range m.rows {
			rows = append(rows, table.Row{
				strconv.Itoa(r.ID),
				r.Name,
				nestedName(r.Site),
				nestedName(r.Location),
				nestedName(r.Role),
				r.Status.Label,
				strconv.Itoa(r.UHeight),
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
func (v *RacksView) View() string {
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
