package encodingmeta

import (
	"encoding/json/v2"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONFieldStorage(t *testing.T) {
	empty, populated := "", "value"
	custom := customZero("supplied")
	for _, tc := range []struct {
		name     string
		value    any
		storage  JSONStorage
		full     string
		optional bool
		action   JSONOmission
		want     string
	}{
		{"custom IsZero remains opaque", &custom, JSONCustom, "", true, JSONExternalCodec, `{"value":"supplied"}`},
		{"authored custom zero remains external", &custom, JSONCustom, "value,omitzero", true, JSONExternalCodec, `{}`},
		{"native false", false, JSONNative, "", true, JSONRetain, `{"value":false}`},
		{"native zero", 0, JSONNative, "", true, JSONRetain, `{"value":0}`},
		{"native empty", "", JSONNative, "", true, JSONRetain, `{"value":""}`},
		{"pointer empty", &empty, JSONNilable, "", true, JSONOmitAbsent, `{"value":""}`},
		{"nil pointer", (*string)(nil), JSONNilable, "", true, JSONOmitAbsent, `{}`},
		{"empty collection", []string{}, JSONNilable, "", true, JSONOmitAbsent, `{"value":[]}`},
		{"required collection", []string{}, JSONNilable, "", false, JSONRetain, `{"value":[]}`},
		{"authored native zero", false, JSONNative, "value,omitzero", true, JSONOmitNativeZero, `{}`},
		{"authored native empty", "", JSONNative, "value,omitzero", true, JSONOmitNativeZero, `{}`},
		{"authored native empty omission", false, JSONNative, "value,omitempty", true, JSONOmitEmpty, `{"value":false}`},
		{"authored predicates forward", false, JSONNative, "value,omitempty,omitzero", true, JSONOmitEmptyOrNativeZero, `{}`},
		{"authored predicates reverse", false, JSONNative, "value,omitzero,omitempty", true, JSONOmitEmptyOrNativeZero, `{}`},
		{"authored nilable empty", &empty, JSONNilable, "value,omitempty", true, JSONOmitEmpty, `{}`},
		{"authored nilable zero", &empty, JSONNilable, "value,omitzero", true, JSONOmitAbsent, `{"value":""}`},
		{"authored literal dash", &populated, JSONNilable, "-,omitempty", true, JSONOmitEmpty, `{"-":"value"}`},
		{"authored exact skip", &empty, JSONNilable, "-", true, JSONRetain, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var meta map[string][]string
			if tc.full != "" {
				meta = map[string][]string{"struct:tag:json": {tc.full}}
			}
			field := ResolveJSONField(meta, "value", JSONFieldPolicy{Storage: tc.storage, Optional: tc.optional})
			require.Equal(t, tc.action, field.Omission)
			typ := reflect.StructOf([]reflect.StructField{{Name: "Value", Type: reflect.TypeOf(tc.value), Tag: reflect.StructTag(`json:"` + field.Tag + `"`)}})
			value := reflect.New(typ).Elem()
			value.Field(0).Set(reflect.ValueOf(tc.value))
			wire, err := json.Marshal(value.Interface())
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(wire))
		})
	}
}
