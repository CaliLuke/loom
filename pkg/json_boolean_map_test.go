package loom

import (
	"encoding/json/v2"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type authoredBoolean bool

func TestPresenceBooleanMapJSON(t *testing.T) {
	for _, tc := range []struct {
		name     string
		value    any
		newValue func() any
	}{
		{"optional", OptionalValue(map[bool]bool{false: true, true: false}), func() any {
			return new(Optional[map[bool]bool])
		}},
		{"nullable", NullableValue(map[bool]bool{false: true, true: false}), func() any {
			return new(Nullable[map[bool]bool])
		}},
		{"nested", OptionalValue(NullableValue(map[bool]bool{false: true, true: false})), func() any {
			return new(Optional[Nullable[map[bool]bool]])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(tc.value, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, `{"false":true,"true":false}`, string(encoded))
			decoded := tc.newValue()
			require.NoError(t, json.Unmarshal(encoded, decoded))
			roundTrip, err := json.Marshal(decoded, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, encoded, roundTrip)
			for _, invalid := range []string{`{"TRUE":false}`, `{"1":false}`, `{"true":"false"}`, `{"true":false,"true":true}`} {
				require.Error(t, json.Unmarshal([]byte(invalid), tc.newValue()), invalid)
			}
		})
	}
}

func TestPresenceBooleanMapPreservesCustomKeyCodec(t *testing.T) {
	want := OptionalValue(NullableValue(map[authoredBoolean]bool{true: false}))
	encoded, err := json.Marshal(want, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, `{"authored":false}`, string(encoded))
	var got Optional[Nullable[map[authoredBoolean]bool]]
	require.NoError(t, json.Unmarshal(encoded, &got))
	require.Equal(t, want, got)
}

// MarshalText chooses an application-owned map key representation.
func (value *authoredBoolean) MarshalText() ([]byte, error) {
	return []byte("authored"), nil
}

// UnmarshalText decodes the application-owned map key representation.
func (value *authoredBoolean) UnmarshalText(data []byte) error {
	if string(data) != "authored" {
		return errors.New("expected authored key")
	}
	*value = true
	return nil
}
