package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAdmittedSourceDefaultNoOpProof(t *testing.T) {
	for _, tc := range []struct {
		name                                             string
		typ                                              expr.DataType
		value                                            any
		nullable, pointer, deferred, custom, materialize bool
	}{
		{name: "boolean", typ: expr.Boolean, value: true},
		{name: "string", typ: expr.String, value: "same"},
		{name: "integer", typ: expr.Int, value: 7},
		{name: "integer alias", typ: &expr.UserTypeExpr{TypeName: "Count", AttributeExpr: &expr.AttributeExpr{Type: expr.Int}}, value: 7},
		{name: "float same value is not proof", typ: expr.Float64, value: 7.0, materialize: true},
		{name: "pointer target", typ: expr.Int, value: 7, pointer: true, materialize: true},
		{name: "deferred target", typ: expr.Int, value: 7, deferred: true, materialize: true},
		{name: "nullable", typ: expr.Int, value: 7, nullable: true, materialize: true},
		{name: "custom", typ: expr.String, value: "same", custom: true, materialize: true},
		{name: "equal raw collection", typ: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}, value: []int{7}, materialize: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: tc.typ, DefaultValue: tc.value, Nullable: tc.nullable}
			if tc.custom {
				field.Meta = expr.MetaExpr{"struct:field:type": {"Token"}}
			}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
			mapped := expr.NewMappedAttributeExpr(parent)
			scope := NewNameScope()
			ta := &TransformAttrs{SourceCtx: NewAttributeContext(true, false, false, "", scope), TargetCtx: NewAttributeContext(tc.pointer, false, !tc.deferred, "", scope), Options: TransformOptions{AdmittedSource: "request"}}
			application := ta.DefaultForField(mapped, mapped, "value")
			require.True(t, application.Apply)
			require.Equal(t, tc.materialize, application.admittedSource)
			if !tc.materialize {
				ordinary, err := GoTransform(parent, parent, "source", "target", ta.SourceCtx, ta.TargetCtx, "convert", true)
				require.NoError(t, err)
				admitted, err := GoTransform(parent, parent, "source", "target", ta.SourceCtx, ta.TargetCtx, "convert", true, ta.Options)
				require.NoError(t, err)
				require.Equal(t, ordinary, admitted, "proved scalar operations retain exact code/helper bytes")
			}
		})
	}
}

func TestAdmittedSourceDefaultHelperClosure(t *testing.T) {
	for _, changed := range []bool{false, true} {
		source, target := admittedDefaultHelperFixture(changed)
		scope := NewNameScope()
		srcCtx, tgtCtx := NewAttributeContext(true, false, false, "", scope), NewAttributeContext(false, false, true, "", scope)
		ordinary, err := GoTransform(source, target, "source", "target", srcCtx, tgtCtx, "convert", true)
		require.NoError(t, err)
		admitted, err := GoTransform(source, target, "source", "target", srcCtx, tgtCtx, "convert", true, TransformOptions{AdmittedSource: "request/first"})
		require.NoError(t, err)
		require.Len(t, admitted.Helpers, 2)
		if !changed {
			require.Equal(t, ordinary, admitted)
			continue
		}
		require.Contains(t, admitted.Helpers[0].Name, "Admitted")
		require.Contains(t, admitted.Helpers[0].Code, admitted.Helpers[0].Name+"(", "recursive call uses selected helper identity")
		require.Equal(t, ordinary.Helpers[1], admitted.Helpers[1], "unaffected sibling preserves its helper")
		second, err := GoTransform(source, target, "source", "target", srcCtx, tgtCtx, "convert", true, TransformOptions{AdmittedSource: "request/second"})
		require.NoError(t, err)
		require.NotEqual(t, admitted.Helpers[0].Name, second.Helpers[0].Name, "distinct source selections cannot alias helpers")
		custom, err := GoTransform(source, target, "source", "target", srcCtx, tgtCtx, "convert", true, TransformOptions{AdmittedSource: "request/custom", HelperName: func(source, target *expr.AttributeExpr) string {
			return "owned" + source.Type.Name() + target.Type.Name()
		}})
		require.NoError(t, err)
		require.Equal(t, "ownedSourceNodeTargetNode", custom.Helpers[0].Name, "custom semantic naming remains caller-owned")
	}
}

func TestAdmittedSourceDefaultFailureClosure(t *testing.T) {
	source, target := customDefaultCycle(true, false)
	scope := NewNameScope()
	srcCtx := NewAttributeContext(true, false, false, "source", scope)
	tgtCtx := NewAttributeContext(true, false, false, "target", scope)
	plan, err := GoTransform(source, target, "source", "target", srcCtx, tgtCtx, "convert", true, TransformOptions{AdmittedSource: "request/custom"})
	require.NoError(t, err)
	require.True(t, plan.ErrorAware)
	for _, helper := range plan.Helpers {
		require.True(t, helper.ErrorAware, helper.Name)
		require.Contains(t, helper.Name, "Admitted")
		require.Contains(t, helper.Code, "transformErr")
	}
}

