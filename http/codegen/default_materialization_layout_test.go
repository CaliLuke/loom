package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestBodyDefaultMaterializationLayout(t *testing.T) {
	for _, fields := range []bool{false, true} {
		for _, elements := range []bool{false, true} {
			t.Run(fmt.Sprintf("fields_%t_elements_%t", fields, elements), func(t *testing.T) {
				attribute := &expr.AttributeExpr{Type: &expr.Object{
					{Name: "count:wire_count", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 9}},
					{Name: "labels", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, DefaultValue: []string{"body"}}},
				}}
				context := httpContext(codegen.NewNameScope(), true, true)
				context.JSONPresence, context.CollectionElementPresence = fields, elements
				literal, err := context.DefaultValue(attribute, map[string]any{}, codegen.DefaultValueOptions{MaterializeMembers: true})
				require.NoError(t, err)
				declaration := goTypeDef(context.Scope.Scope(), attribute, context)
				require.True(t, strings.HasPrefix(literal.Expression, "&"+declaration+"{"), literal.Expression)
				countType, labelsType := "*int", "[]string"
				if elements {
					labelsType = "[]loom.Optional[string]"
				}
				if fields {
					countType, labelsType = "loom.Optional[int]", "loom.Optional["+labelsType+"]"
				}
				require.Contains(t, declaration, "Count "+countType)
				require.Contains(t, declaration, "Labels "+labelsType)
				require.Contains(t, declaration, `json:"wire_count`)
				require.Contains(t, literal.Expression, "9")
				require.Contains(t, literal.Expression, `"body"`)
			})
		}
	}
}
