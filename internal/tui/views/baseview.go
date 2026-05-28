package views

import (
	"context"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RowMapper turns a typed result into a table row. Provided per-view by the
// factory function in the matching category file.
type RowMapper[T any] func(T) table.Row

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
	fetcher FetcherFn[T]

	rows        []T
	selectedIdx int
	inDetail    bool
	loaded      bool
	loading     bool
	err         error
}

// loadedMsg is the per-T async-load result. Generic so different views can't
// accidentally consume each other's load messages.
type loadedMsg[T any] struct{ rows []T }

// newBaseView constructs a configured baseView. Columns and mapper are fixed
// at construction; the fetcher fires on first Focus.
func newBaseView[T any](title string, cols []table.Column, mapper RowMapper[T], fetcher FetcherFn[T]) *baseView[T] {
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(defaultTableStyles())
	return &baseView[T]{
		title:   title,
		table:   t,
		mapper:  mapper,
		fetcher: fetcher,
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

// Update routes async load messages, detail-mode keys (Enter/Esc), and
// otherwise forwards to the table widget for row navigation.
func (b *baseView[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case loadedMsg[T]:
		b.loading = false
		b.loaded = true
		b.err = nil
		b.rows = m.rows
		rows := make([]table.Row, 0, len(m.rows))
		for _, r := range m.rows {
			rows = append(rows, b.mapper(r))
		}
		b.table.SetRows(rows)
	case ErrMsg:
		b.loading = false
		b.err = m.Err
	case tea.KeyMsg:
		switch m.String() {
		case "enter":
			if !b.inDetail && b.loaded && len(b.rows) > 0 {
				b.selectedIdx = b.table.Cursor()
				b.inDetail = true
				return b, nil
			}
		case "esc":
			if b.inDetail {
				b.inDetail = false
				return b, nil
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

// View renders the title and either the table or the detail of the selected row.
func (b *baseView[T]) View() string {
	body := Header(b.title)
	if b.inDetail && b.selectedIdx < len(b.rows) {
		return body + " · " + Hint("detail · esc back") + "\n\n" + RenderDetail(b.rows[b.selectedIdx])
	}
	switch {
	case b.loading:
		return body + "\nloading…\n" + Hint("first fetch can take a moment")
	case b.err != nil:
		return body + "\n" + ErrorBlock(b.err)
	case !b.loaded:
		return body + "\n" + Hint("(no data yet)")
	}
	return body + "\n" + b.table.View() + "\n" + Hint("↑/↓ row · enter detail · q quit")
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
