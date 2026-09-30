package transportir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestNormalizeCollectionElementNullability preserves nullability inherited
// from named scalar elements when normalization removes their names.
func TestNormalizeCollectionElementNullability(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		for _, container := range []string{"array", "map"} {
			name := container + "/non-null"
			if nullable {
				name = container + "/nullable"
			}
			t.Run(name, func(t *testing.T) {
				alias := &expr.UserTypeExpr{TypeName: "Element", AttributeExpr: &expr.AttributeExpr{
					Type: expr.String, Nullable: nullable,
				}}
				element := &expr.AttributeExpr{Type: alias}
				attribute := &expr.AttributeExpr{Type: &expr.Array{ElemType: element}}
				if container == "map" {
					attribute.Type = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: element}
				}
				normalized := normalizeHTTPAttribute(attribute)
				var actual *expr.AttributeExpr
				if container == "array" {
					actual = expr.AsArray(normalized.Type).ElemType
				} else {
					actual = expr.AsMap(normalized.Type).ElemType
				}
				require.Equal(t, expr.String, actual.Type)
				require.Equal(t, nullable, expr.IsNullable(actual))
				require.Same(t, alias, element.Type)
				require.False(t, element.Nullable)
			})
		}
	}
}

func TestNormalizeHTTPAttributeUsesCompleteEffectiveValidation(t *testing.T) {
	base := &expr.UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &expr.AttributeExpr{
		Type: expr.String,
		Validation: &expr.ValidationExpr{
			Values:    []any{"192.0.2.1", "192.0.2.11"},
			Pattern:   "^192",
			Format:    expr.FormatIP,
			MinLength: new(2),
		},
	}}
	middle := &expr.UserTypeExpr{TypeName: "Middle", UID: "Middle", AttributeExpr: &expr.AttributeExpr{
		Type: base,
		Validation: &expr.ValidationExpr{
			Values:    []any{"192.0.2.1"},
			Pattern:   "1$",
			Format:    expr.FormatIPv4,
			MaxLength: new(12),
		},
	}}
	derived := &expr.UserTypeExpr{TypeName: "Derived", UID: "Derived", AttributeExpr: &expr.AttributeExpr{Type: middle}}

	normalized := normalizeHTTPAttribute(&expr.AttributeExpr{Type: derived})
	require.Equal(t, expr.String, normalized.Type)
	require.Nil(t, normalized.Validation.Values)
	require.Equal(t, [][]any{{"192.0.2.1"}}, normalized.Validation.Enums())
	require.Equal(t, []string{"1$", "^192"}, normalized.Validation.Patterns())
	require.Equal(t, []expr.ValidationFormat{expr.FormatIPv4, expr.FormatIP}, normalized.Validation.Formats())
	require.Equal(t, 2, *normalized.Validation.MinLength)
	require.Equal(t, 12, *normalized.Validation.MaxLength)

	minimum, maximum := 1.0, 10.0
	numericBase := &expr.UserTypeExpr{TypeName: "NumericBase", UID: "NumericBase", AttributeExpr: &expr.AttributeExpr{
		Type: expr.Int, Validation: &expr.ValidationExpr{Minimum: &minimum, Maximum: &maximum},
	}}
	numericDerived := &expr.UserTypeExpr{TypeName: "NumericDerived", UID: "NumericDerived", AttributeExpr: &expr.AttributeExpr{
		Type: numericBase, Validation: &expr.ValidationExpr{ExclusiveMinimum: &minimum, ExclusiveMaximum: &maximum},
	}}
	numeric := normalizeHTTPAttribute(&expr.AttributeExpr{Type: numericDerived})
	require.Nil(t, numeric.Validation.Minimum)
	require.Nil(t, numeric.Validation.Maximum)
	require.Equal(t, minimum, *numeric.Validation.ExclusiveMinimum)
	require.Equal(t, maximum, *numeric.Validation.ExclusiveMaximum)

	roundTrip, err := expr.EffectiveConstraintsFor(normalized)
	require.NoError(t, err)
	require.Equal(t, normalized.Validation.Patterns(), roundTrip.Validation().Lowered().Patterns())
	require.Equal(t, normalized.Validation.Formats(), roundTrip.Validation().Lowered().Formats())
}
