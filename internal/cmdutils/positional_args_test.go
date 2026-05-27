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

func TestParseShowArgs_OddArgcFails(t *testing.T) {
	t.Parallel()
	_, err := ParseShowArgs([]string{"name", "hq", "status"}, testSpec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "keyword/value pairs")
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
	// Sorted alphabetically.
	assert.Equal(t, "[limit|name|region|status <value>]...", got)
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
	assert.ElementsMatch(t, []string{"name", "status", "region", "limit"}, kw)

	// One keyword typed → suggest its static values.
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
