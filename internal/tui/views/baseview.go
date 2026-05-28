package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RowMapper turns a typed result into a table row. Provided per-view by the
// factory function in the matching category file.
type RowMapper[T any] func(T) table.Row

// IDOf returns the Netbox ID of a typed record. Every Netbox resource has an
// `ID int` field; the factory provides a tiny closure (`func(s Site) int {
// return s.ID }`) so the generic base can look up records without reflection.
type IDOf[T any] func(T) int

// FetcherFn loads all rows for the view. The concrete factory supplies one
// that wraps netbox.ListAll with the resource's typed fetcher and a paging
// budget appropriate to the resource (e.g. interfaces is capped lower).
type FetcherFn[T any] func(ctx context.Context) ([]T, error)

// baseView is the generic skeleton every resource view uses. It owns:
//
//   - the bubbles/table widget and its row cache
//   - load lifecycle (loaded/loading/err)
//   - detail-mode state and key handling (Enter → detail, Esc → list)
//
// Concrete factories (NewTenants, NewDevices, ...) build a *baseView[T] with
// the right title/columns/mapper/fetcher and return it as a View. The
// concrete types are erased — the generic baseView is the only View
// implementation in the package.
type baseView[T any] struct {
	title   string
	table   table.Model
	mapper  RowMapper[T]
	idOf    IDOf[T]
	fetcher FetcherFn[T]

	// allRows is the full unfiltered cache from the last fetch; rows is the
	// currently-visible slice (== allRows when no filter is active, the
	// filtered subset when search is committed).
	allRows []T
	rows    []T

	selectedIdx int
	inDetail    bool
	detailFKs   []FKRef

	// pendingOpenID is set when OpenDetailByID was called before the view
	// finished loading. The load handler honors it.
	pendingOpenID int

	// search is the inline filter. searching is true while the input is
	// focused; the input's value is the live query. When searching is false
	// but the value is non-empty, the table shows the committed filter.
	searching   bool
	searchInput textinput.Model

	loaded  bool
	loading bool
	err     error
}

// loadedMsg is the per-T async-load result. Generic so different views can't
// accidentally consume each other's load messages.
type loadedMsg[T any] struct{ rows []T }

// newBaseView constructs a configured baseView. Columns, mapper, idOf, and
// fetcher are fixed at construction; the fetcher fires on first Focus.
func newBaseView[T any](title string, cols []table.Column, mapper RowMapper[T], idOf IDOf[T], fetcher FetcherFn[T]) *baseView[T] {
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(defaultTableStyles())

	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "search…"
	ti.CharLimit = 64
	ti.Width = 40

	return &baseView[T]{
		title:       title,
		table:       t,
		mapper:      mapper,
		idOf:        idOf,
		fetcher:     fetcher,
		searchInput: ti,
	}
}

// Title is the human label rendered above the view body.
func (b *baseView[T]) Title() string { return b.title }

// Init is required by tea.Model; loading is driven by Focus.
func (b *baseView[T]) Init() tea.Cmd { return nil }

// Focus fetches data the first time it's called. Re-focus is a no-op.
func (b *baseView[T]) Focus() tea.Cmd {
	if b.loaded || b.loading {
		return nil
	}
	b.loading = true
	return func() tea.Msg {
		rows, err := b.fetcher(context.Background())
		if err != nil {
			return ErrMsg{Err: err}
		}
		return loadedMsg[T]{rows: rows}
	}
}

