package netbox

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Tenant is the /tenancy/tenants/ resource (subset).
type Tenant struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	Display     string     `json:"display"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Group       *NestedRef `json:"group,omitempty"`
	Description string     `json:"description,omitempty"`
	Comments    string     `json:"comments,omitempty"`
	Created     string     `json:"created,omitempty"`
	LastUpdated string     `json:"last_updated,omitempty"`
}

// Contact is the /tenancy/contacts/ resource (subset).
type Contact struct {
	ID          int        `json:"id"`
	URL         string     `json:"url"`
	Display     string     `json:"display"`
	Name        string     `json:"name"`
	Title       string     `json:"title,omitempty"`
	Phone       string     `json:"phone,omitempty"`
	Email       string     `json:"email,omitempty"`
	Address     string     `json:"address,omitempty"`
	Group       *NestedRef `json:"group,omitempty"`
	Description string     `json:"description,omitempty"`
	Created     string     `json:"created,omitempty"`
	LastUpdated string     `json:"last_updated,omitempty"`
}

// ListTenantsOptions filters /tenancy/tenants/.
type ListTenantsOptions struct {
	Name   string
	Slug   string
	Group  string
	Limit  int
	Offset int
	Extra  url.Values
}

// ListTenants returns one page of tenants.
func (c *Client) ListTenants(ctx context.Context, opts ListTenantsOptions) (Page[Tenant], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	if opts.Name != "" {
		q.Set("name", opts.Name)
	}
	if opts.Slug != "" {
		q.Set("slug", opts.Slug)
	}
	if opts.Group != "" {
		q.Set("group", opts.Group)
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[Tenant]
	if err := c.Do(ctx, "GET", "/api/tenancy/tenants/", q, nil, &page); err != nil {
		return Page[Tenant]{}, fmt.Errorf("tenancy: list tenants: %w", err)
	}
	return page, nil
}

// TenantsFetcher returns a PageFetcher bound to opts, suitable for ListAll /
// Iterate. Offset and Limit on opts are overridden per call.
func (c *Client) TenantsFetcher(opts ListTenantsOptions) PageFetcher[Tenant] {
	return func(ctx context.Context, offset, limit int) (Page[Tenant], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListTenants(ctx, opts)
	}
}

// ListContactsOptions filters /tenancy/contacts/.
type ListContactsOptions struct {
	Name   string
	Email  string
	Phone  string
	Group  string
	Limit  int
	Offset int
	Extra  url.Values
}

// ListContacts returns one page of contacts.
func (c *Client) ListContacts(ctx context.Context, opts ListContactsOptions) (Page[Contact], error) {
	q := url.Values{}
	for k, v := range opts.Extra {
		q[k] = v
	}
	if opts.Name != "" {
		q.Set("name", opts.Name)
	}
	if opts.Email != "" {
		q.Set("email", opts.Email)
	}
	if opts.Phone != "" {
		q.Set("phone", opts.Phone)
	}
	if opts.Group != "" {
		q.Set("group", opts.Group)
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	var page Page[Contact]
	if err := c.Do(ctx, "GET", "/api/tenancy/contacts/", q, nil, &page); err != nil {
		return Page[Contact]{}, fmt.Errorf("tenancy: list contacts: %w", err)
	}
	return page, nil
}

// ContactsFetcher returns a PageFetcher bound to opts.
func (c *Client) ContactsFetcher(opts ListContactsOptions) PageFetcher[Contact] {
	return func(ctx context.Context, offset, limit int) (Page[Contact], error) {
		opts.Offset = offset
		opts.Limit = limit
		return c.ListContacts(ctx, opts)
	}
}
