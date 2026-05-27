package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ravinald/nbcli/internal/netbox"
)

// ClustersView lists virtualization clusters.
type ClustersView struct {
	client  *netbox.Client
	table   table.Model
	loaded  bool
	loading bool
	err     error
}

type clustersLoadedMsg struct{ rows []netbox.Cluster }

// NewClusters constructs the view bound to client. No fetch until Focus.
func NewClusters(client *netbox.Client) *ClustersView {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 22},
		{Title: "Type", Width: 18},
		{Title: "Group", Width: 16},
		{Title: "Site", Width: 14},
		{Title: "Status", Width: 12},
		{Title: "VMs", Width: 6},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(defaultTableStyles())
	return &ClustersView{client: client, table: t}
}

// Title is the human label rendered above the view body.
func (v *ClustersView) Title() string { return "Clusters" }

// Init is required by tea.Model; loading is driven by Focus.
func (v *ClustersView) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called.
func (v *ClustersView) Focus() tea.Cmd {
	if v.loaded || v.loading {
		return nil
	}
	v.loading = true
	return v.fetch()
}

func (v *ClustersView) fetch() tea.Cmd {
	return func() tea.Msg {
		rows, err := netbox.ListAll(context.Background(),
			v.client.ClustersFetcher(netbox.ListClustersOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return clustersLoadedMsg{rows: rows}
	}
}

// Update routes async load messages and forwards everything else to the table.
func (v *ClustersView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case clustersLoadedMsg:
		v.loading = false
		v.loaded = true
		v.err = nil
		rows := make([]table.Row, 0, len(m.rows))
		for _, c := range m.rows {
			rows = append(rows, table.Row{
				strconv.Itoa(c.ID),
				c.Name,
				nestedName(c.Type),
				nestedName(c.Group),
				nestedName(c.Site),
				c.Status.Label,
				strconv.Itoa(c.VirtualMachineCount),
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
func (v *ClustersView) View() string {
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
