package unionjson

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestPlanOwnsRepresentationPolicy(t *testing.T) {
	source := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Page", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: "total:count", Attribute: &expr.AttributeExpr{Type: expr.Int}},
			{Name: "hidden", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"openapi:generate": {"false"}}}},
		},
		Validation: &expr.ValidationExpr{Required: []string{"total:count"}},
	}}}
	for _, tc := range []struct {
		name           string
		fieldName      func(string) string
		schema, closed bool
		wire           string
		want           bool
	}{
		{"service spelling", expr.AttributeName, true, false, `{"total":0}`, true},
		{"transport spelling", expr.ElementName, true, false, `{"count":0}`, true},
		{"wrong spelling", expr.ElementName, true, false, `{"total":0}`, false},
		{"default open", expr.ElementName, true, false, `{"count":0,"extra":1}`, true},
		{"API closed", expr.ElementName, true, true, `{"count":0,"extra":1}`, false},
		{"hidden schema field", expr.ElementName, true, true, `{"count":0,"hidden":"x"}`, false},
		{"runtime field", expr.ElementName, false, true, `{"count":0,"hidden":"x"}`, true},
	} {
		for _, container := range []string{"object", "map", "union"} {
			target, wire := source, tc.wire
			if container == "map" {
				target = &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: source}}
				wire = `{"entry":` + wire + `}`
			}
			if container == "union" {
				target = &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
					{Name: "Page", Attribute: source},
					{Name: "Strings", Attribute: &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Strings", AttributeExpr: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}}}},
				}}}
			}
			t.Run(fmt.Sprintf("%s/%s", tc.name, container), func(t *testing.T) {
				plan, err := Plan(target, tc.fieldName, tc.schema, tc.closed)
				require.NoError(t, err)
				shape, err := plan.JSONShape()
				require.NoError(t, err)
				matched, err := shape.Match([]byte(wire))
				require.NoError(t, err)
				require.Equal(t, tc.want, matched)
			})
		}
	}
	require.Empty(t, source.Type.(expr.UserType).Attribute().Meta, "projection must not mutate the authored type")
}
