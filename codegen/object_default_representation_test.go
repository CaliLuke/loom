package codegen

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type (
	defaultJSONCodec struct {
		calls *int
	}

	defaultJSONErrorCodec struct{}
)

func (v defaultJSONCodec) MarshalJSON() ([]byte, error) {
	*v.calls++
	return []byte(`{"custom":true}`), nil
}

func TestObjectDefaultRejectsUnstoredMembers(t *testing.T) {
	for _, nested := range []bool{false, true} {
		leaf := &expr.AttributeExpr{Type: &expr.Object{{Name: "x", Attribute: &expr.AttributeExpr{Type: expr.Int}}}}
		target := leaf
		raw := map[string]any{"x": 1, "extra": 2}
		path := "default.extra"
		if nested {
			target = &expr.AttributeExpr{Type: &expr.Object{{Name: "child", Attribute: leaf}}}
			raw = map[string]any{"child": raw}
			path = "default.child.extra"
		}
		ctx := NewAttributeContext(false, false, true, "", NewNameScope())
		_, err := ctx.DefaultValue(target, raw)
		require.ErrorContains(t, err, path+": authored default member has no native Go field")
	}
}

func TestObjectDefaultRetainsCustomJSONBoundary(t *testing.T) {
	for _, kind := range []string{"root", "array", "object"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			var raw any = defaultJSONCodec{calls: &calls}
			target := &expr.AttributeExpr{Type: expr.Any}
			if kind == "array" {
				target = &expr.AttributeExpr{Type: &expr.Array{ElemType: target}}
				raw = []any{raw}
			}
			if kind == "object" {
				target = &expr.AttributeExpr{Type: &expr.Object{{Name: "data", Attribute: target}}}
				raw = map[string]any{"data": raw}
			}
			ctx := NewAttributeContext(false, false, true, "", NewNameScope())
			literal, err := ctx.DefaultValue(target, raw)
			require.NoError(t, err)
			require.Contains(t, literal, `loom.JSONValue("{\"custom\":true}")`)
			require.Equal(t, 1, calls, "only the selected raw-JSON target codec materializes the value")
		})
	}
}

func TestObjectDefaultErrorPropagation(t *testing.T) {
	for _, carrier := range []string{"Native", "Optional", "Nullable"} {
		t.Run(carrier, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: &expr.Object{{Name: "x", Attribute: &expr.AttributeExpr{Type: expr.Int}}}, Nullable: carrier == "Nullable"}
			source := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: expr.DupAtt(field)}}}
			field.DefaultValue = map[string]any{"x": 1, "extra": 2}
			target := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
			scope := NewNameScope()
			sourceCtx := NewAttributeContext(false, false, false, "", scope)
			sourceCtx.JSONPresence = carrier == "Optional"
			targetCtx := NewAttributeContext(false, false, true, "", scope)
			_, _, err := GoTransform(source, target, "source", "target", sourceCtx, targetCtx, "", true)
			require.ErrorContains(t, err, "default.extra: authored default member has no native Go field")
		})
	}
}

func TestObjectDefaultNullableAnyMember(t *testing.T) {
	target := &expr.AttributeExpr{Type: &expr.Object{{Name: "data", Attribute: &expr.AttributeExpr{Type: expr.Any, Nullable: true}}}}
	context := NewAttributeContext(false, false, true, "", NewNameScope())
	for _, tc := range []struct {
		raw  any
		want string
	}{
		{nil, "loom.NullValue[loom.JSONValue]()"},
		{"value", `loom.NullableValue[loom.JSONValue](loom.JSONValue("\"value\""))`},
	} {
		literal, err := context.DefaultValue(target, map[string]any{"data": tc.raw})
		require.NoError(t, err)
		require.Contains(t, literal, tc.want)
	}
}

func TestObjectDefaultRejectsInfiniteNestedDefaults(t *testing.T) {
	node := &expr.UserTypeExpr{TypeName: "Node", AttributeExpr: &expr.AttributeExpr{}}
	node.Type = &expr.Object{{Name: "next", Attribute: &expr.AttributeExpr{Type: node, DefaultValue: map[string]any{}}}}
	context := NewAttributeContext(false, false, true, "", NewNameScope())
	_, err := context.DefaultValue(&expr.AttributeExpr{Type: node}, map[string]any{})
	require.ErrorContains(t, err, "nested defaults require an infinite object value")
}

func (defaultJSONErrorCodec) MarshalJSON() ([]byte, error) {
	return nil, errors.New("codec refused value")
}

func TestObjectDefaultReturnsCodecError(t *testing.T) {
	context := NewAttributeContext(false, false, true, "", NewNameScope())
	target := &expr.AttributeExpr{Type: &expr.Object{{Name: "data", Attribute: &expr.AttributeExpr{Type: expr.Any}}}}
	_, err := context.DefaultValue(target, map[string]any{"data": defaultJSONErrorCodec{}})
	require.ErrorContains(t, err, "default.data: encode authored JSON value")
	require.ErrorContains(t, err, "codec refused value")
}
