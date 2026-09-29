package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
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
