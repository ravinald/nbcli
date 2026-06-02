// tenancy.go holds the bubbletea factories for the Netbox Tenancy module:
// Tenants and Contacts. Columns come from internal/columns via the
// ColumnsResolver passed by the shell; same registry powers the CLI.

package views

import (
	"context"

	"github.com/ravinald/nbcli/internal/netbox"
)

// NewTenants returns a View listing /tenancy/tenants/.
func NewTenants(client *netbox.Client, resolve ColumnsResolver) View {
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Tenant], error) {
		listOpts := netbox.ListTenantsOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListTenants(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Tenant]{}, err
		}
		return FetchResult[netbox.Tenant]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Tenant]("Tenants", "tenants", resolve, func(t netbox.Tenant) int { return t.ID }, fetcher)
}

// NewContacts returns a View listing /tenancy/contacts/.
func NewContacts(client *netbox.Client, resolve ColumnsResolver) View {
	fetcher := func(ctx context.Context, opts FetchOpts) (FetchResult[netbox.Contact], error) {
		listOpts := netbox.ListContactsOptions{Offset: opts.Offset, Limit: opts.Limit}
		applySearchOrID(&listOpts.Extra, opts)
		page, err := client.ListContacts(ctx, listOpts)
		if err != nil {
			return FetchResult[netbox.Contact]{}, err
		}
		return FetchResult[netbox.Contact]{Rows: page.Results, Total: page.Count}, nil
	}
	return newBaseView[netbox.Contact]("Contacts", "contacts", resolve, func(c netbox.Contact) int { return c.ID }, fetcher)
}
