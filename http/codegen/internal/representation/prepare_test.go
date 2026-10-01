package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestSchemaPreparationDoesNotSelectOrSynthesize(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("values", func() {
			dsl.Method("store", func() {
				dsl.Payload(func() {
					dsl.Attribute("data", dsl.Bytes)
					dsl.Attribute("custom", dsl.Bytes, func() {
						dsl.Meta("struct:field:type", "other.Bytes", "example.com/other")
					})
				})
				dsl.HTTP(func() {
					dsl.POST("/")
				})
			})
		})
	})
	root.API.ExampleGenerator = expr.NewRandom("schema-no-synthesis")
	control := expr.NewRandom("schema-no-synthesis")
	prepared, err := PrepareService(root.API.HTTP.Services[0], nil)
	require.NoError(t, err)
	require.Equal(t, control.Int(), root.API.ExampleGenerator.Int())
	target := prepared.Endpoints[0].Request.BodyValue
	require.NoError(t, target.Error)
	require.Zero(t, target.Source.Example.Outcome())
	fields := underlyingPlan(target.Plan.Root()).Members()
	require.Len(t, fields, 2)
	require.Equal(t, expr.ValueCodecJSON, fields[0].Node.Codec())
	require.Equal(t, expr.ValueCodecCustom, fields[1].Node.Codec())
	semantic := service.NewServicesData(root).Get("values")
	emitter, err := PrepareService(root.API.HTTP.Services[0], semantic)
	require.NoError(t, err)
	paired := emitter.Endpoints[0].Request.BodyValue
	require.Same(t, semantic.Method("store").PayloadValue, paired.Source)
	require.Equal(t, target.Codec, paired.Codec)
	require.Equal(t, target.Selection, paired.Selection)
	paired = BuildValuePlan(paired, emitter.Endpoints[0].Request.Body, schemaContext(true), expr.ValuePlanSchema)
	require.NoError(t, paired.Error)
	actual := underlyingPlan(paired.Plan.Root()).Members()
	for i := range fields {
		require.Equal(t, fields[i].Name, actual[i].Name)
		require.Equal(t, fields[i].WireName, actual[i].WireName)
		require.Equal(t, fields[i].Node.Codec(), actual[i].Node.Codec())
	}
}

func underlyingPlan(node expr.ValuePlanNode) expr.ValuePlanNode {
	for node.Underlying().Valid() {
		node = node.Underlying()
	}
	return node
}

func TestSchemaPreparationMappedLocationAncestry(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("mapped", func() {
			dsl.Method("get", func() {
				dsl.Payload(func() { dsl.Attribute("blob", dsl.Bytes) })
				dsl.HTTP(func() { dsl.GET("/"); dsl.Param("blob") })
			})
		})
	})
	prepared, err := PrepareService(root.API.HTTP.Services[0], nil)
	require.NoError(t, err)
	location := prepared.Endpoints[0].Request.QueryParams[0].Value
	require.NoError(t, location.Error)
	require.Equal(t, expr.ValueCodecText, location.Plan.Root().Codec())
}

func TestPrepareEndpointExamplesSkipsSecurityParameters(t *testing.T) {
	root := codegen.RunDSL(t, testdata.AsyncSessionSecurityDSL)
	httpService := root.API.HTTP.Services[0]
	prepared, err := PrepareService(httpService, nil)
	require.NoError(t, err)
	require.NoError(t, PrepareServiceExamples(prepared, httpService, root.API.ExampleGenerator))

	endpoint := prepared.Endpoints[0]
	require.Len(t, endpoint.Request.PathParams, 1)
	require.Len(t, endpoint.Request.Cookies, 1)
	parameter := endpoint.Request.PathParams[0]
	credential := endpoint.Request.Cookies[0]
	require.Error(t, credential.Value.Error,
		"the transport-owned session credential has no ordinary payload source")
	require.False(t, parameter.Value.ExamplesPrepared)
	require.False(t, credential.Value.ExamplesPrepared)

	require.NoError(t, PrepareEndpointExamples(endpoint, root.API.ExampleGenerator))
	require.True(t, parameter.Value.ExamplesPrepared,
		"ordinary payload parameters must still prepare examples")
	require.False(t, credential.Value.ExamplesPrepared,
		"security parameters omitted from OpenAPI must not be prepared")
}

