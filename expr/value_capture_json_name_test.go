package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestValueCaptureKeepsOccurrenceJSONName captures a member that composition
// refined: the captured copy keeps the constraint-local contract but takes its
// JSON field name from the occurrence, as a transport normalizes it.
func TestValueCaptureKeepsOccurrenceJSONName(t *testing.T) {
	root := expr.RunDSL(t, func() {
		parent := Type("Parent", func() {
			Attribute("kind", String)
		})
		Type("Child", func() {
			Extend(parent)
			Attribute("kind", String, func() {
				Meta("struct:tag:form", "k")
			})
			Attribute("thumb", Bytes)
		})
	})
	occurrence := expr.DupAtt(&expr.AttributeExpr{Type: root.UserType("Child")})
	object := expr.AsObject(occurrence.Type)
	for _, name := range []string{"kind", "thumb"} {
		member := object.Attribute(name)
		member.Meta = expr.MetaExpr{"struct:tag:json:name": {"wire_" + name}}
	}
	context := expr.NewValueContext()
	captured, err := context.NewOccurrence(occurrence)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(captured, expr.ValuePlanRequest{TargetOccurrence: captured, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
	require.NoError(t, err)
	node := plan.Root()
	for node.Underlying().Valid() {
		node = node.Underlying()
	}
	names := map[string]string{}
	for _, member := range node.Members() {
		names[member.Name] = expr.JSONFieldName(member.Name, member.Node.Attribute())
	}
	require.Equal(t, "wire_kind", names["kind"], "a refined member keeps its occurrence name")
	require.Equal(t, "wire_thumb", names["thumb"])
	require.True(t, expr.AsObject(root.UserType("Child").Attribute().Type).Attribute("kind").Meta["struct:tag:json:name"] == nil,
		"the declaration is unchanged")
}
