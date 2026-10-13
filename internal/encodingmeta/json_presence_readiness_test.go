package encodingmeta

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONPresenceReadiness(t *testing.T) {
	empty := ""
	populated := "value"
	for _, tc := range []struct {
		name    string
		value   any
		present bool
	}{
		{"absent string", (*string)(nil), false},
		{"empty string", &empty, true},
		{"populated string", &populated, true},
		{"absent bytes", []byte(nil), false},
		{"empty bytes", []byte{}, true},
		{"populated bytes", []byte("value"), true},
		{"absent array", []string(nil), false},
		{"empty array", []string{}, true},
		{"populated array", []string{"value"}, true},
		{"absent map", map[string]string(nil), false},
		{"empty map", map[string]string{}, true},
		{"populated map", map[string]string{"key": "value"}, true},
		{"absent object", (*struct{})(nil), false},
		{"empty object", &struct{}{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := ResolveJSONField(nil, "value", JSONFieldPolicy{Optional: true, Storage: JSONNilable})
			tag := field.Tag
			require.True(t, field.Present)
			typ := reflect.StructOf([]reflect.StructField{{
				Name: "Value", Type: reflect.TypeOf(tc.value), Tag: reflect.StructTag(`json:"` + tag + `"`),
			}})
			value := reflect.New(typ).Elem()
			value.Field(0).Set(reflect.ValueOf(tc.value))
			wire, err := json.Marshal(value.Interface())
			require.NoError(t, err)
			var decoded map[string]jsontext.Value
			require.NoError(t, json.Unmarshal(wire, &decoded))
			_, present := decoded["value"]
			require.Equal(t, tc.present, present, "%s", wire)
		})
	}
}
