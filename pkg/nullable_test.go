package loom

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNullableJSONPresence(t *testing.T) {
	type document struct {
		Value Nullable[string] `json:"value"`
	}

	var missing document
	require.NoError(t, json.Unmarshal([]byte(`{}`), &missing))
	require.False(t, missing.Value.Present())

	var null document
	require.NoError(t, json.Unmarshal([]byte(`{"value":null}`), &null))
	require.True(t, null.Value.Present())
	require.True(t, null.Value.IsNull())

	var concrete document
	require.NoError(t, json.Unmarshal([]byte(`{"value":"loom"}`), &concrete))
	value, ok := concrete.Value.Value()
	require.True(t, ok)
	require.Equal(t, "loom", value)

	encoded, err := json.Marshal(document{Value: NullValue[string]()})
	require.NoError(t, err)
	require.JSONEq(t, `{"value":null}`, string(encoded))

	type optionalDocument struct {
		Value Nullable[string] `json:"value,omitzero"`
	}
	encoded, err = json.Marshal(optionalDocument{})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(encoded))
	encoded, err = json.Marshal(optionalDocument{Value: NullValue[string]()})
	require.NoError(t, err)
	require.JSONEq(t, `{"value":null}`, string(encoded))
}
