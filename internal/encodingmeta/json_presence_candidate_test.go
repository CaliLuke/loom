package encodingmeta

import (
	"encoding/json/v2"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type customZero string

func TestJSONPresenceCandidateStorage(t *testing.T) {
	value := customZero("supplied")
	flag := false
	for _, tc := range []struct {
		name  string
		value any
		wire  string
	}{
		{
			name: "ordinary nilable carrier",
			value: struct {
				Value []string `json:"value,omitzero"`
			}{Value: []string{}},
			wire: `{"value":[]}`,
		},
		{
			name: "native scalar must not use zero omission",
			value: struct {
				Value string `json:"value,omitzero"`
			}{},
			wire: `{}`,
		},
		{
			name: "native boolean zero omission",
			value: struct {
				Value bool `json:"value,omitzero"`
			}{},
			wire: `{}`,
		},
		{
			name: "native integer zero omission",
			value: struct {
				Value int `json:"value,omitzero"`
			}{},
			wire: `{}`,
		},
		{
			name: "supplied pointer zero is present",
			value: struct {
				Value *bool `json:"value,omitzero"`
			}{Value: &flag},
			wire: `{"value":false}`,
		},
		{
			name: "custom IsZero defeats pointer absence",
			value: struct {
				Value *customZero `json:"value,omitzero"`
			}{Value: &value},
			wire: `{}`,
		},
		{
			name: "wrapper retains custom codec value",
			value: struct {
				Value loom.Optional[customZero] `json:"value,omitzero"`
			}{Value: loom.OptionalValue(value)},
			wire: `{"value":"supplied"}`,
		},
		{
			name: "wrapper absence",
			value: struct {
				Value loom.Optional[customZero] `json:"value,omitzero"`
			}{},
			wire: `{}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := json.Marshal(tc.value)
			require.NoError(t, err)
			require.JSONEq(t, tc.wire, string(wire))
		})
	}
}

// IsZero deliberately disagrees with pointer absence to expose an insufficient
// global omitzero substitution in generated optional custom fields.
func (v customZero) IsZero() bool {
	return true
}
