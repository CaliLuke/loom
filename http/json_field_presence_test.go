package http

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

type jsonFieldEmptyMap map[string]int

func (jsonFieldEmptyMap) MarshalJSON() ([]byte, error) {
	return []byte(`{"authored":true}`), nil
}

func TestJSONFieldCollectionPresence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  string
	}{
		{"nil map", map[string]int(nil), ""},
		{"nil slice", []int(nil), ""},
		{"empty map", map[string]int{}, `{}`},
		{"empty slice", []int{}, `[]`},
		{"custom empty map", jsonFieldEmptyMap{}, `{"authored":true}`},
		{"map", map[string]int{"count": 0}, `{"count":0}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := url.Values{}
			present, err := EncodeJSONField(values, "value", tc.value, FieldValue)
			require.NoError(t, err)
			require.Equal(t, tc.want != "", present)
			if !present {
				require.Empty(t, values)
				return
			}
			require.Equal(t, []string{tc.want}, values["value"])
		})
	}
}
