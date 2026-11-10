package representation

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/stretchr/testify/require"
)

func TestBodyMediaCaptureRebuildAndSlotIsolation(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("slots", func() {
			dsl.Method("show", func() {
				dsl.Result(func() {
					dsl.Attribute("count", dsl.Int)
				})
				dsl.HTTP(func() {
					dsl.GET("/")
				})
			})
		})
	})
	prepared, err := PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	response := prepared.Endpoints[0].Response.Responses[0]
	context := response.BodyValue.ProjectionContext
	media := mediaTargets(response.BodyValue, []string{"one", "two", "invalid"})
	media["invalid"].Selection = []string{"missing"}
	first := prepareBodyValuePlans(response.BodyValue, media, response.Body, context)
	require.NoError(t, first.Error)
	require.ErrorContains(t, media["invalid"].Error, "selection")
	for _, name := range []string{"one", "two"} {
		target := media[name]
		require.NoError(t, target.Error)
		require.Same(t, first.ProjectionContext, target.ProjectionContext)
		require.False(t, first.Plan.Root() == target.Plan.Root())
		require.False(t, first.AnchorPlan.Root() == target.AnchorPlan.Root())
		require.False(t, first.ExamplePlan.Root() == target.ExamplePlan.Root())
	}
	first.ExampleSets = map[expr.ValuePlanNode]*transportir.ValueExampleSet{first.Plan.Root(): {Prepared: true}}
	require.Nil(t, media["one"].ExampleSets, "example attachments belong to each slot")
	field := expr.AsObject(response.Body.Type).Attribute("count")
	field.Meta = expr.MetaExpr{"struct:tag:json": {"renamed"}}
	rebuilt := prepareBodyValuePlans(response.BodyValue, nil, response.Body, context)
	require.NoError(t, rebuilt.Error)
	require.Equal(t, "count", underlyingPlan(first.Plan.Root()).Members()[0].WireName)
	require.Equal(t, "renamed", underlyingPlan(rebuilt.Plan.Root()).Members()[0].WireName)
	require.NotSame(t, first.ProjectionContext, rebuilt.ProjectionContext, "a new preparation captures fresh mutable input")
}

func TestBodyMediaCaptureSeparatesCodecsAndContexts(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("slots", func() {
			dsl.Method("show", func() {
				dsl.Result(dsl.String)
				dsl.HTTP(func() {
					dsl.GET("/")
				})
			})
		})
	})
	prepared, err := PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	response := prepared.Endpoints[0].Response.Responses[0]
	media := mediaTargets(response.BodyValue, []string{"json", "text", "foreign"})
	media["text"].Codec = expr.ValueCodecText
	other := expr.NewValueContext()
	occurrence, err := other.NewOccurrence(root.Services[0].Methods[0].Result)
	require.NoError(t, err)
	media["foreign"] = ValueTarget(&service.ValueData{Context: other, Occurrence: occurrence}, response.Body, "", expr.ValueCodecJSON, true)
	primary := prepareBodyValuePlans(response.BodyValue, media, response.Body, response.BodyValue.ProjectionContext)
	require.NoError(t, primary.Error)
	for name, codec := range map[string]expr.ValueCodec{"json": expr.ValueCodecJSON, "text": expr.ValueCodecText, "foreign": expr.ValueCodecJSON} {
		require.NoError(t, media[name].Error, name)
		require.Equal(t, codec, media[name].Plan.Root().Codec())
	}
	require.Same(t, primary.ProjectionContext, media["json"].ProjectionContext)
	require.NotSame(t, primary.ProjectionContext, media["text"].ProjectionContext)
	require.NotSame(t, primary.ProjectionContext, media["foreign"].ProjectionContext)
}
