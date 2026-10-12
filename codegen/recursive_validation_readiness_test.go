package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestRecursiveValidationReadiness(t *testing.T) {
	for _, shape := range []string{"self", "mutual", "named indirection", "array", "map"} {
		for _, recursiveFirst := range []bool{false, true} {
			for _, presence := range []string{"native", "optional", "nullable"} {
				t.Run(fmt.Sprintf("%s/recursiveFirst=%t/%s", shape, recursiveFirst, presence), func(t *testing.T) {
					node, callee := recursiveValidationReadinessGraph(shape, recursiveFirst, presence == "nullable")
					ctx := NewAttributeContext(true, false, false, "", NewNameScope())
					ctx.JSONPresence = presence == "optional"
					code := ValidationCode(node.Attribute(), node, ctx, true, false, false, "body")
					target := "body.Next"
					if presence != "native" {
						target = "&actual"
					}
					if shape == "array" {
						target = "e"
					}
					if shape == "map" {
						target = "v"
					}
					require.Contains(t, code, "Validate"+callee+"("+target, "named edges must not inline partial buffers: %s", code)
				})
			}
		}
	}
}

func recursiveValidationReadinessGraph(shape string, recursiveFirst, nullable bool) (*expr.UserTypeExpr, string) {
	node := &expr.UserTypeExpr{TypeName: "Node", UID: "Node"}
	var next expr.DataType = node
	callee := "Node"
	switch shape {
	case "mutual":
		other := &expr.UserTypeExpr{TypeName: "Other", UID: "Other"}
		otherFields := expr.Object{
			{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"}}},
			{Name: "back", Attribute: &expr.AttributeExpr{Type: node}},
		}
		other.AttributeExpr = &expr.AttributeExpr{Type: &otherFields}
		next, callee = other, "Other"
	case "named indirection":
		next = &expr.UserTypeExpr{TypeName: "AliasNode", UID: "AliasNode", AttributeExpr: &expr.AttributeExpr{Type: node}}
		callee = "AliasNode"
	case "array":
		next = &expr.Array{ElemType: &expr.AttributeExpr{Type: node}}
	case "map":
		next = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: node}}
	}
	fields := expr.Object{
		{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"}}},
		{Name: "next", Attribute: &expr.AttributeExpr{Type: next, Nullable: nullable}},
	}
	if recursiveFirst {
		fields[0], fields[1] = fields[1], fields[0]
	}
	node.AttributeExpr = &expr.AttributeExpr{Type: &fields, Validation: &expr.ValidationExpr{Required: []string{"value"}}}
	return node, callee
}
