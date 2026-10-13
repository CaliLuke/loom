package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestProtoNamedValidationOccurrenceIdentity(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		for _, name := range []string{"Node", "Message", "Choice"} {
			t.Run(fmt.Sprintf("%s/explicit=%t", name, explicit), func(t *testing.T) {
				node := &expr.UserTypeExpr{TypeName: name, UID: name, AttributeExpr: &expr.AttributeExpr{
					Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{
						Type: expr.String, Validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"},
					}}},
				}}
				field := &expr.AttributeExpr{Type: node, Meta: expr.MetaExpr{}}
				if explicit {
					field.Meta["struct:name:proto"] = []string{"wire_node"}
				}
				ctx := protoBufTypeContext("pb", codegen.NewNameScope(), false)
				definition := codegen.NamedValidationFunction(field, ctx, "node")
				parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "node", Attribute: field}}}
				body := codegen.ValidationCode(parent, nil, ctx, true, false, false, "message")
				require.Contains(t, body, definition.Name+"(message.Node,")
				if explicit {
					require.Equal(t, "ValidateWireNode", definition.Name)
					require.Equal(t, "*pb.WireNode", definition.Ref)
					require.Equal(t, []string{"wire_node"}, field.Meta["struct:name:proto"])
				}
			})
		}
	}
}
