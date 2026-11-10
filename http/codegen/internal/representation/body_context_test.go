package representation

import (
	"fmt"
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

func TestProjectionContextSharesRegistryAndDetachesOverrides(t *testing.T) {
	root := &expr.UserTypeExpr{TypeName: "Root", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}
	sibling := &expr.UserTypeExpr{TypeName: "Sibling", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}
	context := schemaContext(false)
	layout := NewLayout()
	layout.Record(root, false, false, true)
	layout.Record(sibling, true, true, false)
	layout.Bind(context)
	context.TypeLayouts = codegen.CaptureTypeLayouts(context)
	// Include an override outside the target graph: a later contributor must
	// still resolve it, and recorded false must override inherited true.
	layout.JSONPresence[sibling.Hash()] = false
	layout.Pointer[sibling.Hash()] = false
	layout.UseDefault[sibling.Hash()] = true
	captured := captureProjectionContext(context, &expr.AttributeExpr{Type: root})
	denseContext := context.Dup()
	denseContext.TypeLayouts = nil
	dense := captureProjectionContext(denseContext, &expr.AttributeExpr{Type: root})
	identity, err := wireContextIdentity(&WireContext{codec: expr.ValueCodecJSON, context: captured})
	require.NoError(t, err)
	denseIdentity, err := wireContextIdentity(&WireContext{codec: expr.ValueCodecJSON, context: dense})
	require.NoError(t, err)
	require.Equal(t, denseIdentity, identity, "sharing preserves complete contributor identity bytes")
	require.Same(t, context.TypeLayouts, captured.TypeLayouts)
	require.Len(t, captured.JSONPresenceTypes, 1)
	require.NotContains(t, captured.JSONPresenceTypes, root.Hash())
	layout.JSONPresence[sibling.Hash()] = true
	layout.Pointer[sibling.Hash()] = true
	layout.UseDefault[sibling.Hash()] = false
	ApplyLayout(captured, &expr.AttributeExpr{Type: sibling})
	require.False(t, captured.JSONPresence)
	require.False(t, captured.Pointer)
	require.True(t, captured.UseDefault)
	wire := captureProjectionContext(captured, nil)
	captured.JSONPresenceTypes[sibling.Hash()] = true
	require.False(t, wire.TypeLayout(sibling.Hash()).JSONPresence)
}

func TestProjectionContextCaptureAllocation(t *testing.T) {
	for _, count := range []int{16, 1024} {
		t.Run(fmt.Sprintf("declarations_%d", count), func(t *testing.T) {
			context := schemaContext(false)
			layout := NewLayout()
			for index := range count {
				key := fmt.Sprintf("declaration_%d", index)
				layout.JSONPresence[key], layout.Pointer[key], layout.UseDefault[key] = false, false, true
			}
			layout.Bind(context)
			context.TypeLayouts = codegen.CaptureTypeLayouts(context)
			identity, err := wireContextIdentity(&WireContext{codec: expr.ValueCodecJSON, context: context})
			require.NoError(t, err)
			require.Less(t, len(identity), 512, "contributor identities must not duplicate a service registry")
			var captured *codegen.AttributeContext
			allocations := testing.AllocsPerRun(10, func() {
				captured = captureProjectionContext(context, &expr.AttributeExpr{Type: expr.String})
			})
			require.LessOrEqual(t, allocations, 3.0, "capture allocations must be independent of registry size")
			require.Same(t, context.TypeLayouts, captured.TypeLayouts)
		})
	}
}
