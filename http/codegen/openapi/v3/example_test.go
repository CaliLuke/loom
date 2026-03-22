package openapiv3

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/v3/expr"
)

func TestShouldSuppressOpenAPIExamplesHandlesRecursiveUserTypes(t *testing.T) {
	node := &expr.UserTypeExpr{
		TypeName:      "Node",
		AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}},
	}
	obj := node.AttributeExpr.Type.(*expr.Object)
	*obj = append(*obj, &expr.NamedAttributeExpr{
		Name:      "child",
		Attribute: &expr.AttributeExpr{Type: node},
	})

	attr := &expr.AttributeExpr{Type: node}
	require.False(t, shouldSuppressOpenAPIExamples(attr, false))
}
