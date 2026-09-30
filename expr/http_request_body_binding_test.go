package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinalizeExplicitHTTPRequestBodyBindsIndependentMembers(t *testing.T) {
	minimum := 2
	bodyMember := &AttributeExpr{
		Type:        String,
		Description: "authored transport description",
		Docs:        &DocsExpr{Description: "authored transport docs", URL: "https://example.com/body"},
		Validation:  &ValidationExpr{MinLength: &minimum},
		UserExamples: []*ExampleExpr{{
			Summary: "authored transport example",
			Value:   "body",
		}},
		Meta: MetaExpr{"transport:annotation": {"body"}},
	}
	secondBodyMember := &AttributeExpr{Type: Int}
	authoredBody := &AttributeExpr{Type: &UserTypeExpr{
		TypeName: "EntityData",
		AttributeExpr: &AttributeExpr{Type: &Object{
			{Name: "name:body_name", Attribute: bodyMember},
			{Name: "age:body_age", Attribute: secondBodyMember},
		}},
	}}
	payloadMember := &AttributeExpr{Type: String}
	secondPayloadMember := &AttributeExpr{Type: Int}
	payload := &AttributeExpr{Type: &UserTypeExpr{
		TypeName: "Entity",
		AttributeExpr: &AttributeExpr{Type: &Object{
			{Name: "id", Attribute: &AttributeExpr{Type: String}},
			{Name: "name:payload_name", Attribute: payloadMember},
			{Name: "age:payload_age", Attribute: secondPayloadMember},
		}},
	}}

	otherPayloadMember := &AttributeExpr{Type: String}
	otherSecondPayloadMember := &AttributeExpr{Type: Int}
	secondPayload := &AttributeExpr{Type: &UserTypeExpr{
		TypeName: "Entity",
		AttributeExpr: &AttributeExpr{Type: &Object{
			{Name: "id", Attribute: &AttributeExpr{Type: String}},
			{Name: "name:payload_name", Attribute: otherPayloadMember},
			{Name: "age:payload_age", Attribute: otherSecondPayloadMember},
		}},
	}}
	first := finalizeExplicitRequestBodyForTest(t, "first", payload, authoredBody)
	second := finalizeExplicitRequestBodyForTest(t, "second", secondPayload, authoredBody)

	require.NotSame(t, authoredBody, first.Body)
	require.NotSame(t, first.Body, second.Body)
	firstMember := AsObject(first.Body.Type).Attribute("name:body_name")
	secondMember := AsObject(second.Body.Type).Attribute("name:body_name")
	firstSecondMember := AsObject(first.Body.Type).Attribute("age:body_age")
	secondSecondMember := AsObject(second.Body.Type).Attribute("age:body_age")
	require.NotSame(t, bodyMember, firstMember)
	require.NotSame(t, firstMember, secondMember)
	require.NotSame(t, secondBodyMember, firstSecondMember)
	require.NotSame(t, firstSecondMember, secondSecondMember)
	require.Same(t, payloadMember, valueSemanticOrigin(firstMember))
	require.Same(t, secondPayloadMember, valueSemanticOrigin(firstSecondMember))
	require.Same(t, otherPayloadMember, valueSemanticOrigin(secondMember))
	require.Same(t, otherSecondPayloadMember, valueSemanticOrigin(secondSecondMember))
	require.Same(t, authoredBody, valueSemanticOrigin(authoredBody), "finalization must not bind the shared authored body")
	require.Same(t, bodyMember, valueSemanticOrigin(bodyMember), "finalization must not bind the shared authored declaration")
	require.Same(t, secondBodyMember, valueSemanticOrigin(secondBodyMember), "finalization must not bind the shared authored declaration")
	require.Equal(t, bodyMember.Description, firstMember.Description)
	require.Equal(t, bodyMember.Docs, firstMember.Docs)
	require.Equal(t, bodyMember.Validation, firstMember.Validation)
	require.Len(t, firstMember.UserExamples, 1)
	require.Equal(t, bodyMember.UserExamples[0].Summary, firstMember.UserExamples[0].Summary)
	require.Equal(t, bodyMember.UserExamples[0].Value, firstMember.UserExamples[0].Value)
	require.Equal(t, bodyMember.Meta, firstMember.Meta)

	context := NewValueContext()
	occurrence, err := context.NewOccurrence(payload)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: first.Body,
		Codec:  ValueCodecJSON,
		Use:    ValuePlanDocumentation,
	})
	require.NoError(t, err)
	require.Equal(t, "EntityData", plan.Root().TargetDeclarationID())
	require.Equal(t, "body_name", plan.Root().Underlying().Members()[0].WireName)
}

