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
				plan, err := Plan(target, func(name string, attribute *expr.AttributeExpr) string {
					return expr.JSONFieldName(tc.fieldName(name), attribute)
				}, tc.schema, tc.closed)
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

// TestPlanHonorsAuthoredOpenness checks that API closed-object mode closes only
// objects without an authored policy, matching OpenAPI emission.
func TestPlanHonorsAuthoredOpenness(t *testing.T) {
	for _, tc := range []struct {
		name   string
		meta   []string
		closed bool
		want   bool
	}{
		{"unset under API closed", nil, true, false},
		{"explicit open under API closed", []string{"true"}, true, true},
		{"explicit closed", []string{"false"}, false, false},
		{"unset", nil, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extras := &expr.AttributeExpr{Type: &expr.Object{{Name: "tag", Attribute: &expr.AttributeExpr{Type: expr.String}}}}
			if tc.meta != nil {
				extras.Meta = expr.MetaExpr{"openapi:additionalProperties": tc.meta}
			}
			branch := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Branch", AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Object{{Name: "extras", Attribute: &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Extras", AttributeExpr: extras}}}},
			}}}
			plan, err := Plan(branch, transportJSONFieldName, true, tc.closed)
			require.NoError(t, err)
			shape, err := plan.JSONShape()
			require.NoError(t, err)
			matched, err := shape.Match([]byte(`{"extras":{"tag":"t","more":1}}`))
			require.NoError(t, err)
			require.Equal(t, tc.want, matched)
		})
	}
}

// TestPlanAppliesOccurrenceOpenness checks that a named-type occurrence's own
// policy governs that occurrence only, including recursive occurrences.
func TestPlanAppliesOccurrenceOpenness(t *testing.T) {
	extras := &expr.UserTypeExpr{TypeName: "Extras", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "tag", Attribute: &expr.AttributeExpr{Type: expr.String}}},
	}}
	occurrence := func(policy string) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: extras, Meta: expr.MetaExpr{"openapi:additionalProperties": {policy}}}
	}
	branch := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Branch", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "open", Attribute: occurrence("true")},
		{Name: "closed", Attribute: occurrence("false")},
		{Name: "plain", Attribute: &expr.AttributeExpr{Type: extras}},
	}}}}
	node := &expr.UserTypeExpr{TypeName: "Node"}
	node.AttributeExpr = &expr.AttributeExpr{Type: &expr.Object{
		{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "next", Attribute: &expr.AttributeExpr{Type: node, Meta: expr.MetaExpr{"openapi:additionalProperties": {"false"}}}},
	}}
	for _, tc := range []struct {
		name   string
		source *expr.AttributeExpr
		closed bool
		wire   string
		want   bool
	}{
		{"open occurrence under API closed", branch, true, `{"open":{"tag":"t","more":1}}`, true},
		{"plain occurrence under API closed", branch, true, `{"plain":{"tag":"t","more":1}}`, false},
		{"closed occurrence", branch, false, `{"closed":{"tag":"t","more":1}}`, false},
		{"plain occurrence", branch, false, `{"plain":{"tag":"t","more":1}}`, true},
		{"recursive root keeps type policy", &expr.AttributeExpr{Type: node}, false, `{"id":"a","more":1}`, true},
		{"recursive occurrence", &expr.AttributeExpr{Type: node}, false, `{"next":{"id":"b","more":1}}`, false},
		{"nested recursive occurrence", &expr.AttributeExpr{Type: node}, false, `{"next":{"next":{"id":"c","more":1}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := Plan(tc.source, transportJSONFieldName, true, tc.closed)
			require.NoError(t, err)
			shape, err := plan.JSONShape()
			require.NoError(t, err)
			matched, err := shape.Match([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, matched)
		})
	}
	require.Empty(t, extras.Attribute().Meta, "projection must not mutate the authored type")
}
