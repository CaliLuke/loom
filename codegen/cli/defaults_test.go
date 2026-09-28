package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlagDefaultsUseParserRepresentation(t *testing.T) {
	for _, tc := range []struct {
		name, typeName string
		value          any
		expected       string
	}{
		{"string", "string", "text", "text"},
		{"false", "bool", false, "false"},
		{"zero", "int", 0, "0"},
		{"bytes", "[]byte", []byte{0, 255}, "\x00\xff"},
		{"named-string", "TextBody", "text", `"text"`},
		{"named-bytes", "BlobBody", []byte{0, 255}, `"AP8="`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flag := NewFlagData("service", "method", "body", tc.typeName, "", false, tc.value, tc.value)
			require.Equal(t, tc.expected, flagDefaultValue(flag))
		})
	}
}

func TestFlagCollectionDefaultsUseJSON(t *testing.T) {
	for _, tc := range []struct {
		name, typeName string
		value          any
		expected       string
	}{
		{"array", "[]string", []string{"a", "b"}, `["a","b"]`},
		{"empty-array", "[]string", []string{}, `[]`},
		{"map", "map[string]int", map[string]int{"b": 2, "a": 1}, `{"a":1,"b":2}`},
		{"empty-map", "map[string]int", map[string]int{}, `{}`},
		{"boolean-map", "map[bool]string", map[bool]string{true: "on", false: "off"}, `{"false":"off","true":"on"}`},
		{"integer-map", "map[int]string", map[int]string{7: "seven"}, `{"7":"seven"}`},
		{"float-map", "map[float32]string", map[float32]string{1.2: "decimal"}, `{"1.2":"decimal"}`},
		{"nested", "[]map[bool][]bool", []map[bool][]bool{{true: {false, true}}}, `[{"true":[false,true]}]`},
		{"interface-keys", "MapBody", map[any]any{true: []any{false, 0}}, `{"true":[false,0]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flag := NewFlagData("service", "method", "value", tc.typeName, "", false, nil, tc.value)
			require.Equal(t, tc.expected, flagDefaultValue(flag))
		})
	}
}
