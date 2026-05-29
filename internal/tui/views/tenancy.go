// tenancy.go holds the bubbletea factories for the Netbox Tenancy module:
// Tenants and Contacts. Each NewXxx returns a configured *baseView[T] as a
// View; the View interface is the only thing app.go cares about.

package views

import (
	"context"
	"net/url"
	"strconv"

	"github.com/charmbracelet/bubbles/table"

	"github.com/ravinald/nbcli/internal/netbox"
)

// NewTenants returns a View listing /tenancy/tenants/.
func NewTenants(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 26},
		{Title: "Slug", Width: 26},
		{Title: "Group", Width: 22},
	}
	mapper := func(t netbox.Tenant) table.Row {
		return table.Row{
			strconv.Itoa(t.ID),
			t.Name,
			t.Slug,
			nestedName(t.Group),
		}
	}
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Tenant], error) {
		listOpts := netbox.ListTenantsOptions{
			Offset: opts.Offset,
			Limit:  opts.Limit,
		}
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
func NewContacts(client *netbox.Client) View {
	cols := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 24},
		{Title: "Title", Width: 18},
		{Title: "Email", Width: 28},
		{Title: "Phone", Width: 16},
	}
	mapper := func(c netbox.Contact) table.Row {
		return table.Row{
			strconv.Itoa(c.ID),
			c.Name,
			c.Title,
			c.Email,
			c.Phone,
		}
	}
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Contact], error) {
		listOpts := netbox.ListContactsOptions{
			Offset: opts.Offset,
			Limit:  opts.Limit,
		}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListContacts(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Contact]{}, err
		}
		return FetchResult[netbox.Contact]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Contact]("Contacts", cols, mapper, func(c netbox.Contact) int { return c.ID }, fetcher)
}

// applySearchOrID pokes the FetchOpts.Query / FetchOpts.ID into the typed
// list options' Extra url.Values. ID takes precedence (FK drill-down).
// Resource-specific list filters (Name, Status, etc.) intentionally stay
// out of the TUI's reach — the API search via `q=` is the broad-strokes path.
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
