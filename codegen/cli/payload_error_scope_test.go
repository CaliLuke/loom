package cli

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFieldLoadValidationErrorScope(t *testing.T) {
	for _, tc := range []struct {
		typeName string
		parseErr bool
		localErr bool
	}{
		{"string", false, false},
		{"[]byte", false, false},
		{"bool", true, false},
		{"int", true, false},
		{"int64", true, true},
		{"uint64", true, true},
		{"float64", true, true},
	} {
		for _, required := range []bool{false, true} {
			for _, validated := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/required=%t/validated=%t", tc.typeName, required, validated), func(t *testing.T) {
					validation := ""
					if validated {
						validation = "err = validate(value)"
					}
					_, declErr := FieldLoadCode(&FlagData{FullName: "flag", Required: required}, "value", tc.typeName, validation, nil, "*Payload")
					want := tc.parseErr || validated
					if !required && tc.localErr {
						want = false // Conversion declares err inside its block.
					}
					require.Equal(t, want, declErr)
				})
			}
		}
	}
}

func TestFieldLoadErrorUsesDeclaredResultCarrier(t *testing.T) {
	for _, ref := range []string{"loom.Nullable[Payload]", "*Payload", "[]string", "map[string]int", "Scalar"} {
		t.Run(ref, func(t *testing.T) {
			stmt, _ := FieldLoadCode(&FlagData{FullName: "flag", Required: true, Type: "JSON"}, "value", "Payload", "err = validate(value)", nil, ref)
			rendered := renderStatement(t, stmt)
			require.Contains(t, rendered, "var zero "+ref)
			require.NotContains(t, rendered, "return nil,")
		})
	}
}
