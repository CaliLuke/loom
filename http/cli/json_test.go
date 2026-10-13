package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type customCLIValue string

func TestUnmarshalNonNullableJSON(t *testing.T) {
	for _, input := range []string{"", " \t\r\n", "{", "{\"value\":", "null", " \t null\r\n"} {
		t.Run(input, func(t *testing.T) {
			value := map[string]int{"retained": 7}
			require.Error(t, UnmarshalNonNullableJSON([]byte(input), &value))
			require.Equal(t, map[string]int{"retained": 7}, value)
		})
	}
	t.Run("nil destination", func(t *testing.T) {
		var value *string
		require.Error(t, UnmarshalNonNullableJSON([]byte(`"value"`), value))
	})
	t.Run("boolean keys", func(t *testing.T) {
		var value map[bool]int
		require.NoError(t, UnmarshalNonNullableJSON([]byte(`{"true":0,"false":7}`), &value))
		require.Equal(t, map[bool]int{true: 0, false: 7}, value)
	})
	t.Run("empty", func(t *testing.T) {
		var value []int
		require.NoError(t, UnmarshalNonNullableJSON([]byte(`[]`), &value))
		require.NotNil(t, value)
		require.Empty(t, value)
	})
	t.Run("custom decoder", func(t *testing.T) {
		var value customCLIValue
		require.NoError(t, UnmarshalNonNullableJSON([]byte(`"value"`), &value))
		require.Equal(t, customCLIValue(`"value"`), value)
	})
}

// UnmarshalJSON records the original JSON bytes to exercise custom codec dispatch.
func (v *customCLIValue) UnmarshalJSON(data []byte) error {
	*v = customCLIValue(data)
	return nil
}
