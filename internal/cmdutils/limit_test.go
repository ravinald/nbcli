package cmdutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginationKeywords_ShapeIsStable(t *testing.T) {
	t.Parallel()
	kws := PaginationKeywords()
	require.Len(t, kws, 2)
	assert.Equal(t, LimitKeyword, kws[0].Name)
	assert.Equal(t, OffsetKeyword, kws[1].Name)
	// Help text is non-empty so HelpTable() emits something useful.
	assert.NotEmpty(t, kws[0].Description)
	assert.NotEmpty(t, kws[1].Description)
}

func TestApplyLimitOffset(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		kv         map[string]string
		startLimit int
		startOff   int
		wantLimit  int
		wantOff    int
		wantAll    bool
		wantErr    bool
	}{
		{name: "missing keeps defaults", kv: nil, startLimit: 50, wantLimit: 50},
		{name: "explicit page size", kv: map[string]string{"limit": "100"}, startLimit: 50, wantLimit: 100},
		{name: "limit 0 means all", kv: map[string]string{"limit": "0"}, startLimit: 50, wantLimit: 50, wantAll: true},
		{name: "offset only", kv: map[string]string{"offset": "25"}, startLimit: 50, wantLimit: 50, wantOff: 25},
		{name: "both", kv: map[string]string{"limit": "10", "offset": "5"}, wantLimit: 10, wantOff: 5},
		{name: "non-integer limit fails", kv: map[string]string{"limit": "ten"}, wantErr: true},
		{name: "negative limit fails", kv: map[string]string{"limit": "-1"}, wantErr: true},
		{name: "negative offset fails", kv: map[string]string{"offset": "-1"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			limit, offset := tc.startLimit, tc.startOff
			all, err := ApplyLimitOffset(tc.kv, &limit, &offset)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantAll, all)
			assert.Equal(t, tc.wantLimit, limit)
			assert.Equal(t, tc.wantOff, offset)
		})
	}
}
