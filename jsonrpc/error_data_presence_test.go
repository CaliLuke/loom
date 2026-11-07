package jsonrpc

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestErrorEnvelopeDataPresence(t *testing.T) {
	for _, tc := range []struct {
		name string
		data any
		want string
	}{
		{"absent", nil, ""},
		{"nullable null", loom.NullValue[string](), "null"},
		{"typed nil", (*string)(nil), "null"},
		{"string", "", `""`},
		{"array", []string{}, "[]"},
		{"object", map[string]int{}, "{}"},
		{"false", false, "false"},
		{"zero", 0, "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(ErrorResponse{Code: 4001, Message: "failure", Data: tc.data})
			require.NoError(t, err)
			var members map[string]jsontext.Value
			require.NoError(t, json.Unmarshal(data, &members))
			require.Equal(t, tc.want, string(members["data"]))
			raw := RawErrorResponse{Code: 4001, Message: "failure"}
			if tc.want != "" {
				raw.Data = jsontext.Value(tc.want)
			}
			encoded, err := json.Marshal(raw)
			require.NoError(t, err)
			require.Equal(t, string(data), string(encoded), "raw relay preserves the same presence contract")
		})
	}
	_, err := json.Marshal(ErrorResponse{Code: 4001, Data: loom.Nullable[string]{}})
	require.ErrorIs(t, err, loom.ErrAbsentNullable)
}
