package cmd

import (
	"errors"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/output"
	"github.com/ravinald/nbcli/internal/pager"
)

// runPager bridges from a show command's IO + resolved column set to the
// less-like pager. Each show command builds a typed Fetcher closure and
// hands it here; the closure captures the user's positional filters from
// kv plus the *netbox.Client, applying the pager's per-iteration overrides
// (Offset/Limit/Query) on top.
func runPager(io IO, title string, cols []output.Column, fetch pager.Fetcher) error {
	stdout, ok := io.Out.(*os.File)
	if !ok {
		return errors.New("pager: --interactive requires the real os.Stdout (got an in-memory writer)")
	}
	stdin, ok := io.In.(*os.File)
	if !ok {
		return errors.New("pager: --interactive requires the real os.Stdin (got an in-memory reader)")
	}
	return pager.Run(pager.Config{
		Title:   title,
		Columns: cols,
		Out:     stdout,
		In:      stdin,
	}, fetch)
}

// interactiveFlag returns true when --interactive / -i is set on cmd. Lookup
// goes through cobra's inherited-flags machinery so the flag can be declared
// once on the root command and reach every show subcommand.
func interactiveFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("interactive")
	return v
}

// applyPagerQuery sets the search query into the typed list options' Extra
// values, creating Extra if nil. Used by every show command's pager-fetcher
// closure to translate pager.FetchOpts.Query into Netbox's `?q=` filter.
func applyPagerQuery(extra *url.Values, query string) {
	if query == "" {
		return
	}
	if *extra == nil {
		*extra = url.Values{}
	}
	extra.Set("q", query)
}
