package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestProtoFieldNumberIdentityIgnoresUnusedUnionSeed(t *testing.T) {
	baseline := protoAttributeShape(explicitUnionNumberShape("17", "2"))
	for _, seed := range []string{"17", "99", "abc", "18446744073709551616"} {
		t.Run(seed, func(t *testing.T) {
			attribute := explicitUnionNumberShape(seed, "2")
			var shape string
			require.NotPanics(t, func() {
				shape = protoAttributeShape(attribute)
			}, "a non-emitted seed is not a protobuf field")
			require.Equal(t, baseline, shape, "identical emitted branches have identical shape")
			union := expr.AsObject(attribute.Type).Attribute("choice")
			require.Equal(t, seed, union.Meta["rpc:tag"][0], "authored metadata must remain unchanged")
		})
	}
	require.NotEqual(t, baseline, protoAttributeShape(explicitUnionNumberShape("17", "3")), "emitted branch numbers remain part of shape identity")
}

func explicitUnionNumberShape(seed, second string) *expr.AttributeExpr {
	return &expr.AttributeExpr{Type: &expr.Object{{
		Name: "choice",
		Attribute: &expr.AttributeExpr{
			Meta: expr.MetaExpr{"rpc:tag": {seed}},
			Type: &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
				{Name: "first", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"rpc:tag": {"1"}}}},
				{Name: "second", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: expr.MetaExpr{"rpc:tag": {second}}}},
			}},
		},
	}}}
}
