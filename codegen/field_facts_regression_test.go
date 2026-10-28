package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestRequiredFieldQueriesDoNotCopyDescendants(t *testing.T) {
	allocations := make([]float64, 0, 2)
	for _, width := range []int{16, 512} {
		attribute := fieldFactsValidationAttribute(width)
		context := NewAttributeContext(true, false, true, "", NewNameScope())
		context.JSONPresence = true
		render := func() string {
			data := validationRenderData{Attribute: attribute, AttributeCtx: context, Target: "body", Context: "body"}
			var output []string
			appendRequiredValidations(&output, attribute, attribute.Validation, context, data)
			return strings.Join(output, "\n")
		}
		require.Equal(t, 8, strings.Count(render(), ".Present()"))
		allocated := testing.AllocsPerRun(3, func() { render() })
		allocations = append(allocations, allocated)
		t.Logf("unrelated descendant width=%d allocations=%.0f", width, allocated)
	}
	require.Less(t, allocations[1], 2*allocations[0], "field requirement queries must not copy unrelated descendants")
}

func TestFieldPresenceFactSources(t *testing.T) {
	context := &AttributeContext{JSONPresence: true}
	attribute := &expr.AttributeExpr{Type: expr.String}
	required := &expr.AttributeExpr{Type: &expr.Object{}, Validation: &expr.ValidationExpr{Required: []string{"missing"}}}
	for _, test := range []struct {
		name     string
		provider expr.FieldFactsProvider
		want     PresenceKind
	}{
		{"nil interface", nil, NativePresence},
		{"nil snapshot", (*expr.ObjectFields)(nil), NativePresence},
		{"nil mutable mapping", (*expr.MappedAttributeExpr)(nil), NativePresence},
		{"empty snapshot", expr.NewObjectFields(nil), OptionalPresence},
		{"empty mapping", expr.NewMappedAttributeExpr(nil), OptionalPresence},
		{"dangling required snapshot", expr.NewObjectFields(required), NativePresence},
		{"dangling required mapping", expr.NewMappedAttributeExpr(required), NativePresence},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, context.FieldPresence(test.provider, "missing", attribute))
			require.Equal(t, NullablePresence, context.FieldPresence(test.provider, "missing", &expr.AttributeExpr{Type: expr.String, Nullable: true}))
			require.Equal(t, NativePresence, context.FieldPresence(test.provider, "missing", &expr.AttributeExpr{Type: expr.Any}))
		})
	}
}

func TestDefaultRenderingPreservesDuplicateFieldKindSelection(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "n:x", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "n:y", Attribute: &expr.AttributeExpr{Type: &expr.Object{
			{Name: "s", Attribute: &expr.AttributeExpr{Type: expr.String}},
		}}},
	}}
	context := NewAttributeContext(false, false, true, "", NewNameScope())
	value, err := context.DefaultValue(attribute, map[string]any{
		"n:x": "hi", "n:y": map[string]any{"s": "bye"},
	})
	require.NoError(t, err)
	require.Contains(t, value.Expression, `N: "hi"`)
}

func fieldFactsValidationAttribute(width int) *expr.AttributeExpr {
	object := &expr.Object{}
	required := make([]string, 0, 8)
	for index := range 8 {
		name := fmt.Sprintf("field%d", index)
		object.Set(name, &expr.AttributeExpr{Type: expr.String, DefaultValue: "default"})
		required = append(required, name)
	}
	nested := &expr.Object{}
	for index := range width {
		nested.Set(fmt.Sprintf("nested%d", index), &expr.AttributeExpr{Type: expr.String})
	}
	object.Set("unrelated", &expr.AttributeExpr{Type: nested})
	return &expr.AttributeExpr{Type: object, Validation: &expr.ValidationExpr{Required: required}}
}

func TestReferenceRequirementPreservesDeclaredJSONStorage(t *testing.T) {
	field := &expr.AttributeExpr{Type: expr.String}
	declaration := &expr.UserTypeExpr{TypeName: "Value", UID: "Value", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "name", Attribute: field}},
	}}
	occurrence := &expr.AttributeExpr{Type: declaration, Validation: &expr.ValidationExpr{Required: []string{"name"}}}
	for _, pointer := range []bool{false, true} {
		context := NewAttributeContext(pointer, false, false, "", NewNameScope())
		context.JSONPresence = true
		facts := expr.NewObjectFields(occurrence)
		require.True(t, facts.FieldFacts("name").Required)
		require.Equal(t, OptionalPresence, context.FieldPresence(facts, "name", field))
		code := ValidationCode(occurrence, nil, context, true, false, false, "body")
		require.Contains(t, code, "!body.Name.Present()")
		require.NotContains(t, code, "body.Name == nil")
	}
	require.False(t, declaration.IsRequired("name"))
}
