// tenancy.go holds the bubbletea factories for the Netbox Tenancy module:
// Tenants and Contacts. Each NewXxx returns a configured *baseView[T] as a
// View; the View interface is the only thing app.go cares about.

package views

import (
	"context"
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
	fetcher := func(ctx context.Context) ([]netbox.Tenant, error) {
		return netbox.ListAll(ctx,
			client.TenantsFetcher(netbox.ListTenantsOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
	}
	return newBaseView[netbox.Tenant]("Tenants", cols, mapper, fetcher)
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
	fetcher := func(ctx context.Context) ([]netbox.Contact, error) {
		return netbox.ListAll(ctx,
			client.ContactsFetcher(netbox.ListContactsOptions{}),
			netbox.IterateOptions{PageSize: 100, MaxPages: 50})
	}
	return newBaseView[netbox.Contact]("Contacts", cols, mapper, fetcher)
}
