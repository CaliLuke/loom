package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestProjectContributorUsesBodyWireContext projects Parent into the form
// contexts of two Child bodies. Each contributor keeps Parent's own members and
// takes the body's resolved wire facts for the members the body shares: the
// Child-added form name of kind and the occurrence's JSON content override of
// blob. A Child-only encoding slot is not copied, and the two contexts have
// distinct identities.
func TestProjectContributorUsesBodyWireContext(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		parent := dsl.Type("Parent", func() {
			dsl.Meta("openapi:discriminator:propertyName", "kind")
			dsl.Attribute("kind", dsl.String)
			dsl.Attribute("blob", dsl.Bytes)
			dsl.Required("kind")
		})
		for _, wire := range []string{"first", "second"} {
			child := dsl.Type(wire+"Child", func() {
				dsl.Extend(parent)
				dsl.Meta("openapi:allOf:reference", "true")
				dsl.Attribute("kind", dsl.String, func() {
					dsl.Meta("struct:tag:form", wire)
				})
				dsl.Attribute("thumb", dsl.Bytes)
			})
			dsl.Service(wire, func() {
				dsl.Method("submit", func() {
					dsl.Payload(child, func() {
						dsl.Meta("openapi:encoding:blob:contentType", "application/json")
						dsl.Meta("openapi:encoding:thumb:contentType", "application/json")
					})
					dsl.HTTP(func() {
						dsl.POST("/" + wire)
						dsl.FormRequest()
					})
				})
			})
		}
	})
	parent := root.UserType("Parent")
	keys := map[string]bool{}
	for index, wire := range []string{"first", "second"} {
		prepared, err := PrepareService(root.API.HTTP.Services[index], nil, nil)
		require.NoError(t, err)
		request := prepared.Endpoints[0].Request
		body := request.Body
		context, err := CaptureWireContext(body, request.BodyValue, request.BodyValue.Plan.Root())
		require.NoError(t, err)
		contributor, err := ProjectContributor(context, body.HTTPBodyProjection(), parent)
		require.NoError(t, err, "a Child-only encoding slot is not a Parent member")
		require.NoError(t, contributor.Target.Error)
		keys[contributor.Key] = true
		node := contributor.Target.Plan.Root()
		for node.Underlying().Valid() {
			node = node.Underlying()
		}
		members := map[string]expr.ValuePlanNode{}
		for _, member := range node.Members() {
			members[member.Name] = member.Node
		}
		require.NotContains(t, members, "thumb")
		require.Equal(t, wire, expr.JSONFieldName("kind", members["kind"].Attribute()), "shared members use the body's wire name")
		require.Equal(t, expr.ValueCodecJSON, members["blob"].Codec(), "shared members use the body's codec override")
		require.True(t, contributor.Attribute.IsRequired("kind"), "Parent keeps its own requiredness")
	}
	require.Len(t, keys, 2, "different wire contexts have different identities")
}
