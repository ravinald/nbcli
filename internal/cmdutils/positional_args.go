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
}

// ParseShowArgs walks args as [keyword value]* pairs into a map. It rejects:
//
//   - odd argc — a keyword with no value
//   - unknown keyword — caught loudly so typos can't silently drop a filter
//   - duplicate keyword — caller should use one filter per attribute
//
// Strict structure (keyword-then-value); order across pairs is free, so
// `name hq status active` and `status active name hq` are equivalent.
func ParseShowArgs(args []string, allowed []KeywordSpec) (map[string]string, error) {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, k := range allowed {
		allowedSet[k.Name] = struct{}{}
	}
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("expected keyword/value pairs (got %d args; last is %q)", len(args), args[len(args)-1])
	}
	out := make(map[string]string, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		kw, val := args[i], args[i+1] //nolint:gosec // len(args) is even per the check above

		if _, ok := allowedSet[kw]; !ok {
			return nil, fmt.Errorf("unknown keyword %q (expected one of: %s)", kw, allowedList(allowed))
		}
		if _, dup := out[kw]; dup {
			return nil, fmt.Errorf("duplicate keyword %q", kw)
		}
		out[kw] = val
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

// UsageLine builds a short "[kw1|kw2|... <value>]..." suffix suitable for the
// cobra.Command.Use field.
func UsageLine(allowed []KeywordSpec) string {
	return "[" + strings.Join(allowedNames(allowed), "|") + " <value>]..."
}

// HelpTable renders an indented "keyword  description" block for Command.Long.
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
		ex := ""
		if k.Example != "" {
			ex = " (e.g. " + k.Example + ")"
		}
		fmt.Fprintf(&b, "  %-*s  %s%s\n", width, k.Name, k.Description, ex)
	}
	return b.String()
}

// CompletionFunc returns a cobra.ValidArgsFunction that completes the next
// positional argument: at an even index it offers unused keywords; at an odd
// index it offers the static Values declared for the preceding keyword (or
// nothing if the keyword is free-form).
func CompletionFunc(allowed []KeywordSpec) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	byName := make(map[string]KeywordSpec, len(allowed))
	for _, k := range allowed {
		byName[k.Name] = k
	}
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		// Track keywords the user has already filled in to avoid offering them again.
		used := make(map[string]struct{})
		for i := 0; i+1 < len(args); i += 2 {
			used[args[i]] = struct{}{}
		}
		if len(args)%2 == 0 {
			// Even index → suggest a keyword.
			out := make([]string, 0, len(allowed))
			for _, k := range allowed {
				if _, dup := used[k.Name]; !dup {
					out = append(out, k.Name)
				}
			}
			sort.Strings(out)
			return out, cobra.ShellCompDirectiveNoFileComp
		}
		// Odd index → value for the keyword just typed.
		if k, ok := byName[args[len(args)-1]]; ok && len(k.Values) > 0 {
			return append([]string(nil), k.Values...), cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
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
