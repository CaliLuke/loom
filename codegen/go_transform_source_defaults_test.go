package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAdmittedSourceDefaultSelection(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		source, target              any
		pointer, deferred, required bool
		want                        any
		apply                       bool
	}{
		{name: "body replaces payload", source: 9, target: 7, want: 9, apply: true},
		{name: "source only pointer target", source: 9, pointer: true, want: 9, apply: true},
		{name: "source independent of deferred target", source: 9, target: 7, deferred: true, want: 9, apply: true},
		{name: "ordinary fallback without source default", target: 7, want: 7, apply: true},
		{name: "required input not repaired", source: 9, target: 7, required: true, want: 7},
		{name: "same scalar", source: 7, target: 7, want: 7, apply: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &expr.AttributeExpr{Type: &expr.Object{{Name: "count:wire", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: tc.source}}}}
			target := &expr.AttributeExpr{Type: &expr.Object{{Name: "count:service", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: tc.target}}}}
			if tc.required {
				source.Validation = &expr.ValidationExpr{Required: []string{"count:wire"}}
			}
			scope := NewNameScope()
			ta := &TransformAttrs{
				SourceCtx: NewAttributeContext(true, false, false, "", scope),
				TargetCtx: NewAttributeContext(tc.pointer, false, !tc.deferred, "", scope),
				Options:   TransformOptions{AdmittedSource: "HTTP/test/body"},
			}
			plan := ta.DefaultForField(expr.NewMappedAttributeExpr(source), expr.NewMappedAttributeExpr(target), "count")
			require.Equal(t, tc.apply, plan.Apply)
			require.Equal(t, tc.want, plan.Value)
		})
	}
}

func TestAdmittedSourceDefaultMappedWinner(t *testing.T) {
	inherited := &expr.UserTypeExpr{TypeName: "Base", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 7}}}}}
	local := &expr.AttributeExpr{Type: expr.Int, DefaultValue: 9}
	source := &expr.AttributeExpr{Type: &expr.Object{{Name: "count:wire", Attribute: local}}, References: []expr.DataType{inherited}}
	target := &expr.AttributeExpr{Type: inherited}
	scope := NewNameScope()
	ta := &TransformAttrs{SourceCtx: NewAttributeContext(true, false, false, "", scope), TargetCtx: NewAttributeContext(false, false, true, "", scope), Options: TransformOptions{AdmittedSource: "mapped-body"}}
	walkMatches(source, target, func(src, tgt *expr.MappedAttributeExpr, child, _ *expr.AttributeExpr, name string) {
		require.Same(t, local, child, "local mapped winner precedes the referenced exact-name child")
		require.Equal(t, 9, src.GetDefault(name))
		require.Equal(t, 9, ta.DefaultForField(src, tgt, name).Value)
	})
}
