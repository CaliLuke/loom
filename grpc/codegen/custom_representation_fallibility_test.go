package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestCustomStringProtoFallibility(t *testing.T) {
	custom := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"scalar.Text", "example.com/customgrpc/scalar"}}}
	native := &expr.AttributeExpr{Type: expr.String}
	for _, customFirst := range []bool{false, true} {
		fields := expr.Object{{Name: "native", Attribute: native}, {Name: "custom", Attribute: custom}}
		if customFirst {
			fields[0], fields[1] = fields[1], fields[0]
		}
		service := &expr.AttributeExpr{Type: &fields}
		message := expr.DupAtt(service)
		scope := codegen.NewNameScope()
		svcCtx := serviceTypeContext("svc", scope)
		pbCtx := protoBufTypeContext("pb", scope, false)
		require.True(t, conversionCanFail(service, message, svcCtx, pbCtx))
		require.True(t, conversionCanFail(message, service, pbCtx, svcCtx))
		selected := &expr.AttributeExpr{Type: &expr.Object{{Name: "native", Attribute: native}}}
		require.False(t, conversionCanFail(service, selected, svcCtx, pbCtx))
		require.False(t, conversionCanFail(selected, service, pbCtx, svcCtx))
	}
}
