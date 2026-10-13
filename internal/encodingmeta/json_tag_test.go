package encodingmeta

import (
	"encoding/json/v2"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONTagOmissionNormalization(t *testing.T) {
	for _, tc := range []struct {
		tag      string
		presence bool
		want     string
	}{
		{"-", false, "-"},
		{"-", true, "-"},
		{"-,", false, "-,,omitempty"},
		{"-,omitempty", true, "-,omitzero"},
		{"secret", false, "secret,omitempty"},
		{"secret,omitzero", false, "secret,omitempty"},
		{"secret,omitempty,string", true, "secret,omitzero,string"},
		{"secret,string", true, "secret,string,omitzero"},
		{"secret,omitempty,omitzero", true, "secret,omitzero"},
		{"-name", true, "-name,omitzero"},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			field := ResolveJSONField(map[string][]string{"struct:tag:json": {tc.tag}}, "value", JSONFieldPolicy{Optional: true, Storage: JSONCustom, Presence: tc.presence, ForceOmission: true})
			tag := field.Tag
			require.True(t, field.Present)
			require.Equal(t, tc.want, tag)
		})
	}
}

func TestJSONTagAuthoredWire(t *testing.T) {
	empty, private := "", "private"
	for _, tc := range []struct {
		name  string
		tag   string
		value *string
		wire  string
	}{
		{"full absent", "wire", nil, `{"wire":null}`},
		{"full empty", "wire", &empty, `{"wire":""}`},
		{"zero absent", "wire,omitzero", nil, `{}`},
		{"zero empty", "wire,omitzero", &empty, `{"wire":""}`},
		{"empty omission", "wire,omitempty", &empty, `{}`},
		{"exact skip", "-", &private, `{}`},
		{"literal dash", "-,omitempty", &private, `{"-":"private"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := ResolveJSONField(map[string][]string{"struct:tag:json": {tc.tag}}, "value", JSONFieldPolicy{Optional: true, Storage: JSONNilable})
			tag := field.Tag
			require.True(t, field.Present)
			typ := reflect.StructOf([]reflect.StructField{{Name: "Value", Type: reflect.TypeFor[*string](), Tag: reflect.StructTag(`json:"` + tag + `"`)}})
			value := reflect.New(typ).Elem()
			value.Field(0).Set(reflect.ValueOf(tc.value))
			wire, err := json.Marshal(value.Interface())
			require.NoError(t, err)
			require.JSONEq(t, tc.wire, string(wire))
		})
	}
}
