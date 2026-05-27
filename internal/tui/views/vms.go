package views

import (
	"context"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ravinald/nbcli/internal/netbox"
)

// VMsView lists virtual machines.
type VMsView struct {
	client  *netbox.Client
	table   table.Model
	loaded  bool
	loading bool
	err     error
}

type vmsLoadedMsg struct{ rows []netbox.VirtualMachine }

// NewVMs constructs the view bound to client. No fetch until Focus.
func NewVMs(client *netbox.Client) *VMsView {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 22},
		{Title: "Status", Width: 12},
		{Title: "Cluster", Width: 16},
		{Title: "Site", Width: 14},
		{Title: "vCPUs", Width: 7},
		{Title: "MemMB", Width: 8},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(defaultTableStyles())
	return &VMsView{client: client, table: t}
}

// Title is the human label rendered above the view body.
func (v *VMsView) Title() string { return "Virtual Machines" }

// Init is required by tea.Model; loading is driven by Focus.
func (v *VMsView) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called.
func (v *VMsView) Focus() tea.Cmd {
	if v.loaded || v.loading {
		return nil
	}
	v.loading = true
	return v.fetch()
}

func (v *VMsView) fetch() tea.Cmd {
	return func() tea.Msg {
		rows, err := netbox.ListAll(context.Background(),
			v.client.VMsFetcher(netbox.ListVMsOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
		if err != nil {
			return ErrMsg{Err: err}
		}
		return vmsLoadedMsg{rows: rows}
	}
}

// Update routes async load messages and forwards everything else to the table.
func (v *VMsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case vmsLoadedMsg:
		v.loading = false
		v.loaded = true
		v.err = nil
		rows := make([]table.Row, 0, len(m.rows))
		for _, vm := range m.rows {
			vcpu := ""
			if vm.VCPUs != nil {
				vcpu = strconv.FormatFloat(*vm.VCPUs, 'f', -1, 64)
			}
			mem := ""
			if vm.Memory != nil {
				mem = strconv.Itoa(*vm.Memory)
			}
			rows = append(rows, table.Row{
				strconv.Itoa(vm.ID),
				vm.Name,
				vm.Status.Label,
				nestedName(vm.Cluster),
				nestedName(vm.Site),
				vcpu,
				mem,
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
func (v *VMsView) View() string {
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
