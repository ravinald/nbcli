// Package cmdutils hosts shared helpers for nbcli's cobra commands.
//
// Its main export is the Junos-style positional keyword parser. nbcli reserves
// flags (--format, --url, --config, ...) for *altering how the tool operates*
// and uses positional keyword/value pairs for *interacting with the Netbox API*.
//
// Example:
//
//	nbcli show sites status active region us-west limit 100
//	                ^----- positional pairs -----^
//	                ^ network-engineer-friendly, Junos-shaped
//
// Each command declares its allowed keyword set with KeywordSpec; ParseShowArgs
// validates and unmarshals the slice into a map[string]string. Validator wraps
// the parser as a cobra.PositionalArgs so malformed input dies before RunE.
// CompletionFunc gives shell completion of unused keywords.
//
// This file is the future extraction target for the standalone module
// github.com/ravinald/positional-args (see docs-internal/positional-args-extraction.md).
// Anything REST-API-specific (pagination, filters, ...) lives in limit.go or
// elsewhere — never here.
package cmdutils

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// KeywordSpec describes one positional keyword a command accepts.
type KeywordSpec struct {
	// Name is the keyword as the user types it. Lowercase, no spaces.
	Name string

	// Description is the one-line help text shown in the command's long help.
	Description string

	// Example, when set, decorates the keyword in help output (e.g. "active").
	Example string

	// Values, when non-empty, is the static list of valid values used for
	// shell completion. Leave nil for free-form values.
	Values []string

	// NoValue marks a switch-style keyword that takes no value. The user
	// types just the keyword (e.g. `nbcli show sites pager`). Parsed into
	// the kv map as Name → "true".
	NoValue bool
}

// ParseShowArgs walks args as positional keywords into a map. Two shapes:
//
//   - value keyword: `name hq` → out["name"] = "hq"
//   - switch keyword (KeywordSpec.NoValue): `pager` → out["pager"] = "true"
//
// Errors on:
//
//   - unknown keyword — caught loudly so typos can't silently drop a filter
//   - duplicate keyword — caller should use one filter per attribute
//   - value keyword with no value (end-of-args)
//
// Order across keywords is free, so `name hq status active pager` and
// `pager status active name hq` are equivalent.
func ParseShowArgs(args []string, allowed []KeywordSpec) (map[string]string, error) {
	byName := make(map[string]KeywordSpec, len(allowed))
	for _, k := range allowed {
		byName[k.Name] = k
	}
	out := make(map[string]string)
	for i := 0; i < len(args); {
		kw := args[i]
		spec, known := byName[kw]
		if !known {
			return nil, fmt.Errorf("unknown keyword %q (expected one of: %s)", kw, allowedList(allowed))
		}
		if _, dup := out[kw]; dup {
			return nil, fmt.Errorf("duplicate keyword %q", kw)
		}
		if spec.NoValue {
			out[kw] = "true"
			i++
			continue
		}
		if i+1 >= len(args) {
			return nil, fmt.Errorf("keyword %q expects a value", kw)
		}
		out[kw] = args[i+1] //nolint:gosec // i+1 < len(args) per the check above
		i += 2
	}
	return out, nil
}

// Validator returns a cobra.PositionalArgs that runs ParseShowArgs for its
// validation side-effect. Attach to Command.Args.
func Validator(allowed []KeywordSpec) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		_, err := ParseShowArgs(args, allowed)
		return err
	}
}

// UsageLine builds a short usage suffix suitable for the cobra.Command.Use
// field. Value-taking keywords and switch-style keywords list separately:
//
//	[name|slug|status <value>]... [pager]...
func UsageLine(allowed []KeywordSpec) string {
	var withVal, switches []string
	for _, k := range allowed {
		if k.NoValue {
			switches = append(switches, k.Name)
		} else {
			withVal = append(withVal, k.Name)
		}
	}
	sort.Strings(withVal)
	sort.Strings(switches)
	var parts []string
	if len(withVal) > 0 {
		parts = append(parts, "["+strings.Join(withVal, "|")+" <value>]...")
	}
	if len(switches) > 0 {
		parts = append(parts, "["+strings.Join(switches, "|")+"]...")
	}
	return strings.Join(parts, " ")
}

// HelpTable renders an indented "keyword  description" block for Command.Long.
// Switch-style keywords (NoValue) are tagged "(switch)" so users know they
// take no value.
func HelpTable(allowed []KeywordSpec) string {
	width := 0
	for _, k := range allowed {
		if len(k.Name) > width {
			width = len(k.Name)
		}
	}
	var b strings.Builder
	b.WriteString("Positional filters (any order):\n")
	for _, k := range allowed {
		suffix := ""
		switch {
		case k.NoValue:
			suffix = " (switch)"
		case k.Example != "":
			suffix = " (e.g. " + k.Example + ")"
		}
		fmt.Fprintf(&b, "  %-*s  %s%s\n", width, k.Name, k.Description, suffix)
	}
	return b.String()
}

// CompletionFunc returns a cobra.ValidArgsFunction that completes the next
// positional argument. Walks the existing args, tracking which keywords are
// consumed (counting NoValue ones as one slot, value-taking ones as two).
// Returns unused keywords at a keyword position, or the Values list at a
// value position.
func CompletionFunc(allowed []KeywordSpec) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	byName := make(map[string]KeywordSpec, len(allowed))
	for _, k := range allowed {
		byName[k.Name] = k
	}
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		used := make(map[string]struct{})
		var pendingValueKw *KeywordSpec
		for i := 0; i < len(args); {
			kw := args[i]
			used[kw] = struct{}{}
			spec, known := byName[kw]
			if !known {
				i++
				pendingValueKw = nil
				continue
			}
			if spec.NoValue {
				i++
				pendingValueKw = nil
				continue
			}
			if i == len(args)-1 {
				// Value-taking keyword typed but no value yet → value position.
				s := spec
				pendingValueKw = &s
				break
			}
			i += 2
			pendingValueKw = nil
		}
		if pendingValueKw != nil {
			if len(pendingValueKw.Values) == 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return append([]string(nil), pendingValueKw.Values...), cobra.ShellCompDirectiveNoFileComp
		}
		out := make([]string, 0, len(allowed))
		for _, k := range allowed {
			if _, dup := used[k.Name]; !dup {
				out = append(out, k.Name)
			}
		}
		sort.Strings(out)
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

func allowedNames(allowed []KeywordSpec) []string {
	names := make([]string, len(allowed))
	for i, k := range allowed {
		names[i] = k.Name
	}
	sort.Strings(names)
	return names
}

func allowedList(allowed []KeywordSpec) string {
	return strings.Join(allowedNames(allowed), ", ")
}