func TestAdmittedSourceDefaultInvalidResolutionIsError(t *testing.T) {
	maximum, err := expr.NewNumericBound(5)
	require.NoError(t, err)
	field := &expr.AttributeExpr{Type: expr.Int, DefaultValue: 9, Validation: &expr.ValidationExpr{Maximum: &maximum}}
	parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
	scope := NewNameScope()
	_, err = GoTransform(parent, parent, "source", "target", NewAttributeContext(true, false, false, "", scope), NewAttributeContext(false, false, true, "", scope), "convert", true, TransformOptions{AdmittedSource: "request"})
	var materialization *DefaultValueError
	require.ErrorAs(t, err, &materialization)
}

func admittedDefaultHelperFixture(changed bool) (*expr.AttributeExpr, *expr.AttributeExpr) {
	build := func(prefix string, value int) *expr.AttributeExpr {
		fields := expr.Object{}
		node := &expr.UserTypeExpr{TypeName: prefix + "Node", UID: prefix + "Node", AttributeExpr: &expr.AttributeExpr{Type: &fields}}
		fields.Set("count", &expr.AttributeExpr{Type: expr.Int, DefaultValue: value})
		fields.Set("next", &expr.AttributeExpr{Type: node})
		other := &expr.UserTypeExpr{TypeName: prefix + "Other", UID: prefix + "Other", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "label", Attribute: &expr.AttributeExpr{Type: expr.String}}}}}
		return &expr.AttributeExpr{Type: &expr.Object{{Name: "node", Attribute: &expr.AttributeExpr{Type: node}}, {Name: "other", Attribute: &expr.AttributeExpr{Type: other}}}}
	}
	value := 7
	if changed {
		value = 9
	}
	return build("Source", value), build("Target", 7)
}

func TestAdmittedSourceDefaultOccurrenceIdentity(t *testing.T) {
	first, target := admittedDefaultHelperFixture(true)
	second, _ := admittedDefaultHelperFixture(true)
	expr.AsObject(second.Find("node").Type).Attribute("count").DefaultValue = 11
	source := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "first:wire_first", Attribute: first.Find("node")},
		{Name: "second:wire_second", Attribute: second.Find("node")},
	}}
	target = &expr.AttributeExpr{Type: &expr.Object{
		{Name: "first:service_first", Attribute: target.Find("node")},
		{Name: "second:service_second", Attribute: target.Find("node")},
	}}
	beforeSource, beforeTarget := expr.DupAtt(source), expr.DupAtt(target)
	scope := NewNameScope()
	plan, err := GoTransform(source, target, "source", "target", NewAttributeContext(true, false, false, "", scope), NewAttributeContext(false, false, true, "", scope), "convert", true, TransformOptions{AdmittedSource: "request"})
	require.NoError(t, err)
	require.Len(t, plan.Helpers, 2)
	require.NotEqual(t, plan.Helpers[0].Name, plan.Helpers[1].Name)
	require.Equal(t, plan.Helpers[0].ParamTypeRef, plan.Helpers[1].ParamTypeRef)
	require.Contains(t, plan.Helpers[0].Code, "= 9")
	require.Contains(t, plan.Helpers[1].Code, "= 11")
	for _, helper := range plan.Helpers {
		require.Contains(t, helper.Code, helper.Name+"(")
	}
	require.Equal(t, beforeSource, expr.DupAtt(source), "mapped traversal must not mutate selected source")
	require.Equal(t, beforeTarget, expr.DupAtt(target), "mapped traversal must not mutate target")
}

func TestAdmittedSourceDefaultInheritedNullableHelper(t *testing.T) {
	source, target := admittedDefaultHelperFixture(true)
	source, target = source.Find("node"), target.Find("node")
	source.Type.(expr.UserType).Attribute().Nullable = true
	target.Type.(expr.UserType).Attribute().Nullable = true
	scope := NewNameScope()
	plan, err := GoTransform(source, target, "source", "target", NewAttributeContext(true, false, false, "", scope), NewAttributeContext(false, false, true, "", scope), "convert", true, TransformOptions{AdmittedSource: "request"})
	require.NoError(t, err)
	require.Len(t, plan.Helpers, 1)
	require.Contains(t, plan.Helpers[0].Name, "Admitted")
	require.Contains(t, plan.Code, plan.Helpers[0].Name+"(")
	require.Contains(t, plan.Helpers[0].Code, plan.Helpers[0].Name+"(")
}
