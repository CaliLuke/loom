package representation

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestBodyProjectionContextCapturesBoundLayout(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("bound", func() {
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("count", dsl.Int, func() {
						dsl.Default(0)
						dsl.Meta("struct:tag:json", "count,omitzero")
					})
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
	selected := request.Body.Type.(expr.UserType)
	context := schemaContext(true)
	layout := NewLayout()
	layout.Record(selected, false, false, true)
	layout.Bind(context)
	target := BuildValuePlan(request.BodyValue, request.Body, context, expr.ValuePlanSchema)
	require.NoError(t, target.Error)
	require.False(t, target.ProjectionContext.Pointer)
	require.True(t, target.ProjectionContext.UseDefault)
	require.False(t, target.ProjectionContext.JSONPresence)
	require.False(t, target.ProjectionContext.CollectionElementPresence)
	require.Equal(t, expr.ValueFieldOmitZero, underlyingPlan(target.Plan.Root()).Members()[0].Presence)
	wire, err := CaptureWireContext(request.Body, target, target.Plan.Root())
	require.NoError(t, err)
	identity := wire.ProjectionIdentity(request.Body.HTTPBodyProjection())
	layout.JSONPresence[selected.Hash()] = true
	layout.Pointer[selected.Hash()] = true
	layout.UseDefault[selected.Hash()] = false
	require.False(t, target.ProjectionContext.JSONPresenceTypes[selected.Hash()], "capture detaches mutable layout registry")
	target.ProjectionContext.JSONPresenceTypes[selected.Hash()] = true
	require.Equal(t, identity, wire.ProjectionIdentity(request.Body.HTTPBodyProjection()), "wire capture owns its policy snapshot")
	require.False(t, wire.context.JSONPresenceTypes[selected.Hash()])
	rebuilt := BuildValuePlan(target, request.Body, context, expr.ValuePlanSchema)
	require.NoError(t, rebuilt.Error)
	require.True(t, rebuilt.ProjectionContext.JSONPresence)
	require.True(t, rebuilt.ProjectionContext.Pointer)
	require.False(t, rebuilt.ProjectionContext.UseDefault)
	require.False(t, wire.context.JSONPresence)
}
