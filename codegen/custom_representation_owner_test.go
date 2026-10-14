package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestCustomStringValidationUsesSemanticRepresentation(t *testing.T) {
	minimum := 1
	for _, tc := range []struct {
		name    string
		pointer bool
	}{{"value", false}, {"pointer", true}} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{
				Type:       expr.String,
				Meta:       expr.MetaExpr{"struct:field:type": {"uuid.UUID", "github.com/google/uuid"}},
				Validation: &expr.ValidationExpr{MinLength: &minimum},
			}
			context := NewAttributeContext(tc.pointer, false, false, "", NewNameScope())
			code := ValidationCode(attribute, nil, context, true, false, true, "value")
			require.NotContains(t, code, "utf8.RuneCountInString(value)", "opaque storage must not be passed to native String operations")
			require.NotContains(t, code, "utf8.RuneCountInString(*value)", "dereferencing does not change the physical Go type")
			require.Contains(t, code, "InvalidLengthError", "authored validation must remain enforced")
		})
	}
}

func TestCustomStringDefaultUsesPhysicalRepresentation(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type: expr.String,
		Meta: expr.MetaExpr{"struct:field:type": {"uuid.UUID", "github.com/google/uuid"}},
	}
	context := NewAttributeContext(false, false, true, "", NewNameScope())
	valuePlan, err := context.DefaultValue(attribute, "550e8400-e29b-41d4-a716-446655440000")
	value := valuePlan.Expression
	require.NoError(t, err)
	require.NotEqual(t, `"550e8400-e29b-41d4-a716-446655440000"`, value, "native String literal is not a UUID representation")
}

func TestNamedCustomStringValidationUsesRunes(t *testing.T) {
	minimum := 2
	attribute := &expr.AttributeExpr{
		Type:       &expr.UserTypeExpr{TypeName: "Label", AttributeExpr: &expr.AttributeExpr{Type: expr.String}},
		Meta:       expr.MetaExpr{"struct:field:type": {"codec.Label", "example.com/codec"}},
		Validation: &expr.ValidationExpr{MinLength: &minimum},
	}
	context := NewAttributeContext(false, false, false, "", NewNameScope())
	code := validationCode(attribute, context, true, true, "value", "result.name")
	require.Contains(t, code, "utf8.RuneCountInString(customTextValue)")
	require.NotContains(t, code, "len(customTextValue)")
}
