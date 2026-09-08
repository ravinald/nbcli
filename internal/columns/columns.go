// Package columns is the single source of truth for what cells are renderable
// for each Netbox resource. The CLI's tabular output and the TUI's bubbles
// table both consume Sets from this package — picking which columns to show
// and in what order via the Config.Columns map.
//
// Conceptual model (mirrors the Netbox web UI):
//
//   - A Set is the menu of all renderable columns for one resource.
//   - Each Column has a stable Name, a display Header, a suggested Width, a
//     Default flag (visible without configuration), and an Extract closure
//     that pulls the cell value from a typed row.
//   - The user's config.yaml `columns: { sites: [...] }` overrides which
//     columns are visible AND their order. Unknown names are silently
//     skipped so column lists survive future renames.
package columns

// Column is a single named, rendered column for a Netbox resource.
type Column struct {
	// Name is the stable identifier used in config files (e.g. "name", "site").
	// Lowercase, no spaces, never changes for the lifetime of the resource.
	Name string

	// Header is the display label shown in the table (e.g. "Name", "Site").
	Header string

	// Width is the suggested column width in cells. The TUI uses it
	// directly; CLI table renderer uses it as a hint.
	Width int

	// Default marks columns that are visible when the user hasn't set an
	// override in config. False = available-but-hidden by default.
	Default bool

	// Extract pulls the cell value from a typed row. The row is passed as
	// `any` so the same closure powers reflection-light TUI and CLI paths;
	// the closure does a type assertion to the concrete Netbox type.
	Extract func(row any) string
}

// Set is the column menu for one resource.
type Set struct {
	// Resource is the stable identifier used as the key in config.Columns
	// and in the Registry map.
	Resource string

	// Columns lists every renderable column for the resource, in the
	// preferred default display order.
	Columns []Column
}

// VisibleNames returns the column names that should be displayed.
// If override is non-empty, those names win (in that order). Otherwise the
// default-flagged columns are used (in declared order).
func (s Set) VisibleNames(override []string) []string {
	if len(override) > 0 {
		return override
	}
	var defaults []string
	for _, c := range s.Columns {
		if c.Default {
			defaults = append(defaults, c.Name)
		}
	}
	return defaults
}

// VisibleColumns returns the actual Column values for VisibleNames,
// preserving override order. Unknown names are silently skipped.
func (s Set) VisibleColumns(override []string) []Column {
	wanted := s.VisibleNames(override)
	byName := make(map[string]Column, len(s.Columns))
	for _, c := range s.Columns {
		byName[c.Name] = c
	}
	out := make([]Column, 0, len(wanted))
	for _, n := range wanted {
		if c, ok := byName[n]; ok {
			out = append(out, c)
		}
	}
	return out
}

// Names returns every available column name for s — the menu of choices,
// regardless of default visibility.
func (s Set) Names() []string {
	out := make([]string, len(s.Columns))
	for i, c := range s.Columns {
		out[i] = c.Name
	}
	return out
}

// Registry returns all built-in column sets keyed by Resource name. The
// keys match what config.yaml uses under `columns: { <key>: [...] }`.
// Callers should NOT mutate the returned map or sets; treat them as fixed.
func Registry() map[string]Set {
	return map[string]Set{
		"sites":            SitesSet(),
		"racks":            RacksSet(),
		"devices":          DevicesSet(),
		"interfaces":       InterfacesSet(),
		"prefixes":         PrefixesSet(),
		"ip-addresses":     IPAddressesSet(),
		"vlans":            VLANsSet(),
		"vrfs":             VRFsSet(),
		"tenants":          TenantsSet(),
		"contacts":         ContactsSet(),
		"virtual-machines": VMsSet(),
		"clusters":         ClustersSet(),
		"search":           SearchSet(),
	}
}

// Resolve returns the visible columns for resource according to overrides
// (from config.Columns). When the resource isn't in the registry, returns
// nil — callers should treat that as "render nothing" or use a fallback.
func Resolve(resource string, override []string) []Column {
	set, ok := Registry()[resource]
	if !ok {
		return nil
	}
	return set.VisibleColumns(override)
}

// col builds a default-visible Column. Used by the per-resource Set
// factories to keep their declarations dense.
func col(name, header string, width int, extract func(any) string) Column {
	return Column{Name: name, Header: header, Width: width, Default: true, Extract: extract}
}

// opt builds an available-but-hidden Column (Default == false). The user
// opts these in via config.Columns.
func opt(name, header string, width int, extract func(any) string) Column {
	return Column{Name: name, Header: header, Width: width, Default: false, Extract: extract}
}
