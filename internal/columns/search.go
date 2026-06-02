package columns

import (
	"encoding/json"
	"strconv"

	"github.com/ravinald/nbcli/internal/netbox"
)

// searchObjectShim is the LCD subset every Netbox object guarantees in its
// embedded payload. Decoded lazily inside the extractors so we don't pay for
// it unless the column is visible.
type searchObjectShim struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	Display string `json:"display"`
}

func searchObject(r any) searchObjectShim {
	hit, ok := r.(netbox.SearchResult)
	if !ok || len(hit.Object) == 0 {
		return searchObjectShim{}
	}
	var s searchObjectShim
	_ = json.Unmarshal(hit.Object, &s)
	return s
}

// SearchSet returns the column menu for /api/search/ hits.
//
// Defaults — type, field, value, display — answer "what kind of thing, why
// did it match, and what's its name?" in a single row. The id/url columns
// are available but hidden because they're rarely useful at a glance and
// add visual noise across heterogeneous result sets.
func SearchSet() Set {
	return Set{
		Resource: "search",
		Columns: []Column{
			col("type", "Type", 22, func(r any) string { return r.(netbox.SearchResult).Type }),
			col("field", "Field", 16, func(r any) string { return r.(netbox.SearchResult).Field }),
			col("value", "Value", 28, func(r any) string { return r.(netbox.SearchResult).Value }),
			col("display", "Display", 28, func(r any) string { return searchObject(r).Display }),
			opt("id", "ID", 8, func(r any) string { return strconv.Itoa(searchObject(r).ID) }),
			opt("url", "URL", 48, func(r any) string { return searchObject(r).URL }),
		},
	}
}