// Update routes async load messages, detail-mode keys (Enter/Esc), search
// keys ('/' to open, typing while open, Enter to commit, Esc to cancel),
// and otherwise forwards to the table widget for row navigation.
func (b *baseView[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case loadedMsg[T]:
		b.loading = false
		b.loaded = true
		b.err = nil
		b.allRows = m.rows
		b.applyFilter()
		// Pick up a deferred drill-down request now that data is in.
		if b.pendingOpenID != 0 {
			wanted := b.pendingOpenID
			b.pendingOpenID = 0
			b.openDetailLocal(wanted)
		}
		return b, nil
	case ErrMsg:
		b.loading = false
		b.err = m.Err
		return b, nil
	case tea.KeyMsg:
		// Search-mode keys consume everything (textinput owns the keyboard).
		if b.searching {
			switch m.String() {
			case "esc":
				b.searching = false
				b.searchInput.Blur()
				b.searchInput.SetValue("")
				b.applyFilter()
				return b, nil
			case "enter":
				b.searching = false
				b.searchInput.Blur()
				return b, nil
			}
			var cmd tea.Cmd
			b.searchInput, cmd = b.searchInput.Update(msg)
			b.applyFilter()
			return b, cmd
		}
		// Detail-mode FK navigation: digit keys jump to the [N]-tagged FK.
		if b.inDetail {
			s := m.String()
			if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
				idx := int(s[0]-'0') - 1
				if idx < len(b.detailFKs) {
					fk := b.detailFKs[idx]
					return b, func() tea.Msg {
						return NavMsg{ViewName: fk.ViewName, ID: fk.ID}
					}
				}
			}
		}

		// Normal (list / detail) mode.
		switch m.String() {
		case "enter":
			if !b.inDetail && b.loaded && len(b.rows) > 0 {
				b.selectedIdx = b.table.Cursor()
				b.inDetail = true
				b.detailFKs = DetailFKs(b.rows[b.selectedIdx])
				return b, nil
			}
		case "esc":
			if b.inDetail {
				b.inDetail = false
				b.detailFKs = nil
				return b, nil
			}
			// Esc with a committed filter clears it.
			if b.searchInput.Value() != "" {
				b.searchInput.SetValue("")
				b.applyFilter()
				return b, nil
			}
			// Nothing internal to dismiss → bubble up so the shell can de-focus.
			return b, func() tea.Msg { return EscapeUpMsg{} }
		case "/":
			if !b.inDetail && b.loaded {
				b.searching = true
				return b, b.searchInput.Focus()
			}
		}
	}
	if b.inDetail {
		return b, nil
	}
	var cmd tea.Cmd
	b.table, cmd = b.table.Update(msg)
	return b, cmd
}

// applyFilter rebuilds b.rows and the table from b.allRows and the current
// search query. Empty query == no filter == all rows visible.
func (b *baseView[T]) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(b.searchInput.Value()))
	if q == "" {
		b.rows = b.allRows
	} else {
		filtered := make([]T, 0, len(b.allRows))
		for _, r := range b.allRows {
			row := b.mapper(r)
			matched := false
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), q) {
					matched = true
					break
				}
			}
			if matched {
				filtered = append(filtered, r)
			}
		}
		b.rows = filtered
	}
	tblRows := make([]table.Row, 0, len(b.rows))
	for _, r := range b.rows {
		tblRows = append(tblRows, b.mapper(r))
	}
	b.table.SetRows(tblRows)
}

// OpenDetailByID switches the view to detail mode for the record matching id.
// If the view hasn't loaded yet the request is deferred until loadedMsg
// arrives. Returns a tea.Cmd that kicks off the load on first call.
func (b *baseView[T]) OpenDetailByID(id int) tea.Cmd {
	if b.loaded {
		b.openDetailLocal(id)
		return nil
	}
	b.pendingOpenID = id
	return b.Focus()
}

// openDetailLocal looks up id in allRows and opens detail. No-op if the id
// isn't in the loaded set (the user just sees the list view of the target).
func (b *baseView[T]) openDetailLocal(id int) {
	if b.idOf == nil {
		return
	}
	// Clear any active filter so the target row is visible.
	if b.searchInput.Value() != "" {
		b.searchInput.SetValue("")
	}
	b.applyFilter()
	for i, r := range b.rows {
		if b.idOf(r) == id {
			b.selectedIdx = i
			b.inDetail = true
			b.detailFKs = DetailFKs(r)
			b.table.SetCursor(i)
			return
		}
	}
}

// View renders the title and either the table or the detail of the selected row.
func (b *baseView[T]) View() string {
	body := Header(b.title)
	if b.inDetail && b.selectedIdx < len(b.rows) {
		hint := "detail · esc back"
		if len(b.detailFKs) > 0 {
			hint = "detail · 1-9 follow link · esc back"
		}
		return body + " · " + Hint(hint) + "\n\n" + RenderDetail(b.rows[b.selectedIdx])
	}
	switch {
	case b.loading:
		return body + "\nloading…\n" + Hint("first fetch can take a moment")
	case b.err != nil:
		return body + "\n" + ErrorBlock(b.err)
	case !b.loaded:
		return body + "\n" + Hint("(no data yet)")
	}
	return body + "\n" + b.table.View() + "\n" + b.statusLine()
}

// statusLine renders the bottom strip — search input when active, filter
// summary when a query is committed, default keybind hint otherwise.
func (b *baseView[T]) statusLine() string {
	switch {
	case b.searching:
		return "/" + b.searchInput.View() + "  " + Hint("enter keep · esc clear")
	case b.searchInput.Value() != "":
		return Hint(fmt.Sprintf("filter %q · %d/%d rows · esc clear · / edit",
			b.searchInput.Value(), len(b.rows), len(b.allRows)))
	default:
		return Hint("↑/↓ row · enter detail · / search · q quit")
	}
}

// defaultTableStyles is the shared lipgloss table style used by every view.
// Centralized so theme tweaks land in one place.
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
