package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAdditionalValuePlanContainers(t *testing.T) {
	for _, kind := range []string{"object", "union", "nested", "recursive"} {
		t.Run(kind, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				leaf := dsl.Type("Leaf", func() {
					dsl.Attribute("name", dsl.String)
					dsl.Required("name")
				})
				var additional expr.DataType = leaf
				switch kind {
				case "union":
					additional = dsl.Type("Entry", dsl.OneOf(leaf, dsl.Type("Numbers", dsl.ArrayOf(dsl.Int))), dsl.Untagged)
				case "nested":
					additional = dsl.ArrayOf(dsl.MapOf(dsl.String, leaf))
				}
				record := dsl.Type("Record", func() {
					dsl.Attribute("id", dsl.String)
					if kind == "recursive" {
						dsl.AdditionalAttributes("Record")
					} else {
						dsl.AdditionalAttributes(additional)
					}
				})
				dsl.Service("records", func() {
					dsl.Method("store", func() {
						dsl.Payload(record)
						dsl.HTTP(func() {
							dsl.POST("/")
						})
					})
				})
			})
			prepared, err := PrepareService(root.API.HTTP.Services[0], nil, nil)
			require.NoError(t, err)
			request := prepared.Endpoints[0].Request
			target := BuildValuePlan(request.BodyValue, request.Body, schemaContext(true), expr.ValuePlanRuntime)
			require.NoError(t, target.Error)
			shape, err := target.Plan.JSONShape()
			require.NoError(t, err)
			for shape.Kind == "alias" {
				shape = shape.Child
			}
			require.NotNil(t, shape.Additional)
			additional := shape.Additional
			for additional.Kind == "alias" {
				additional = additional.Child
			}
			want := "object"
			if kind == "union" {
				want = "union"
			} else if kind == "nested" {
				want = "array"
			}
			require.Equal(t, want, additional.Kind)
		})
	}
}

func TestAdditionalValuePlanSeparatesDecoderContexts(t *testing.T) {
	leaf := &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		Meta: expr.MetaExpr{"openapi:additionalProperties": {"false"}},
	}}
	union := &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "Record", Attribute: &expr.AttributeExpr{Type: leaf}},
		{Name: "List", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: leaf}}}},
	}}
	target := TargetGraph(&expr.AttributeExpr{Type: &expr.Object{}, AdditionalAttributes: &expr.AttributeExpr{Type: union}})
	branches := expr.AsUnion(target.AdditionalAttributes.Type).Values
	record := branches[0].Attribute.Type.(expr.UserType).Attribute()
	list := expr.AsArray(branches[1].Attribute.Type).ElemType.Type.(expr.UserType).Attribute()
	require.NotSame(t, record, list, "untagged root checks unknown members; ordinary nested decoder ignores them")
	builder := httpValuePlanBuilder{seen: make(map[*expr.AttributeExpr]bool)}
	require.NoError(t, builder.walk(target, schemaContext(true), false, true))
	policies := make(map[*expr.AttributeExpr]bool)
	for _, policy := range builder.request.Containers {
		policies[policy.Target] = policy.RejectUnknownMembers
	}
	require.Contains(t, policies, record)
	require.Contains(t, policies, list)
	require.True(t, policies[record])
	require.False(t, policies[list])
}

func TestRequiredAdditionalOccurrenceValuePlan(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		record := dsl.Type("Record", func() {
			dsl.Attribute("id", dsl.String)
			dsl.AdditionalAttributes(dsl.Bytes)
		})
		dsl.Service("records", func() {
			dsl.Method("store", func() {
				dsl.Payload(func() {
					dsl.Attribute("record", record, func() {
						dsl.Required("x")
					})
					dsl.Required("record")
				})
				dsl.HTTP(func() {
					dsl.POST("/")
				})
			})
		})
	})
	prepared, err := PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	request := prepared.Endpoints[0].Request
	target := BuildValuePlan(request.BodyValue, request.Body, schemaContext(true), expr.ValuePlanRuntime)
	require.NoError(t, target.Error)
	match, err := target.Plan.MatchJSONSchema([]byte(`{"record":{}}`))
	require.NoError(t, err)
	require.False(t, match, "missing required additional x must not match schema")
	shape, err := target.Plan.JSONShape()
	require.NoError(t, err)
	shapeMatch, err := shape.Match([]byte(`{"record":{}}`))
	require.NoError(t, err)
	require.False(t, shapeMatch, "missing required additional x must not match descriptor")
}
