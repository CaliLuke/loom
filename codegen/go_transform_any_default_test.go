package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestTransformAnyDefaultContext exhausts the context switches that govern
// default assignment. Raw JSON uses nil for absence and nonnil bytes for null.
func TestTransformAnyDefaultContext(t *testing.T) {
	for mask := range 128 {
		named := mask&1 != 0
		sourceDefaults := mask&2 != 0
		targetDefaults := mask&4 != 0
		pointer := mask&8 != 0
		required := mask&16 != 0
		defaulted := mask&32 != 0
		jsonPresence := mask&64 != 0
		name := fmt.Sprintf("named_%t/source_defaults_%t/target_defaults_%t/pointer_%t/required_%t/default_%t/json_%t",
			named, sourceDefaults, targetDefaults, pointer, required, defaulted, jsonPresence)
		t.Run(name, func(t *testing.T) {
			var datatype expr.DataType = expr.Any
			if named {
				datatype = &expr.UserTypeExpr{TypeName: "Metadata", AttributeExpr: &expr.AttributeExpr{Type: expr.Any}}
			}
			field := &expr.AttributeExpr{Type: datatype}
			if defaulted {
				field.DefaultValue = "fallback"
			}
			attribute := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
			if required {
				attribute.Validation = &expr.ValidationExpr{Required: []string{"value"}}
			}
			scope := NewNameScope()
			source := NewAttributeContext(true, false, sourceDefaults, "", scope)
			source.JSONPresence = jsonPresence
			target := NewAttributeContext(pointer, false, targetDefaults, "", scope)
			code, _, err := GoTransform(attribute, attribute, "body", "payload", source, target, "", true)
			require.NoError(t, err)
			if defaulted && targetDefaults && !pointer && !required {
				require.Contains(t, code, `loom.JSONValue("\"fallback\"")`)
				require.Contains(t, code, "== nil")
				return
			}
			require.NotContains(t, code, "fallback")
		})
	}
}

func TestTransformAnyDefaultPreservesCustomRepresentation(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{{
		Name: "value",
		Attribute: &expr.AttributeExpr{
			Type:         expr.Any,
			DefaultValue: "fallback",
			Meta:         expr.MetaExpr{"struct:field:type": {"string"}},
		},
	}}}
	scope := NewNameScope()
	source := NewAttributeContext(false, false, false, "", scope)
	target := NewAttributeContext(false, false, true, "", scope)
	code, _, err := GoTransform(attribute, attribute, "body", "payload", source, target, "", true)
	require.NoError(t, err)
	require.NotContains(t, code, "== nil")
	require.NotContains(t, code, "loom.JSONValue")
}
