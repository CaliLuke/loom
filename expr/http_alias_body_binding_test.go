package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExplicitAliasBodyBinding(t *testing.T) {
	for _, response := range []bool{false, true} {
		for depth := range 3 {
			t.Run(fmt.Sprintf("response=%t/aliases=%d", response, depth), func(t *testing.T) {
				source := aliasBodySource("Service")
				body := aliasBodyDeclaration(depth)
				target := finalizedAliasBody(t, response, "one", source, body)
				context := NewValueContext()
				occurrence, err := context.NewOccurrence(source)
				require.NoError(t, err)
				plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
				require.NoError(t, err)
				resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"message": "sample"}}), ValueRoleExample)
				require.Equal(t, ValueResolved, resolved.Outcome())
				projected := context.ProjectJSON(resolved, plan)
				require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
				wire, present := projected.JSON()
				require.True(t, present)
				require.Equal(t, `{"message":"sample"}`, string(wire))
			})
		}
	}
}

func TestExplicitAliasBodyBindingOwnsOccurrences(t *testing.T) {
	for _, response := range []bool{false, true} {
		t.Run(fmt.Sprint(response), func(t *testing.T) {
			body := aliasBodyDeclaration(1)
			member := AsObject(body.Type).Attribute("message")
			firstSource, secondSource := aliasBodySource("First"), aliasBodySource("Second")
			first := finalizedAliasBody(t, response, "first", firstSource, body)
			second := finalizedAliasBody(t, response, "second", secondSource, body)
			require.Same(t, member, valueSemanticOrigin(member), "shared authored member must remain unbound")
			require.Same(t, body, valueSemanticOrigin(body), "shared authored root must remain unbound")
			require.NotContains(t, body.Type.(UserType).Attribute().Meta, "name:original", "transport naming must not mutate the declaration")
			require.Same(t, AsObject(firstSource.Type).Attribute("message"), valueSemanticOrigin(AsObject(first.Type).Attribute("message")))
			require.Same(t, AsObject(secondSource.Type).Attribute("message"), valueSemanticOrigin(AsObject(second.Type).Attribute("message")))
		})
	}
}

func TestExplicitAliasBodyBindingPreservesRecursiveDeclaration(t *testing.T) {
	node := &UserTypeExpr{TypeName: "Node", AttributeExpr: &AttributeExpr{}}
	node.Type = &Object{{Name: "child", Attribute: &AttributeExpr{Type: node}}}
	derived := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Status400", AttributeExpr: DupAtt(node.AttributeExpr)}}
	require.Same(t, node, node.Find("child").Type)
	require.NotSame(t, node, AsObject(derived.Type).Attribute("child").Type)
	finalizedAliasBody(t, true, "error", derived, &AttributeExpr{Type: node})
	require.Same(t, node, node.Find("child").Type, "transport binding must not replace the authored recursive back edge")
}

func aliasBodySource(name string) *AttributeExpr {
	return &AttributeExpr{Type: &UserTypeExpr{TypeName: name, AttributeExpr: &AttributeExpr{Type: &Object{
		{Name: "message", Attribute: &AttributeExpr{Type: String}},
	}}}}
}

func aliasBodyDeclaration(depth int) *AttributeExpr {
	body := aliasBodySource("AuthoredBody")
	for index := range depth {
		body = &AttributeExpr{Type: &UserTypeExpr{TypeName: fmt.Sprintf("BodyAlias%d", index), AttributeExpr: body}}
	}
	return body
}

func finalizedAliasBody(t *testing.T, response bool, name string, source, body *AttributeExpr) *AttributeExpr {
	t.Helper()
	if !response {
		return finalizeExplicitRequestBodyForTest(t, name, source, body).Body
	}
	service := &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "bindings"}}
	endpoint := &HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: name, Result: source, Service: service.ServiceExpr}, Service: service}
	// Body(namedType) creates a fresh outer occurrence but shares the named
	// declaration until transport preparation detaches it.
	occurrence := *body
	r := &HTTPResponseExpr{Body: &occurrence}
	r.finalizeBody(endpoint, source)
	return r.Body
}
