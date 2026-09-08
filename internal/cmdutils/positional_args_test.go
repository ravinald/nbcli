package cmdutils

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSpec = []KeywordSpec{
	{Name: "name"},
	{Name: "status", Values: []string{"active", "planned", "decommissioning"}},
	{Name: "region"},
	{Name: "limit"},
	{Name: "pager", NoValue: true, Description: "interactive pager"},
}

func TestParseShowArgs_Empty(t *testing.T) {
	t.Parallel()
	out, err := ParseShowArgs(nil, testSpec)
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestParseShowArgs_PairsAnyOrder(t *testing.T) {
	t.Parallel()
	a, err := ParseShowArgs([]string{"name", "hq", "status", "active"}, testSpec)
	require.NoError(t, err)
	b, err := ParseShowArgs([]string{"status", "active", "name", "hq"}, testSpec)
	require.NoError(t, err)
	assert.Equal(t, a, b)
	assert.Equal(t, "hq", a["name"])
	assert.Equal(t, "active", a["status"])
}

func TestParseShowArgs_ValueKeywordMissingValue(t *testing.T) {
	t.Parallel()
	_, err := ParseShowArgs([]string{"name", "hq", "status"}, testSpec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `keyword "status" expects a value`)
}

func TestParseShowArgs_SwitchKeyword(t *testing.T) {
	t.Parallel()
	out, err := ParseShowArgs([]string{"name", "hq", "pager"}, testSpec)
	require.NoError(t, err)
	assert.Equal(t, "hq", out["name"])
	assert.Equal(t, "true", out["pager"])
}

func TestParseShowArgs_SwitchInterleaved(t *testing.T) {
	t.Parallel()
	// pager (switch) can appear anywhere in the arg stream.
	out, err := ParseShowArgs([]string{"pager", "status", "active", "name", "hq"}, testSpec)
	require.NoError(t, err)
	assert.Equal(t, "true", out["pager"])
	assert.Equal(t, "active", out["status"])
	assert.Equal(t, "hq", out["name"])
}

func TestParseShowArgs_SwitchDuplicate(t *testing.T) {
	t.Parallel()
	_, err := ParseShowArgs([]string{"pager", "pager"}, testSpec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate keyword "pager"`)
}

func TestParseShowArgs_UnknownKeyword(t *testing.T) {
	t.Parallel()
	_, err := ParseShowArgs([]string{"naem", "hq"}, testSpec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown keyword "naem"`)
	// Allowed list shown for discoverability.
	assert.Contains(t, err.Error(), "limit")
	assert.Contains(t, err.Error(), "region")
}

func TestParseShowArgs_Duplicate(t *testing.T) {
	t.Parallel()
	_, err := ParseShowArgs([]string{"name", "a", "name", "b"}, testSpec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate keyword "name"`)
}

func TestValidator_BlocksRunE(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{
		Use:  "x",
		Args: Validator(testSpec),
		RunE: func(_ *cobra.Command, _ []string) error { return nil },
	}
	cmd.SetArgs([]string{"naem", "hq"})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestUsageLine_Stable(t *testing.T) {
	t.Parallel()
	got := UsageLine(testSpec)
	// Value-taking keywords first, switches segregated, both alphabetical.
	assert.Equal(t, "[limit|name|region|status <value>]... [pager]...", got)
}

func TestHelpTable_TagsSwitches(t *testing.T) {
	t.Parallel()
	out := HelpTable(testSpec)
	assert.Contains(t, out, "pager")
	assert.Contains(t, out, "(switch)")
}

func TestHelpTable_IncludesEveryKeyword(t *testing.T) {
	t.Parallel()
	out := HelpTable(testSpec)
	for _, k := range testSpec {
		assert.Contains(t, out, k.Name)
	}
}

func TestCompletion_KeywordsThenValuesThenUnused(t *testing.T) {
	t.Parallel()
	fn := CompletionFunc(testSpec)

	// No args yet → all keywords.
	kw, _ := fn(nil, nil, "")
	assert.ElementsMatch(t, []string{"name", "status", "region", "limit", "pager"}, kw)

	// One value-taking keyword typed → suggest its static values.
	vals, _ := fn(nil, []string{"status"}, "")
	assert.Equal(t, []string{"active", "planned", "decommissioning"}, vals)

	// Free-form value → empty completion (don't lie about valid input).
	free, _ := fn(nil, []string{"name"}, "")
	assert.Empty(t, free)

	// Once a pair is done, the keyword is not offered again.
	next, _ := fn(nil, []string{"name", "hq"}, "")
	assert.NotContains(t, strings.Join(next, ","), "name")
	assert.Contains(t, strings.Join(next, ","), "status")
}

func TestCompletion_SwitchAdvancesOneSlot(t *testing.T) {
	t.Parallel()
	fn := CompletionFunc(testSpec)

	// After a switch, we're at a keyword position again — not stuck waiting
	// for a value that doesn't exist.
	next, _ := fn(nil, []string{"pager"}, "")
	joined := strings.Join(next, ",")
	assert.Contains(t, joined, "name")
	assert.Contains(t, joined, "status")
	assert.NotContains(t, joined, "pager")
}
