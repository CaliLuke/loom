package encodingmeta

import (
	"encoding/json/v2"
	"reflect"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestJSONPresenceReadinessControls(t *testing.T) {
	for _, tc := range []struct {
		name     string
		value    any
		optional bool
		explicit bool
		want     string
	}{
		{"required false", false, false, false, `{"value":false}`},
		{"required zero", 0, false, false, `{"value":0}`},
		{"required empty string", "", false, false, `{"value":""}`},
		{"required empty array", []string{}, false, false, `{"value":[]}`},
		{"optional absent", loom.Optional[string]{}, true, true, `{}`},
		{"optional empty", loom.OptionalValue(""), true, true, `{"value":""}`},
		{"nullable absent", loom.Nullable[string]{}, true, true, `{}`},
		{"nullable null", loom.NullValue[string](), true, true, `{"value":null}`},
		{"nullable empty", loom.NullableValue(""), true, true, `{"value":""}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := ResolveJSONField(nil, "value", JSONFieldPolicy{Optional: tc.optional, Storage: JSONPresence, Presence: tc.explicit})
			tag := field.Tag
			require.True(t, field.Present)
			typ := reflect.StructOf([]reflect.StructField{{Name: "Value", Type: reflect.TypeOf(tc.value), Tag: reflect.StructTag(`json:"` + tag + `"`)}})
			value := reflect.New(typ).Elem()
			value.Field(0).Set(reflect.ValueOf(tc.value))
			wire, err := json.Marshal(value.Interface())
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(wire))
		})
	}
}