func TestFinalizeExplicitHTTPRequestBodyBindsSelectedPayloadMembers(t *testing.T) {
	sourceMember := &AttributeExpr{Type: String}
	selected := &AttributeExpr{Type: &UserTypeExpr{
		TypeName: "SelectedPayload",
		AttributeExpr: &AttributeExpr{Type: &Object{
			{Name: "name", Attribute: sourceMember},
		}},
	}}
	payload := &AttributeExpr{Type: &Object{
		{Name: "selected", Attribute: selected},
		{Name: "metadata", Attribute: &AttributeExpr{Type: String}},
	}}
	authoredMember := &AttributeExpr{Type: String}
	authoredBody := &AttributeExpr{
		Type: &UserTypeExpr{
			TypeName: "SelectedBody",
			AttributeExpr: &AttributeExpr{Type: &Object{
				{Name: "name", Attribute: authoredMember},
			}},
		},
		Meta: MetaExpr{"origin:attribute": {"selected"}},
	}

	endpoint := finalizeExplicitRequestBodyForTest(t, "selected", payload, authoredBody)
	require.Same(t, selected, valueSemanticOrigin(endpoint.Body))
	require.Same(t, sourceMember, valueSemanticOrigin(AsObject(endpoint.Body.Type).Attribute("name")))

	context := NewValueContext()
	occurrence, err := context.NewOccurrence(payload)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target:    endpoint.Body,
		Selection: []string{"selected"},
		Codec:     ValueCodecJSON,
		Use:       ValuePlanDocumentation,
	})
	require.NoError(t, err)
	require.Equal(t, "SelectedBody", plan.Root().TargetDeclarationID())
}

func TestFinalizeExplicitHTTPRequestBodyLeavesUnrelatedMembersUnbound(t *testing.T) {
	payload := &AttributeExpr{Type: &Object{
		{Name: "name", Attribute: &AttributeExpr{Type: String}},
	}}
	authoredBody := &AttributeExpr{Type: &UserTypeExpr{
		TypeName: "UnrelatedBody",
		AttributeExpr: &AttributeExpr{Type: &Object{
			{Name: "name", Attribute: &AttributeExpr{Type: String}},
			{Name: "extra", Attribute: &AttributeExpr{Type: String}},
		}},
	}}
	endpoint := finalizeExplicitRequestBodyForTest(t, "unrelated", payload, authoredBody)

	context := NewValueContext()
	occurrence, err := context.NewOccurrence(payload)
	require.NoError(t, err)
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: endpoint.Body,
		Codec:  ValueCodecJSON,
		Use:    ValuePlanDocumentation,
	})
	require.ErrorContains(t, err, `value plan has unrelated member "extra"`)
}

func finalizeExplicitRequestBodyForTest(t *testing.T, name string, payload, body *AttributeExpr) *HTTPEndpointExpr {
	t.Helper()
	service := &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "bindings"}}
	endpoint := &HTTPEndpointExpr{
		MethodExpr: &MethodExpr{Name: name, Payload: payload, Service: service.ServiceExpr},
		Service:    service,
		Body:       body,
	}
	endpoint.initTransportAttributes()
	endpoint.finalizeTransportBodies()
	return endpoint
}
