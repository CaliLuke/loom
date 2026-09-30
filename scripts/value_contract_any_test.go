package scripts_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

// TestValueContractAnyMaterialization exercises the actual generated Any carrier.
// Native Go any is not a substitute: Loom JSONValue retains numeric spellings.
func TestValueContractAnyMaterialization(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input any
		wire  string
	}{
		{name: "nil bytes", input: []byte(nil), wire: `""`},
		{name: "empty bytes", input: []byte{}, wire: `""`},
		{name: "binary bytes", input: []byte("hi"), wire: `"aGk="`},
		{name: "literal base64 string", input: "aGk=", wire: `"aGk="`},
		{name: "literal plain string", input: "hi", wire: `"hi"`},
		{name: "nil array", input: []string(nil), wire: `[]`},
		{name: "nil map", input: map[string]any(nil), wire: `{}`},
		{name: "exact large integer", input: int64(9007199254740993), wire: `9007199254740993`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			materialized, err := loom.JSONValueFrom(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.wire, string(materialized))
			var decoded loom.JSONValue
			require.NoError(t, json.Unmarshal(materialized, &decoded))
			require.Equal(t, tc.wire, string(decoded))
			require.True(t, loom.JSONValueEqual(decoded, tc.input))
			reencoded, err := json.Marshal(decoded)
			require.NoError(t, err)
			require.Equal(t, tc.wire, string(reencoded))
		})
	}
}
