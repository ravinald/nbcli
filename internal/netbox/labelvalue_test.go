package netbox

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelValue_UnmarshalString(t *testing.T) {
	t.Parallel()
	var lv LabelValue
	require.NoError(t, json.Unmarshal([]byte(`{"value":"active","label":"Active"}`), &lv))
	assert.Equal(t, "active", lv.Value)
	assert.Equal(t, "Active", lv.Label)
}

func TestLabelValue_UnmarshalIntFamily(t *testing.T) {
	t.Parallel()
	// IPAM prefixes/IP-addresses return family.value as a JSON number.
	var lv LabelValue
	require.NoError(t, json.Unmarshal([]byte(`{"value":4,"label":"IPv4"}`), &lv))
	assert.Equal(t, "4", lv.Value)
	assert.Equal(t, "IPv4", lv.Label)
}

func TestLabelValue_UnmarshalBool(t *testing.T) {
	t.Parallel()
	var lv LabelValue
	require.NoError(t, json.Unmarshal([]byte(`{"value":true,"label":"Yes"}`), &lv))
	assert.Equal(t, "true", lv.Value)
}

func TestLabelValue_UnmarshalMissingValue(t *testing.T) {
	t.Parallel()
	var lv LabelValue
	require.NoError(t, json.Unmarshal([]byte(`{"label":"Active"}`), &lv))
	assert.Empty(t, lv.Value)
	assert.Equal(t, "Active", lv.Label)
}

func TestLabelValue_UnmarshalNullValue(t *testing.T) {
	t.Parallel()
	var lv LabelValue
	require.NoError(t, json.Unmarshal([]byte(`{"value":null,"label":"None"}`), &lv))
	assert.Empty(t, lv.Value)
}
