package expr_test

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestNumericEnumNestedRefinement(t *testing.T) {
	var derived expr.UserType
	expr.RunDSL(t, func() {
		base := Type("Base", func() {
			Attribute("n", Int)
			Enum(map[string]any{"n": 1}, map[string]any{"n": 2})
		})
		derived = Type("Derived", func() {
			Extend(base)
			Attribute("n", Int, func() {
				Minimum(2)
			})
		})
		Service("numbers", func() {
			Method("read", func() {
				Payload(derived)
			})
		})
	})
	attribute := &expr.AttributeExpr{Type: derived}
	wire, err := expr.InlineJSONSchema(attribute)
	require.NoError(t, err)
	require.Contains(t, string(wire), `"enum":[{"n":1},{"n":2}]`)
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation})
	require.NoError(t, err)
	shape, err := plan.JSONShape()
	require.NoError(t, err)
	for _, test := range []struct {
		wire string
		want bool
	}{{`{"n":1}`, false}, {`{"n":2}`, true}, {`{"n":3}`, false}} {
		accepted, err := plan.MatchJSONSchema(jsontext.Value(test.wire))
		require.NoError(t, err)
		require.Equal(t, test.want, accepted)
		accepted, err = shape.Match(jsontext.Value(test.wire))
		require.NoError(t, err)
		require.Equal(t, test.want, accepted)
	}
	// Declaration identities must survive while target field names still come
	// from the actual representation plan, for both schema and runtime checks.
	object := derived.Attribute()
	objectOccurrence, err := context.NewOccurrence(object)
	require.NoError(t, err)
	target := expr.DupAtt(object)
	for _, use := range []expr.ValuePlanUse{expr.ValuePlanDocumentation, expr.ValuePlanRuntime} {
		request := expr.ValuePlanRequest{
			Target: target, Codec: expr.ValueCodecJSON, Use: use,
			Fields:     []expr.ValueFieldPolicy{{Parent: target, Target: target.Find("n"), Name: "n", WireName: "number", Visible: true, Presence: expr.ValueFieldRetain, NumericKind: expr.IntKind}},
			Containers: []expr.ValueContainerPolicy{{Target: target, PreserveAdditional: true}},
		}
		mapped, err := context.NewValuePlan(objectOccurrence, request)
		require.NoError(t, err)
		accepted, err := mapped.MatchJSONSchema(jsontext.Value(`{"number":2}`))
		require.NoError(t, err)
		require.True(t, accepted)
		resolved := context.Resolve(objectOccurrence, context.SupplyValue(expr.ValueInput{Raw: map[string]any{"n": 2}}), expr.ValueRoleExample)
		projected := context.ProjectJSON(resolved, mapped)
		require.Equal(t, expr.ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
		if use == expr.ValuePlanDocumentation {
			annotations, err := context.PrepareJSONSchemaAnnotations(objectOccurrence, mapped)
			require.NoError(t, err)
			require.Equal(t, "Base", annotations.Enums[0][0].Declaration().Declaration)
			wire, present := annotations.Enums[0][0].Projection().JSON()
			require.True(t, present)
			require.JSONEq(t, `{"number":1}`, string(wire))
		}
	}
}
