package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RowMapper turns a typed result into a table row. Provided per-view by the
// factory function in the matching category file.
type RowMapper[T any] func(T) table.Row

// IDOf returns the Netbox ID of a typed record. Every Netbox resource has an
// `ID int` field; the factory provides a tiny closure so the generic base
// can look up records without reflection.
type IDOf[T any] func(T) int

// FetcherFn loads one page (or one single record) according to opts. The
// factory closure wraps the resource's typed client method with the right
// query/filter plumbing.
type FetcherFn[T any] func(ctx context.Context, opts FetchOpts) (FetchResult[T], error)

// baseView is the generic skeleton every resource view uses. It owns:
//
//   - the bubbles/table widget
//   - pagination state (offset/limit/total)
//   - committed search query (sent to the API, not client-filtered)
//   - detail-mode state and key handling
//   - FK drill-down (OpenDetailByID + ID-passthrough on the fetcher)
//
// Concrete factories (NewTenants, NewDevices, ...) build a *baseView[T] with
// the right title/columns/mapper/idOf/fetcher and return it as a View.
type baseView[T any] struct {
	title   string
	table   table.Model
	mapper  RowMapper[T]
	idOf    IDOf[T]
	fetcher FetcherFn[T]

	// Pagination + search state.
	rows   []T    // current page's rows
	offset int    // current page's start index
	limit  int    // page size; viewport-derived after first SizeMsg
	total  int    // total matching records reported by the API
	query  string // committed search query ("" → no search)

	selectedIdx int
	inDetail    bool
	detailFKs   []FKRef
	// detailCursor is the highlighted FK in detail mode. -1 means no FKs
	// (or the view has no detail open). Up/Down/k/j move it; Enter follows.
	detailCursor int

	// pendingOpenID is consumed by the next fetchCmd. Set by OpenDetailByID
	// when the target ID isn't in the current page; the fetch uses ?id=<n>
	// to retrieve just that record and opens detail on load.
	pendingOpenID int

	// searching is true while the textinput owns the keyboard.
	searching   bool
	searchInput textinput.Model

	loaded  bool
	loading bool
	err     error
}

// loadedMsg carries the typed FetchResult plus a marker indicating whether
// the fetch was an ID-by-ID drill-down (so the receiver opens detail).
type loadedMsg[T any] struct {
	result     FetchResult[T]
	wasIDFetch bool
}

// defaultPageSize is the limit used until the first SizeMsg arrives.
const defaultPageSize = 50

// newBaseView constructs a configured baseView.
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

// Init is required by tea.Model; loading is driven by Focus / SizeMsg.
func (b *baseView[T]) Init() tea.Cmd { return nil }

// Focus fetches the first page on first call. Re-focus on an already-loaded
// view is a no-op unless a pending ID drill-down is queued.
func (b *baseView[T]) Focus() tea.Cmd {
	if b.pendingOpenID > 0 {
		return b.fetchCmd()
	}
	if b.loaded || b.loading {
		return nil
	}
	return b.fetchCmd()
}

// fetchCmd snapshots the current pagination/search/ID state and returns a
// Cmd that asks the closure for that page. Always reset pendingOpenID at
// snapshot time so a follow-up fetch (page change, search) doesn't drag it.
func (b *baseView[T]) fetchCmd() tea.Cmd {
	b.loading = true
	opts := FetchOpts{
		Offset: b.offset,
		Limit:  b.limit,
		Query:  b.query,
	}
	if opts.Limit <= 0 {
		opts.Limit = defaultPageSize
	}
	wasIDFetch := false
	if b.pendingOpenID > 0 {
		opts.ID = b.pendingOpenID
		opts.Offset = 0
		opts.Limit = 1
		opts.Query = ""
		b.pendingOpenID = 0
		wasIDFetch = true
	}
	fn := b.fetcher
	return func() tea.Msg {
		result, err := fn(context.Background(), opts)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return loadedMsg[T]{result: result, wasIDFetch: wasIDFetch}
	}
}

