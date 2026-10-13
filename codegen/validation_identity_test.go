package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNamedValidationOccurrenceIdentity(t *testing.T) {
	for _, named := range []bool{false, true} {
		for _, intrinsic := range []bool{false, true} {
			for _, occurrence := range []bool{false, true} {
				t.Run(fmt.Sprintf("named=%t/intrinsic=%t/occurrence=%t", named, intrinsic, occurrence), func(t *testing.T) {
					node := &expr.UserTypeExpr{TypeName: "Node", UID: "Node", AttributeExpr: &expr.AttributeExpr{
						Nullable: intrinsic,
						Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{
							Type: expr.String, Validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"},
						}}},
					}}
					meta := expr.MetaExpr{}
					if named {
						meta["struct:type:name"] = []string{"WireNode"}
					}
					field := &expr.AttributeExpr{Type: node, Meta: meta, Nullable: occurrence,
						Validation: &expr.ValidationExpr{Required: []string{"value"}}}
					parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "node", Attribute: field}}}
					ctx := NewAttributeContext(true, false, false, "", NewNameScope())
					ctx.ValidationPrefix = "validateAccess"
					definition := NamedValidationFunction(&expr.AttributeExpr{Type: node, Meta: meta}, ctx, "node")
					body := ValidationCode(parent, nil, ctx, true, false, false, "body")
					require.Contains(t, body, definition.Name+"(")
					if named {
						require.NotContains(t, body, "validateAccessNode(")
					}
					if intrinsic {
						require.Contains(t, body, definition.Name+"(body.Node,")
						require.NotContains(t, body, definition.Name+"(&actual,")
					} else if occurrence {
						require.Contains(t, body, definition.Name+"(&actual,")
					}
					require.Contains(t, body, "Value == nil", "occurrence requiredness must remain local")
					require.Equal(t, occurrence, field.Nullable)
					require.Equal(t, intrinsic, node.Nullable)
				})
			}
		}
	}
}
