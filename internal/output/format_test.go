package output

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    Format
		wantErr bool
	}{
		{"", "", false},
		{"table", FormatTable, false},
		{"TBL", FormatTable, false},
		{"  JSON  ", FormatJSON, false},
		{"yaml", FormatYAML, false},
		{"yml", FormatYAML, false},
		{"tsv", FormatTSV, false},
		{"plain", FormatTSV, false},
		{"toml", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := Parse(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolve_ExplicitWins(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	assert.Equal(t, FormatYAML, Resolve(FormatYAML, &buf))
}

func TestResolve_NonTTYDefaultsJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer // never a TTY
	assert.Equal(t, FormatJSON, Resolve("", &buf))
}

func TestResolve_NonFileWriter(t *testing.T) {
	t.Parallel()
	// /dev/null as *os.File is not a terminal, so resolve → JSON.
	f, err := os.Open(os.DevNull)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	assert.Equal(t, FormatJSON, Resolve("", f))
}

type row struct {
	Name string `json:"name" yaml:"name"`
	N    int    `json:"n" yaml:"n"`
}

func TestRender_AllFormats(t *testing.T) {
	t.Parallel()
	cols := []Column{
		{Header: "Name", Extract: func(r any) string { return r.(row).Name }},
		{Header: "N", Extract: func(r any) string { return "42" }},
	}
	rows := []row{{Name: "alpha", N: 1}, {Name: "beta", N: 2}}

	for _, f := range []Format{FormatTable, FormatJSON, FormatYAML, FormatTSV} {
		t.Run(string(f), func(t *testing.T) {
			r, err := New(f)
			require.NoError(t, err)
			var buf bytes.Buffer
			require.NoError(t, r.Render(&buf, cols, rows))
			assert.Contains(t, buf.String(), "alpha")
			assert.Contains(t, buf.String(), "beta")
		})
	}
}

func TestTSV_StripsEmbeddedTabs(t *testing.T) {
	t.Parallel()
	cols := []Column{
		{Header: "Name", Extract: func(r any) string { return r.(row).Name }},
	}
	rows := []row{{Name: "a\tb\nc"}}
	r, _ := New(FormatTSV)
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, cols, rows))
	// One header line + one row, no extra splits inside the value.
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 2)
	assert.Equal(t, "a b c", lines[1])
}
