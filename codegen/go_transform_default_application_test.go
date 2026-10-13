package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestTransformDefaultApplication(t *testing.T) {
	for _, tc := range []struct {
		name     string
		typeName string
		typ      expr.DataType
		value    any
		pointer  bool
		optional bool
		nullable bool
		required bool
		apply    bool
		presence PresenceKind
	}{
		{name: "native scalar", typ: expr.String, value: "fallback"},
		{name: "native alias", typ: &expr.UserTypeExpr{TypeName: "Label", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}, value: "fallback"},
		{name: "pointer", typ: expr.String, value: "fallback", pointer: true, apply: true},
		{name: "bytes", typ: expr.Bytes, value: []byte("fallback"), apply: true},
		{name: "bytes alias", typ: &expr.UserTypeExpr{TypeName: "Blob", AttributeExpr: &expr.AttributeExpr{Type: expr.Bytes}}, value: []byte("fallback"), apply: true},
		{name: "array", typ: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, value: []string{"fallback"}, apply: true},
		{name: "map", typ: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}, value: map[string]string{"key": "fallback"}, apply: true},
		{name: "optional", typ: expr.String, value: "fallback", optional: true, apply: true, presence: OptionalPresence},
		{name: "nullable", typ: expr.String, value: "fallback", nullable: true, apply: true, presence: NullablePresence},
		{name: "required pointer", typ: expr.String, value: "fallback", pointer: true, required: true},
		{name: "required optional", typ: expr.String, value: "fallback", optional: true, required: true, presence: OptionalPresence},
		{name: "required nullable", typ: expr.String, value: "fallback", nullable: true, required: true, presence: NullablePresence},
		{name: "opaque custom", typ: expr.String, value: "fallback", typeName: "CustomString"},
		{name: "custom pointer", typ: expr.String, value: "fallback", typeName: "CustomString", pointer: true, apply: true},
		{name: "custom slice", typ: expr.String, value: "fallback", typeName: "[]byte", apply: true},
		{name: "custom map", typ: expr.String, value: "fallback", typeName: "map[string]string", apply: true},
		{name: "raw JSON", typ: expr.String, value: "fallback", typeName: "jsontext.Value", apply: true},
		{name: "overridden bytes", typ: expr.Bytes, value: []byte("fallback"), typeName: "CustomBytes"},
		{name: "no default", typ: expr.String, pointer: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: tc.typ, DefaultValue: tc.value, Nullable: tc.nullable}
			if tc.typeName != "" {
				attribute.Meta = expr.MetaExpr{"struct:field:type": {tc.typeName}}
			}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: attribute}}}
			if tc.required {
				parent.Validation = &expr.ValidationExpr{Required: []string{"value"}}
			}
			mapped := expr.NewMappedAttributeExpr(parent)
			for mask := range 4 {
				scope := NewNameScope()
				source := NewAttributeContext(tc.pointer, false, true, "", scope)
				source.JSONPresence = tc.optional
				target := NewAttributeContext(mask&1 != 0, false, mask&2 != 0, "", scope)
				plan := (&TransformAttrs{SourceCtx: source, TargetCtx: target}).DefaultForField(mapped, mapped, "value")
				require.Equal(t, tc.value, plan.Value)
				require.Equal(t, tc.presence, plan.SourcePresence)
				require.Equal(t, tc.apply && mask == 2, plan.Apply, "target mask=%d", mask)
			}
		})
	}
}

func TestTransformDefaultUsesTargetOccurrence(t *testing.T) {
	source := expr.NewMappedAttributeExpr(&expr.AttributeExpr{Type: &expr.Object{{Name: "value:source", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 7}}}})
	target := expr.NewMappedAttributeExpr(&expr.AttributeExpr{Type: &expr.Object{{Name: "value:target", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 9}}}})
	scope := NewNameScope()
	transform := &TransformAttrs{
		SourceCtx: NewAttributeContext(true, false, false, "", scope),
		TargetCtx: NewAttributeContext(false, false, true, "", scope),
	}
	plan := transform.DefaultForField(source, target, "value")
	require.True(t, plan.Apply)
	require.Equal(t, 9, plan.Value)
}
