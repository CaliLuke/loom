package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestEffectiveNumericAliasConjunction(t *testing.T) {
	tests := []struct {
		name                                                 string
		minimum, exclusiveMinimum, maximum, exclusiveMaximum float64
	}{
		{name: "lower", minimum: 5, exclusiveMinimum: 1, maximum: 8, exclusiveMaximum: 9},
		{name: "upper", minimum: -2, exclusiveMinimum: -3, maximum: 1, exclusiveMaximum: 5},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var authored expr.UserType
			RunDSL(t, func() {
				base := dsl.Type("NumericBase", dsl.Int)
				authored = dsl.Type("NumericBounds", base, func() {
					dsl.Minimum(test.minimum)
					dsl.ExclusiveMinimum(test.exclusiveMinimum)
					dsl.Maximum(test.maximum)
					dsl.ExclusiveMaximum(test.exclusiveMaximum)
				})
				dsl.Service("numeric", func() {
					dsl.Method("send", func() { dsl.Payload(authored) })
				})
			})
			attribute := &expr.AttributeExpr{Type: authored}
			context := expr.NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			result := context.Resolve(
				occurrence,
				context.SupplyValue(expr.ValueInput{Raw: 3}),
				expr.ValueRoleDefault,
			)
			require.Equal(t, expr.ValueInvalid, result.Outcome())

			validation := mergedValidation(attribute)
			require.NotNil(t, validation)
			code := ValidationCode(attribute, nil, NewAttributeContext(false, false, true, "", NewNameScope()), true, false, false, "value")
			require.NotEmpty(t, code)
			if test.name == "lower" {
				require.Equal(t, test.minimum, *validation.Minimum)
				require.Nil(t, validation.ExclusiveMinimum)
				require.Equal(t, test.maximum, *validation.Maximum)
				require.Nil(t, validation.ExclusiveMaximum)
				require.Contains(t, code, "if value < 5")
				require.Contains(t, code, "if value > 8")
			} else {
				require.Equal(t, test.minimum, *validation.Minimum)
				require.Equal(t, test.maximum, *validation.Maximum)
				require.Contains(t, code, "if value < -2")
				require.Contains(t, code, "if value > 1")
			}
		})
	}
}

func TestEffectiveStringClausesRenderOnceInDerivedToBaseOrder(t *testing.T) {
	base := &expr.UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &expr.AttributeExpr{
		Type: expr.String,
		Validation: &expr.ValidationExpr{
			Pattern: "^a",
			Format:  expr.FormatIP,
		},
	}}
	derived := &expr.UserTypeExpr{TypeName: "Derived", UID: "Derived", AttributeExpr: &expr.AttributeExpr{
		Type: base,
		Validation: &expr.ValidationExpr{
			Pattern:        "b$",
			PatternClauses: []string{"^a"},
			Format:         expr.FormatIPv4,
			FormatClauses:  []expr.ValidationFormat{expr.FormatIP},
		},
	}}
	attribute := &expr.AttributeExpr{Type: derived}
	code := ValidationCode(attribute, nil, NewAttributeContext(false, false, true, "", NewNameScope()), true, false, false, "value")

	require.Equal(t, 1, strings.Count(code, `loom.ValidatePattern("value", value, "b$")`))
	require.Equal(t, 1, strings.Count(code, `loom.ValidatePattern("value", value, "^a")`))
	require.Equal(t, 1, strings.Count(code, `loom.ValidateFormat("value", value, loom.FormatIPv4)`))
	require.Equal(t, 1, strings.Count(code, `loom.ValidateFormat("value", value, loom.FormatIP)`))
	require.Less(t, strings.Index(code, `"b$"`), strings.Index(code, `"^a"`))
}

