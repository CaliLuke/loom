package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestUnionNativeAnyPhysicalRepresentation(t *testing.T) {
	for _, builder := range []struct {
		name  string
		build func(*expr.Union, *codegen.NameScope, *codegen.Location, ...string) *UnionTypeData
	}{
		{"service", buildUnionTypeData},
		{"view", buildViewUnionTypeData},
	} {
		t.Run(builder.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: expr.Any}
			union := &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{{Name: "value", Attribute: attribute}}}
			data := builder.build(union, codegen.NewNameScope(), nil)
			require.Len(t, data.Fields, 1)
			require.True(t, data.Fields[0].EmitPrimitiveAlias)
			require.Equal(t, "loom.JSONValue", data.Fields[0].PrimitiveAliasType)
		})
	}
}
