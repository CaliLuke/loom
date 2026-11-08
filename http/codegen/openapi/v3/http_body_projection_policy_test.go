package openapiv3_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/stretchr/testify/require"
)

func TestHTTPBodyContributorCapturedFieldPolicy(t *testing.T) {
	var parent expr.UserType
	root := httpgen.RunHTTPDSL(t, func() {
		API("captured field policy", func() {
			Meta("openapi:version", "3.1")
		})
		parent = Type("Parent", func() {
			Attribute("id", String)
			Attribute("count", Int, func() {
				Meta("struct:tag:json", "count,omitzero")
			})
			Attribute("nested", func() {
				Attribute("count", Int, func() {
					Meta("struct:tag:json", "count,omitzero")
				})
			})
			Example(map[string]any{"count": 0})
		})
		child := Type("Child", func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Required("count")
			Example(map[string]any{"count": 0})
		})
		Service("policy", func() {
			Method("response", func() {
				Result(child)
				HTTP(func() {
					GET("/response")
					Response(StatusOK, func() {
						Header("id")
					})
				})
			})
		})
	})
	prepared, err := representation.PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	response := prepared.Endpoints[0].Response.Responses[0]
	actual := response.BodyValue
	rootNode := actual.Plan.Root()
	for rootNode.Underlying().Valid() {
		rootNode = rootNode.Underlying()
	}
	require.Len(t, rootNode.Members(), 2)
	selected := rootNode.Members()[0]
	require.True(t, selected.Required)
	require.Equal(t, expr.ValueFieldOmitZero, selected.Presence, "actual selected native required field omits zero")
	wire, err := representation.CaptureWireContext(response.Body, actual, actual.Plan.Root())
	require.NoError(t, err)
	contributor, err := representation.ProjectContributor(wire, response.Body.HTTPBodyProjection(), parent)
	require.NoError(t, err)
	node := contributor.Target.Plan.Root()
	for node.Underlying().Valid() {
		node = node.Underlying()
	}
	require.Len(t, node.Members(), 2)
	require.False(t, node.Members()[0].Required, "Parent own optional semantics remain independent of Child")
	require.Equal(t, selected.Presence, node.Members()[0].Presence, "common member retains actual selected body omission")
	nested := node.Members()[1].Node
	for nested.Underlying().Valid() {
		nested = nested.Underlying()
	}
	require.Len(t, nested.Members(), 1)
	actualNested := rootNode.Members()[1].Node
	for actualNested.Underlying().Valid() {
		actualNested = actualNested.Underlying()
	}
	require.Len(t, actualNested.Members(), 1)
	require.Equal(t, actualNested.Members()[0].Presence, nested.Members()[0].Presence, "nested field keeps its own captured policy")
	require.NotEqual(t, selected.Presence, nested.Members()[0].Presence, "same-named nested field does not inherit root member policy")
}
