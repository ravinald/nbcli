package output

import (
	"bytes"
	"encoding/json"
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stRow struct {
	Name string `json:"name" yaml:"name"`
	N    int    `json:"n" yaml:"n"`
}

func seqOf(rows []stRow) iter.Seq[any] {
	return func(yield func(any) bool) {
		for _, r := range rows {
			if !yield(r) {
				return
			}
		}
	}
}

func TestJSONStream_MatchesBatchOutput(t *testing.T) {
	t.Parallel()
	rows := []stRow{{Name: "alpha", N: 1}, {Name: "beta", N: 2}}
	cols := []Column{{Header: "Name"}, {Header: "N"}}

	var batch bytes.Buffer
	require.NoError(t, jsonRenderer{}.Render(&batch, cols, rows))

	var stream bytes.Buffer
	require.NoError(t, jsonRenderer{}.Stream(&stream, cols, seqOf(rows)))

	// Both produce the same logical structure.
	var batchDoc, streamDoc any
	require.NoError(t, json.Unmarshal(batch.Bytes(), &batchDoc))
	require.NoError(t, json.Unmarshal(stream.Bytes(), &streamDoc))
	assert.Equal(t, batchDoc, streamDoc)
}

func TestYAMLStream_ContainsAllRows(t *testing.T) {
	t.Parallel()
	rows := []stRow{{Name: "alpha", N: 1}, {Name: "beta", N: 2}}
	var buf bytes.Buffer
	require.NoError(t, yamlRenderer{}.Stream(&buf, nil, seqOf(rows)))
	out := buf.String()
	assert.Contains(t, out, "- name: alpha")
	assert.Contains(t, out, "- name: beta")
	// yaml.v3 quotes "n" because it's a reserved scalar; both renderings are valid.
	assert.Contains(t, out, ": 1")
	assert.Contains(t, out, ": 2")
}

func TestTSVStream_HeaderAndRows(t *testing.T) {
	t.Parallel()
	cols := []Column{
		{Header: "Name", Extract: func(r any) string { return r.(stRow).Name }},
		{Header: "N", Extract: func(r any) string { return "42" }},
	}
	rows := []stRow{{Name: "alpha"}, {Name: "beta"}}
	var buf bytes.Buffer
	require.NoError(t, tsvRenderer{}.Stream(&buf, cols, seqOf(rows)))
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, "name\tn", lines[0])
	assert.Equal(t, "alpha\t42", lines[1])
	assert.Equal(t, "beta\t42", lines[2])
}

func TestJSONStream_EarlyStop(t *testing.T) {
	t.Parallel()
	rows := []stRow{{Name: "alpha"}, {Name: "beta"}, {Name: "gamma"}}
	stopAfter := func(rows []stRow, n int) iter.Seq[any] {
		return func(yield func(any) bool) {
			for i, r := range rows {
				if i >= n {
					return
				}
				if !yield(r) {
					return
				}
			}
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jsonRenderer{}.Stream(&buf, nil, stopAfter(rows, 2)))
	var doc []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))
	require.Len(t, doc, 2)
	names := []string{doc[0]["name"].(string), doc[1]["name"].(string)}
	assert.True(t, slices.Contains(names, "alpha"))
	assert.True(t, slices.Contains(names, "beta"))
}