// Update routes async load messages, pagination keys, search keys, detail
// keys, and otherwise forwards to the table widget.
func (b *baseView[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case SizeMsg:
		// View chrome inside the right viewport's content area:
		//   title (1) + title margin blank (1) + status hint (1) = 3 lines
		// The table widget itself draws its column header + separator
		// inside its allotted height, so visible data rows = tableArea - 2.
		tableArea := m.Height - 3
		if tableArea < 5 {
			tableArea = 5
		}
		b.table.SetHeight(tableArea)
		if m.Width > 4 {
			b.table.SetWidth(m.Width)
		}
		// Page size = visible data rows. Set once on first sizing; later
		// resizes don't change page size mid-browse to avoid a disruptive
		// refetch.
		if b.limit == 0 {
			b.limit = tableArea - 2
		}
		return b, nil
	case loadedMsg[T]:
		b.loading = false
		b.loaded = true
		b.err = nil
		b.rows = m.result.Rows
		b.total = m.result.Total
		b.refreshTable()
		if m.wasIDFetch && len(b.rows) > 0 {
			b.selectedIdx = 0
			b.inDetail = true
			b.detailFKs = DetailFKs(b.rows[0])
			b.detailCursor = -1
			if len(b.detailFKs) > 0 {
				b.detailCursor = 0
			}
			b.table.SetCursor(0)
		}
		return b, nil
	case ErrMsg:
		b.loading = false
		b.err = m.Err
		return b, nil
	case tea.KeyMsg:
		// Search-mode keys consume everything; the textinput owns the keyboard.
		if b.searching {
			switch m.String() {
			case "esc":
				b.searching = false
				b.searchInput.Blur()
				// Esc in search input always cancels what the user was
				// about to commit. If a query was already committed, leave
				// it (a separate Esc from the list clears it).
				return b, nil
			case "enter":
				b.searching = false
				b.searchInput.Blur()
				newQuery := b.searchInput.Value()
				if newQuery == b.query {
					return b, nil
				}
				b.query = newQuery
				b.offset = 0
				return b, b.fetchCmd()
			}
			var cmd tea.Cmd
			b.searchInput, cmd = b.searchInput.Update(msg)
			return b, cmd
		}

		// Detail-mode keymap is self-contained — different from list mode.
		if b.inDetail {
			s := m.String()
			// Direct digit jumps: works even for FKs beyond the cursor.
			if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
				idx := int(s[0]-'0') - 1
				if idx < len(b.detailFKs) {
					fk := b.detailFKs[idx]
					return b, func() tea.Msg {
						return NavMsg{ViewName: fk.ViewName, ID: fk.ID}
					}
				}
				return b, nil
			}
			switch s {
			case "up", "k":
				if b.detailCursor > 0 {
					b.detailCursor--
				}
				return b, nil
			case "down", "j":
				if b.detailCursor >= 0 && b.detailCursor < len(b.detailFKs)-1 {
					b.detailCursor++
				}
				return b, nil
			case "enter":
				if b.detailCursor >= 0 && b.detailCursor < len(b.detailFKs) {
					fk := b.detailFKs[b.detailCursor]
					return b, func() tea.Msg {
						return NavMsg{ViewName: fk.ViewName, ID: fk.ID}
					}
				}
				return b, nil
			case "esc":
				b.inDetail = false
				b.detailFKs = nil
				b.detailCursor = -1
				return b, nil
			}
			// Swallow other keys in detail; don't forward to table.
			return b, nil
		}

		switch m.String() {
		case "enter":
			if b.loaded && len(b.rows) > 0 {
				b.selectedIdx = b.table.Cursor()
				b.inDetail = true
				b.detailFKs = DetailFKs(b.rows[b.selectedIdx])
				b.detailCursor = -1
				if len(b.detailFKs) > 0 {
					b.detailCursor = 0
				}
				return b, nil
			}
		case "esc":
			// Esc with a committed query clears it and reloads.
			if b.query != "" {
				b.query = ""
				b.offset = 0
				b.searchInput.SetValue("")
				return b, b.fetchCmd()
			}
			// Nothing internal to dismiss → bubble up so the shell can de-focus.
			return b, func() tea.Msg { return EscapeUpMsg{} }
		case "/":
			if !b.inDetail && b.loaded {
				b.searching = true
				b.searchInput.SetValue(b.query)
				b.searchInput.CursorEnd()
				return b, b.searchInput.Focus()
			}
		case "pgdown", "ctrl+f":
			if !b.inDetail && b.loaded && b.offset+b.limit < b.total {
				b.offset += b.limit
				return b, b.fetchCmd()
			}
		case "pgup", "ctrl+b":
			if !b.inDetail && b.loaded && b.offset > 0 {
				b.offset -= b.limit
				if b.offset < 0 {
					b.offset = 0
				}
				return b, b.fetchCmd()
			}
		case "home":
			if !b.inDetail && b.loaded && b.offset > 0 {
				b.offset = 0
				return b, b.fetchCmd()
			}
		case "end":
			if !b.inDetail && b.loaded && b.total > 0 && b.limit > 0 {
				last := ((b.total - 1) / b.limit) * b.limit
				if last != b.offset {
					b.offset = last
					return b, b.fetchCmd()
				}
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

// refreshTable rebuilds the bubbles/table's rows from b.rows via the mapper.
func (b *baseView[T]) refreshTable() {
	tblRows := make([]table.Row, 0, len(b.rows))
	for _, r := range b.rows {
		tblRows = append(tblRows, b.mapper(r))
	}
	b.table.SetRows(tblRows)
}

// OpenDetailByID asks the view to switch to detail mode for the record with
// the given ID. If the ID is in the current page, opens detail directly;
// otherwise queues a single-record fetch by ID. The fetcher must support the
// FetchOpts.ID field.
func (b *baseView[T]) OpenDetailByID(id int) tea.Cmd {
	if b.idOf != nil {
		for i, r := range b.rows {
			if b.idOf(r) == id {
				b.selectedIdx = i
				b.inDetail = true
				b.detailFKs = DetailFKs(r)
				b.detailCursor = -1
				if len(b.detailFKs) > 0 {
					b.detailCursor = 0
				}
				b.table.SetCursor(i)
				return nil
			}
		}
	}
	// Not in current page → ask the fetcher to GET this one record by id.
	b.pendingOpenID = id
	return b.fetchCmd()
}

// View renders the title and either the table or the detail of the selected row.
func (b *baseView[T]) View() string {
	body := Header(b.title)
	if b.inDetail && b.selectedIdx < len(b.rows) {
		detail := RenderDetailCursor(b.rows[b.selectedIdx], b.detailCursor)
		topHint := Hint("detail · esc back")
		bottomHelp := ""
		if len(b.detailFKs) > 0 {
			bottomHelp = "\n\n" + Hint("↑/↓ select link · enter follow · 1-9 jump · esc back")
		}
		return body + " · " + topHint + "\n\n" + detail + bottomHelp
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

// statusLine builds the bottom strip — search input when active, page info
// + committed search summary otherwise.
func (b *baseView[T]) statusLine() string {
	if b.searching {
		return "/" + b.searchInput.View() + "  " + Hint("enter search · esc cancel")
	}
	pages := ""
	if b.limit > 0 && b.total > 0 {
		cur := b.offset/b.limit + 1
		tot := (b.total + b.limit - 1) / b.limit
		end := b.offset + len(b.rows)
		if end > b.total {
			end = b.total
		}
		pages = fmt.Sprintf(" · page %d/%d · rows %d-%d of %d",
			cur, tot, b.offset+1, end, b.total)
	}
	if b.query != "" {
		return Hint(fmt.Sprintf("search %q%s · esc clear · / edit", b.query, pages))
	}
	return Hint(fmt.Sprintf("pgup/pgdn page%s · / search · enter detail · q quit", pages))
}

// defaultTableStyles is the shared lipgloss table style used by every view.
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
