// tenancy.go holds the bubbletea factories for the Netbox Tenancy module:
// Tenants and Contacts. Columns come from internal/columns via the
// ColumnsResolver passed by the shell; same registry powers the CLI.

package views

import (
	"context"
	"net/url"
	"strconv"

	"github.com/charmbracelet/bubbles/table"

	"github.com/ravinald/nbcli/internal/columns"
	"github.com/ravinald/nbcli/internal/netbox"
)

// NewTenants returns a View listing /tenancy/tenants/.
func NewTenants(client *netbox.Client, resolve ColumnsResolver) View {
	visible := resolve("tenants")
	cols, mapper := buildCols[netbox.Tenant](visible)
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Tenant], error) {
		listOpts := netbox.ListTenantsOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListTenants(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Tenant]{}, err
		}
		return FetchResult[netbox.Tenant]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Tenant]("Tenants", cols, mapper, func(t netbox.Tenant) int { return t.ID }, fetcher)
}

// NewContacts returns a View listing /tenancy/contacts/.
func NewContacts(client *netbox.Client, resolve ColumnsResolver) View {
	visible := resolve("contacts")
	cols, mapper := buildCols[netbox.Contact](visible)
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Contact], error) {
		listOpts := netbox.ListContactsOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListContacts(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Contact]{}, err
		}
		return FetchResult[netbox.Contact]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Contact]("Contacts", cols, mapper, func(c netbox.Contact) int { return c.ID }, fetcher)
}

// buildCols adapts a slice of columns.Column into the bubbles/table.Column
// definitions and a typed RowMapper. Used by every factory in this package
// so column wiring stays in one place.
func buildCols[T any](visible []columns.Column) ([]table.Column, RowMapper[T]) {
	tcols := make([]table.Column, len(visible))
	for i, c := range visible {
		tcols[i] = table.Column{Title: c.Header, Width: c.Width}
	}
	mapper := func(row T) table.Row {
		cells := make(table.Row, len(visible))
		for i, c := range visible {
			cells[i] = c.Extract(row)
		}
		return cells
	}
	return tcols, mapper
}

// applySearchOrID pokes the FetchOpts.Query / FetchOpts.ID into the typed
// list options' Extra url.Values. ID takes precedence (FK drill-down).
func applySearchOrID(extra *url.Values, opts FetchOpts) {
	if opts.ID > 0 {
		if *extra == nil {
			*extra = url.Values{}
		}
		extra.Set("id", strconv.Itoa(opts.ID))
		return
	}
	if opts.Query != "" {
		if *extra == nil {
			*extra = url.Values{}
		}
		extra.Set("q", opts.Query)
	}
}