func TestInheritedEnumAndDerivedPatternRenderAsConjunction(t *testing.T) {
	base := &expr.UserTypeExpr{TypeName: "Base", UID: "Base", AttributeExpr: &expr.AttributeExpr{
		Type:       expr.String,
		Validation: &expr.ValidationExpr{Values: []any{"ab", "ax"}},
	}}
	derived := &expr.UserTypeExpr{TypeName: "Derived", UID: "Derived", AttributeExpr: &expr.AttributeExpr{
		Type:       base,
		Validation: &expr.ValidationExpr{Pattern: "b$"},
	}}
	attribute := &expr.AttributeExpr{Type: derived}
	code := ValidationCode(attribute, nil, NewAttributeContext(false, false, true, "", NewNameScope()), true, false, false, "value")
	require.Contains(t, code, `value == "ab"`)
	require.Contains(t, code, `value == "ax"`)
	require.Contains(t, code, `loom.ValidatePattern("value", value, "b$")`)

	narrowed := &expr.UserTypeExpr{TypeName: "Narrowed", UID: "Narrowed", AttributeExpr: &expr.AttributeExpr{
		Type:       base,
		Validation: &expr.ValidationExpr{Pattern: "b$", Values: []any{"ab"}},
	}}
	narrowedCode := ValidationCode(&expr.AttributeExpr{Type: narrowed}, nil, NewAttributeContext(false, false, true, "", NewNameScope()), true, false, false, "value")
	require.Contains(t, narrowedCode, `value == "ab"`)
	require.NotContains(t, narrowedCode, `value == "ax"`)
	require.Contains(t, narrowedCode, `loom.ValidatePattern("value", value, "b$")`)

	invalid := &expr.UserTypeExpr{TypeName: "Invalid", UID: "Invalid", AttributeExpr: &expr.AttributeExpr{
		Type:       base,
		Validation: &expr.ValidationExpr{Pattern: "b$", Values: []any{"ab", "ax"}},
	}}
	_, err := expr.EffectiveConstraintsFor(&expr.AttributeExpr{Type: invalid})
	require.ErrorContains(t, err, `enum member "ax" declared by "Invalid"`)
}

func TestEnumClausesRenderEveryMembershipPredicate(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.Int, Validation: &expr.ValidationExpr{
		EnumClauses: [][]any{{1, 2}, {2, 3}},
	}}
	code := ValidationCode(attribute, nil, NewAttributeContext(false, false, true, "", NewNameScope()), true, false, false, "value")
	require.Equal(t, 2, strings.Count(code, "InvalidEnumValueError"))
	require.Contains(t, code, "value == 1 || value == 2")
	require.Contains(t, code, "value == 2 || value == 3")

	empty := &expr.AttributeExpr{Type: expr.Int, Validation: &expr.ValidationExpr{EnumClauses: [][]any{{}}}}
	emptyCode := ValidationCode(empty, nil, NewAttributeContext(false, false, true, "", NewNameScope()), true, false, false, "value")
	require.Contains(t, emptyCode, "if !(false)")

	authoredEmpty := &expr.AttributeExpr{Type: expr.Int, Validation: &expr.ValidationExpr{Values: []any{}}}
	constraints, err := expr.EffectiveConstraintsFor(authoredEmpty)
	require.NoError(t, err)
	lowered := constraints.Validation().Lowered()
	require.Nil(t, lowered.Values)
	require.Equal(t, [][]any{{}}, lowered.EnumClauses)
	authoredEmptyCode := ValidationCode(
		authoredEmpty,
		nil,
		NewAttributeContext(false, false, true, "", NewNameScope()),
		true,
		false,
		false,
		"value",
	)
	require.Contains(t, authoredEmptyCode, "if !(false)")
}

func TestEffectiveNumericAliasContradictionRemainsAdmitted(t *testing.T) {
	var derived expr.UserType
	RunDSL(t, func() {
		base := dsl.Type("Minimum", dsl.Int, func() { dsl.Minimum(5) })
		derived = dsl.Type("EmptyRange", base, func() { dsl.Maximum(3) })
		dsl.Service("numeric", func() {
			dsl.Method("send", func() { dsl.Payload(derived) })
		})
	})
	attribute := &expr.AttributeExpr{Type: derived}
	constraints, err := expr.EffectiveConstraintsFor(attribute)
	require.NoError(t, err)
	validation := constraints.Validation().Lowered()
	require.Equal(t, 5.0, *validation.Minimum)
	require.Equal(t, 3.0, *validation.Maximum)
	for _, value := range []int{2, 4, 6} {
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		result := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: value}), expr.ValueRoleDefault)
		require.Equal(t, expr.ValueInvalid, result.Outcome())
	}
}
