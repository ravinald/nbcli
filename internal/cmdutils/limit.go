// limit.go carries the REST-API pagination convention used by every `show`
// command. It deliberately lives outside positional_args.go so the generic
// Junos-style parser is cleanly separable when it gets extracted into the
// standalone github.com/ravinald/positional-args module. See
// docs-internal/positional-args-extraction.md.

package cmdutils

import (
	"fmt"
	"strconv"
)

// LimitKeyword and OffsetKeyword are the conventional positional-arg names
// every nbcli list command accepts. Const so callers can't typo and silently
// miss a filter at runtime.
const (
	LimitKeyword  = "limit"
	OffsetKeyword = "offset"
)

// PaginationKeywords returns the two KeywordSpec entries every nbcli list
// command appends to its own filters. Convention: limit 0 means "all pages"
// (see ApplyLimitOffset).
//
// Usage:
//
//	var siteKeywords = append([]cmdutils.KeywordSpec{
//	    {Name: "name",   Description: "exact site name"},
//	    // ... resource-specific filters ...
//	}, cmdutils.PaginationKeywords()...)
func PaginationKeywords() []KeywordSpec {
	return []KeywordSpec{
		{Name: LimitKeyword, Description: "page size (0 = all pages)", Example: "50"},
		{Name: OffsetKeyword, Description: "page offset"},
	}
}

// PagerKeyword is the switch-style positional keyword every show command
// supports for opening the less-like interactive pager. Junos-shaped:
//
//	nbcli show sites pager
//	nbcli show devices status active pager
func PagerKeyword() KeywordSpec {
	return KeywordSpec{
		Name:        "pager",
		Description: "open the less-like interactive pager (n/p step, / search, g goto, q quit)",
		NoValue:     true,
	}
}

// FormatKeyword is the positional `format <value>` keyword every show + search
// command supports. Junos-shaped output-modifier; wins over cfg.Format /
// NBCLI_FORMAT. Static enum so shell completion offers the choices.
func FormatKeyword() KeywordSpec {
	return KeywordSpec{
		Name:        "format",
		Description: "output format (overrides config and NBCLI_FORMAT)",
		Example:     "json",
		Values:      []string{"table", "json", "yaml", "tsv"},
	}
}

// ColumnsKeyword is the positional `columns <comma-list>` keyword every
// show + search command supports. Wins over cfg.Columns[resource], which
// wins over the registry's default-visible set. Values are free-form per
// resource — `nbcli columns <resource>` lists what's available.
func ColumnsKeyword() KeywordSpec {
	return KeywordSpec{
		Name:        "columns",
		Description: "comma-separated column names (overrides config)",
		Example:     "id,name,status",
	}
}

// PresentationKeywords returns the trio every show + search command appends:
// format (value), columns (value), pager (switch). Bundled because they
// always travel together and adding a new presentation modifier should touch
// one place.
func PresentationKeywords() []KeywordSpec {
	return []KeywordSpec{
		FormatKeyword(),
		ColumnsKeyword(),
		PagerKeyword(),
	}
}

// TrailingKeywords is the standard tail every show + search command appends
// to its resource-specific filters: pagination (limit/offset) + presentation
// (format/columns/pager). One call site keeps the trailing grammar uniform
// across the CLI surface.
func TrailingKeywords() []KeywordSpec {
	out := PaginationKeywords()
	return append(out, PresentationKeywords()...)
}

// ApplyLimitOffset parses the "limit" and "offset" keywords out of kv into
// *limit and *offset. Convention used by every `show` command:
//
//	limit 0   → fetchAll == true; caller should auto-paginate via netbox.ListAll
//	limit N   → fetchAll == false; *limit = N (single page)
//	missing   → fetchAll == false; *limit untouched
//
// Negative values are rejected with a clear error.
func ApplyLimitOffset(kv map[string]string, limit, offset *int) (fetchAll bool, err error) {
	if v, ok := kv[LimitKeyword]; ok {
		n, perr := strconv.Atoi(v)
		if perr != nil {
			return false, fmt.Errorf("limit must be an integer: %w", perr)
		}
		if n < 0 {
			return false, fmt.Errorf("limit must be >= 0 (got %d)", n)
		}
		if n == 0 {
			fetchAll = true
		} else {
			*limit = n
		}
	}
	if v, ok := kv[OffsetKeyword]; ok {
		n, perr := strconv.Atoi(v)
		if perr != nil {
			return false, fmt.Errorf("offset must be an integer: %w", perr)
		}
		if n < 0 {
			return false, fmt.Errorf("offset must be >= 0 (got %d)", n)
		}
		*offset = n
	}
	return fetchAll, nil
}
