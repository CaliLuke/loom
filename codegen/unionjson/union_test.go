package unionjson

import (
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
		t.Run(tc.name, func(t *testing.T) {
			plan, err := Plan(source, tc.fieldName, tc.schema, tc.closed)
			require.NoError(t, err)
			shape, err := plan.JSONShape()
			require.NoError(t, err)
			matched, err := shape.Match([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, matched)
		})
	}
	require.Empty(t, source.Type.(expr.UserType).Attribute().Meta, "projection must not mutate the authored type")
}