func TestPreparedSelectedBodyExamplesUseSelectedAnchorPlan(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("selected", func() {
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("body", dsl.String, func() {
						dsl.Example("first")
						dsl.Example("second")
					})
				})
				dsl.HTTP(func() {
					dsl.POST("/")
					dsl.Body("body")
				})
			})
		})
	})
	service := root.API.HTTP.Services[0]
	prepared, err := PrepareService(service, nil)
	require.NoError(t, err)
	require.NoError(t, PrepareServiceExamples(prepared, service, root.API.ExampleGenerator))
	require.NoError(t, PrepareEndpointExamples(prepared.Endpoints[0], root.API.ExampleGenerator))
	target := prepared.Endpoints[0].Request.BodyValue
	require.Equal(t, []string{"body"}, target.Selection)
	require.Len(t, target.Examples, 2)
	for _, example := range target.Examples {
		projected := example.Source.Context.ProjectJSON(example.Source.Example, example.Plan)
		_, ok := projected.JSON()
		require.True(t, ok, "selected child ownership must pair with its exact captured target plan")
	}
}

func TestPreparedSelectedBodyRepresentativeKeepsWholeAnchorPlan(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("selected", func() {
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("body", dsl.String)
					dsl.Attribute("header", dsl.String)
				})
				dsl.HTTP(func() {
					dsl.POST("/")
					dsl.Body("body")
					dsl.Header("header")
				})
			})
		})
	})
	httpService := root.API.HTTP.Services[0]
	prepared, err := PrepareService(httpService, nil)
	require.NoError(t, err)
	require.NoError(t, PrepareServiceExamples(prepared, httpService, root.API.ExampleGenerator))
	require.NoError(t, PrepareEndpointExamples(prepared.Endpoints[0], root.API.ExampleGenerator))

	target := prepared.Endpoints[0].Request.BodyValue
	require.NotNil(t, target.Representative)
	require.Same(t, target.Anchor, target.Representative.Source)
	require.Equal(t, target.AnchorPlan.Root(), target.Representative.Plan.Root())
	projected := target.Representative.Source.Context.ProjectJSON(
		target.Representative.Source.Example,
		target.Representative.Plan,
	)
	value, ok := projected.JSON()
	require.True(t, ok)
	require.NotContains(t, string(value), "header")
}

func TestPreparedInlineBodyWrapperReusesStructuralSource(t *testing.T) {
	root := codegen.RunDSL(t, testdata.PayloadBodyInlineObjectDSL)
	httpService := root.API.HTTP.Services[0]
	prepared, err := PrepareService(httpService, nil)
	require.NoError(t, err)
	target := prepared.Endpoints[0].Request.BodyValue
	require.True(t, target.Plan.Root().UnderlyingReusesSource())
	require.True(t, target.AnchorPlan.Root().UnderlyingReusesSource())
	require.False(t, target.ExamplePlan.Root().UnderlyingReusesSource(),
		"the independently captured target occurrence owns its declared wrapper")

	require.NoError(t, PrepareServiceExamples(prepared, httpService, root.API.ExampleGenerator))
	require.NoError(t, PrepareEndpointExamples(prepared.Endpoints[0], root.API.ExampleGenerator))
	require.NotNil(t, target.ExampleSets[target.Plan.Root().Underlying()])
}

func TestBuildValuePlanRebindsPreparedExamplesToRebuiltNodes(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("rebuilt", func() {
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("message", dsl.String)
				})
				dsl.HTTP(func() {
					dsl.POST("/")
				})
			})
		})
	})
	httpService := root.API.HTTP.Services[0]
	prepared, err := PrepareService(httpService, nil)
	require.NoError(t, err)
	require.NoError(t, PrepareServiceExamples(prepared, httpService, root.API.ExampleGenerator))
	require.NoError(t, PrepareEndpointExamples(prepared.Endpoints[0], root.API.ExampleGenerator))

	target := prepared.Endpoints[0].Request.BodyValue
	oldRoot := target.Plan.Root()
	oldSource := target.Source
	oldAnchor := target.Anchor
	oldResult := target.Source.Example
	require.Contains(t, target.ExampleSets, oldRoot)

	rebuiltContext := schemaContext(true)
	rebuiltContext.JSONPresence = false
	target = BuildValuePlan(target, prepared.Endpoints[0].Request.Body, rebuiltContext, expr.ValuePlanSchema)
	require.NoError(t, target.Error)
	require.Same(t, oldSource, target.Source)
	require.Same(t, oldAnchor, target.Anchor)
	require.Equal(t, oldResult, target.Source.Example)
	require.False(t, target.ExamplesPrepared)
	require.Empty(t, target.Examples)
	require.Nil(t, target.Representative)
	require.Nil(t, target.ExampleSets)
	require.NotEqual(t, oldRoot, target.Plan.Root())

	generator := expr.NewRandom("rebind-without-resampling")
	control := expr.NewRandom("rebind-without-resampling")
	require.NoError(t, PrepareTargetExamples(target, prepared.Endpoints[0].Request.Body, generator))
	require.Equal(t, control.Int(), generator.Int(), "retained source result must prevent resampling")
	require.Contains(t, target.ExampleSets, target.Plan.Root())
	require.NotContains(t, target.ExampleSets, oldRoot)
	require.Same(t, oldSource, target.Representative.Source)
}
