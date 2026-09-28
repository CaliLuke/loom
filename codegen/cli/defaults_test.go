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

func TestFlagCollectionDefaultsPreserveExistingRepresentation(t *testing.T) {
	for _, tc := range []struct {
		name, typeName string
		value          any
		expected       string
	}{
		{"array", "[]string", []string{"a", "b"}, "[a b]"},
		{"map", "map[string]int", map[string]int{"b": 2, "a": 1}, "map[a:1 b:2]"},
		{"boolean-map", "map[bool]string", map[bool]string{true: "on", false: "off"}, "map[false:off true:on]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flag := NewFlagData("service", "method", "value", tc.typeName, "", false, nil, tc.value)
			require.Equal(t, tc.expected, flagDefaultValue(flag))
		})
	}
}
