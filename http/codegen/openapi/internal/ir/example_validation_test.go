package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestOpenAPIFieldExampleMatchesStructuredContracts(t *testing.T) {
	t.Parallel()

	minLength, maxLength := 1, 2
	text := &expr.AttributeExpr{Type: expr.String}
	object := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "name", Attribute: text}},
		Validation: &expr.ValidationExpr{Required: []string{"name"}},
	}
	array := &expr.AttributeExpr{
		Type:       &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int64}},
		Validation: &expr.ValidationExpr{MinLength: &minLength, MaxLength: &maxLength},
	}
	mapping := &expr.AttributeExpr{
		Type:       &expr.Map{ElemType: &expr.AttributeExpr{Type: expr.Boolean}},
		Validation: &expr.ValidationExpr{MinLength: &minLength, MaxLength: &maxLength},
	}
	tagged := &expr.AttributeExpr{Type: &expr.Union{
		TypeKey:  "kind",
		ValueKey: "data",
		Values: []*expr.NamedAttributeExpr{{
			Name:      "Text",
			Attribute: text,
		}},
	}}
	kind := func(value string) *expr.AttributeExpr {
		return &expr.AttributeExpr{
			Type: &expr.Object{{
				Name: "kind",
				Attribute: &expr.AttributeExpr{
					Type:       expr.String,
					Validation: &expr.ValidationExpr{Values: []any{value}},
				},
			}},
			Validation: &expr.ValidationExpr{Required: []string{"kind"}},
		}
	}
	untagged := &expr.AttributeExpr{Type: &expr.Union{
		Untagged: true,
		Values: []*expr.NamedAttributeExpr{
			{Name: "A", Attribute: kind("a")},
			{Name: "B", Attribute: kind("b")},
		},
	}}
	userType := &expr.UserTypeExpr{TypeName: "Text", AttributeExpr: text}

	tests := []struct {
		name  string
		attr  *expr.AttributeExpr
		value any
		want  bool
	}{
		{name: "nil attribute", value: "value", want: false},
		{name: "nullable null", attr: &expr.AttributeExpr{Type: expr.String, Nullable: true}, value: nil, want: true},
		{name: "nonnullable null", attr: text, value: nil, want: false},
		{name: "user type", attr: &expr.AttributeExpr{Type: userType}, value: "value", want: true},
		{name: "object", attr: object, value: map[string]any{"name": "loom"}, want: true},
		{name: "object missing required", attr: object, value: map[string]any{}, want: false},
		{name: "array", attr: array, value: []any{int64(1), int64(2)}, want: true},
		{name: "array too short", attr: array, value: []any{}, want: false},
		{name: "array wrong container", attr: array, value: "value", want: false},
		{name: "array wrong element", attr: array, value: []any{"one"}, want: false},
		{name: "map", attr: mapping, value: map[string]any{"enabled": true}, want: true},
		{name: "map too long", attr: mapping, value: map[string]any{"a": true, "b": true, "c": true}, want: false},
		{name: "map wrong container", attr: mapping, value: []any{true}, want: false},
		{name: "map wrong element", attr: mapping, value: map[string]any{"enabled": "yes"}, want: false},
		{name: "tagged union", attr: tagged, value: map[string]any{"kind": "Text", "data": "loom"}, want: true},
		{name: "tagged union wrong container", attr: tagged, value: "loom", want: false},
		{name: "tagged union unknown branch", attr: tagged, value: map[string]any{"kind": "Missing", "data": "loom"}, want: false},
		{name: "untagged union", attr: untagged, value: map[string]any{"kind": "a"}, want: true},
		{name: "untagged union ambiguous", attr: &expr.AttributeExpr{Type: &expr.Union{Untagged: true, Values: []*expr.NamedAttributeExpr{{Name: "One", Attribute: object}, {Name: "Two", Attribute: object}}}}, value: map[string]any{"name": "loom"}, want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.want, openAPIFieldExampleMatches(testCase.attr, testCase.value))
		})
	}
}

func TestValidationMatchesExactConstraints(t *testing.T) {
	t.Parallel()

	minLength, maxLength := 2, 4
	minimum, exclusiveMinimum := 1.0, 0.0
	maximum, exclusiveMaximum := 5.0, 6.0
	validation := &expr.ValidationExpr{
		Values:           []any{2, 3, 4, 5},
		Minimum:          &minimum,
		ExclusiveMinimum: &exclusiveMinimum,
		Maximum:          &maximum,
		ExclusiveMaximum: &exclusiveMaximum,
	}

	require.True(t, validationMatches(validation, int64(2)))
	require.False(t, validationMatches(validation, int64(1)))
	require.False(t, validationMatches(validation, int64(6)))
	require.True(t, validationMatches(&expr.ValidationExpr{Pattern: "^[a-z]+$", MinLength: &minLength, MaxLength: &maxLength}, "loom"))
	require.False(t, validationMatches(&expr.ValidationExpr{Pattern: "^[a-z]+$"}, "123"))
	require.False(t, validationMatches(&expr.ValidationExpr{MinLength: &minLength}, "x"))
	require.False(t, validationMatches(&expr.ValidationExpr{MaxLength: &maxLength}, "longer"))
}
