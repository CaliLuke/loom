package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestMapBranchConversionCollectsReferencedHelper(t *testing.T) {
	for _, proto := range []bool{false, true} {
		name := "from protobuf"
		if proto {
			name = "to protobuf"
		}
		t.Run(name, func(t *testing.T) {
			service := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
				TypeName: "Index",
				AttributeExpr: &expr.AttributeExpr{Type: &expr.Map{
					KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int},
				}},
			}}
			message := expr.DupAtt(service)
			wrapCollectionUserType(message)
			source, target := message, service
			sourceContext := protoBufTypeContext("pb", codegen.NewNameScope(), true)
			targetContext := serviceTypeContext("svc", codegen.NewNameScope())
			if proto {
				source, target = target, source
				sourceContext, targetContext = targetContext, sourceContext
			}
			attributes := &transformAttrs{
				TransformAttrs: &codegen.TransformAttrs{SourceCtx: sourceContext, TargetCtx: targetContext},
				proto:          proto,
			}
			seen := make(map[string]*codegen.TransformFunctionData)
			helper, stop, err := unionBranchMessageHelper(source, target, attributes, seen)
			require.NoError(t, err)
			require.False(t, stop)
			require.NotNil(t, helper, "a wrapped map branch needs its conversion helper")
			require.Contains(t, convertType(source, target, false, false, "value", attributes), helper.Name+"(value)")
			require.Contains(t, helper.Code, ".Field")
			helper, stop, err = unionBranchMessageHelper(source, target, attributes, seen)
			require.NoError(t, err)
			require.True(t, stop)
			require.Nil(t, helper)
		})
	}
}
