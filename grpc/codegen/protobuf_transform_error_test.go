package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// TestProtoBufTransformError drives the error paths of protoBufTransform with
// incompatible source and target attribute trees. The expressions are built by
// hand because valid designs always produce matching protobuf and service
// types; only broken or hand-mutated trees reach these branches.
func TestProtoBufTransformError(t *testing.T) {
	scope := codegen.NewNameScope()
	svcCtx := serviceTypeContext("", scope)
	pbCtx := protoBufTypeContext("", scope, false)

	newObjectUT := func(name string, fieldType expr.DataType) expr.UserType {
		return &expr.UserTypeExpr{
			TypeName: name,
			AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Object{
					{Name: "a", Attribute: &expr.AttributeExpr{Type: fieldType}},
				},
			},
		}
	}
	arrayOf := func(elem expr.DataType) *expr.Array {
		return &expr.Array{ElemType: &expr.AttributeExpr{Type: elem}}
	}
	mapOf := func(key, elem expr.DataType) *expr.Map {
		return &expr.Map{
			KeyType:  &expr.AttributeExpr{Type: key},
			ElemType: &expr.AttributeExpr{Type: elem},
		}
	}
	unionOf := func(name string, types ...expr.DataType) *expr.Union {
		values := make([]*expr.NamedAttributeExpr, len(types))
		for i, dt := range types {
			values[i] = &expr.NamedAttributeExpr{
				Name:      dt.Name(),
				Attribute: &expr.AttributeExpr{Type: dt},
			}
		}
		return &expr.Union{TypeName: name, Values: values}
	}

	cases := []struct {
		Name   string
		Source expr.DataType
		Target expr.DataType
		Proto  bool
		Error  string
	}{
		{
			Name:   "top-level-primitive-kind-mismatch",
			Source: expr.String,
			Target: expr.Int,
			Proto:  true,
			Error:  "source is a string but target.Field type is int",
		},
		{
			Name:   "top-level-object-to-array",
			Source: newObjectUT("Source", expr.String),
			Target: &expr.UserTypeExpr{TypeName: "Target", AttributeExpr: &expr.AttributeExpr{Type: arrayOf(expr.String)}},
			Proto:  false,
			Error:  "source.Field is an object but target type is Target",
		},
		{
			Name:   "nested-array-element-mismatch",
			Source: arrayOf(arrayOf(expr.String)),
			Target: arrayOf(arrayOf(expr.Int)),
			Proto:  true,
			Error:  "[0] is a string but [0] type is int",
		},
		{
			Name:   "map-key-mismatch",
			Source: mapOf(expr.String, expr.Int),
			Target: mapOf(expr.Int, expr.Int),
			Proto:  true,
			Error:  "is a string but",
		},
		{
			Name:   "map-element-mismatch",
			Source: mapOf(expr.String, arrayOf(expr.String)),
			Target: mapOf(expr.String, mapOf(expr.String, expr.String)),
			Proto:  true,
			Error:  "is an array but",
		},
		{
			Name:   "object-field-kind-mismatch",
			Source: newObjectUT("SourceHolder", arrayOf(expr.String)),
			Target: newObjectUT("TargetHolder", mapOf(expr.String, expr.String)),
			Proto:  true,
			Error:  "is an array but",
		},
		{
			Name:   "union-value-count-mismatch",
			Source: newObjectUT("SourceUnionHolder", unionOf("SourceUnion", expr.String)),
			Target: newObjectUT("TargetUnionHolder", unionOf("TargetUnion", expr.String, expr.Int)),
			Proto:  true,
			Error:  "cannot transform union attribute SourceUnion with 1 types to union attribute TargetUnion with 2 types",
		},
		{
			Name: "helper-body-mismatch",
			Source: newObjectUT("SourceWrapper", newObjectUT("SourceInner",
				arrayOf(expr.String))),
			Target: newObjectUT("TargetWrapper", newObjectUT("TargetInner",
				arrayOf(expr.Int))),
			Proto: true,
			Error: "is a string but",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			source := &expr.AttributeExpr{Type: c.Source}
			target := &expr.AttributeExpr{Type: c.Target}
			srcCtx, tgtCtx := svcCtx, pbCtx
			if !c.Proto {
				srcCtx, tgtCtx = pbCtx, svcCtx
			}
			code, helpers, err := protoBufTransform(source, target, "source", "target", srcCtx, tgtCtx, c.Proto, true)
			require.Error(t, err)
			require.ErrorContains(t, err, c.Error)
			require.Empty(t, code)
			require.Empty(t, helpers)
		})
	}
}
