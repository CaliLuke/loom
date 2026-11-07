package http

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestJSONFieldNullPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, raw                      string
		exists, allowNull, valid, null bool
	}{
		{"absent", "", false, true, true, false},
		{"empty", "", true, true, false, false},
		{"whitespace", " \t\n", true, true, false, false},
		{"malformed", "}", true, true, false, false},
		{"truncated", "[", true, true, false, false},
		{"concrete", "3", true, true, true, false},
		{"null", "null", true, true, true, true},
		{"padded null", " \tnull\n", true, true, true, true},
		{"rejected null", "null", true, false, false, false},
		{"trailing", "3 4", true, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := url.Values{}
			if tc.exists {
				values["value"] = []string{tc.raw}
			}
			target := loom.NullableValue(7)
			present, err := DecodeJSONField(values, "value", &target, tc.allowNull, FieldValue)
			require.Equal(t, tc.exists, present)
			if !tc.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.null, target.IsNull())
			if !tc.null {
				actual, exists := target.Value()
				require.True(t, exists)
				if tc.exists {
					require.Equal(t, 3, actual)
				} else {
					require.Equal(t, 7, actual)
				}
			}
		})
	}
}
