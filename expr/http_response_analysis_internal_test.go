package expr

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
)

func TestResponseAnalysisFullKeys(t *testing.T) {
	final := []HTTPStatusSelector{{Code: 200}, {Class: Status2XX}, {Code: 404}, {Code: 404}, {Class: StatusFallback}}
	e := &HTTPEndpointExpr{
		Responses: []*HTTPResponseExpr{{Status: final[0]}, {Status: final[1]}},
		HTTPErrors: []*HTTPErrorExpr{
			{Response: &HTTPResponseExpr{Status: final[2]}},
			{Response: &HTTPResponseExpr{Status: final[3]}},
			{Response: &HTTPResponseExpr{Status: final[4]}},
			{Response: &HTTPResponseExpr{JSONRPCError: &JSONRPCErrorMapping{Code: 4403}}},
		},
	}
	require.Equal(t, final, e.FinalStatusKeys())
	keys := e.FinalStatusKeys()
	keys[0] = HTTPStatusSelector{Code: 999}
	require.Equal(t, final, e.FinalStatusKeys())
	e.Responses = nil
	require.Equal(t, final[2:], e.FinalStatusKeys())
}

func TestResponseAuthoredBodyCapture(t *testing.T) {
	for _, shape := range []string{"implicit", "inline", "named", "selected", "Empty"} {
		t.Run(shape, func(t *testing.T) {
			source := &AttributeExpr{Type: &Object{
				&NamedAttributeExpr{Name: "value", Attribute: &AttributeExpr{Type: String}},
			}}
			var authored *AttributeExpr
			switch shape {
			case "inline", "named":
				authored = DupAtt(source)
				authored.Description = "local body"
				authored.UserExamples = []*ExampleExpr{{Summary: "authored", Value: map[string]any{"value": "kept"}}}
				if shape == "named" {
					authored.Type = &UserTypeExpr{TypeName: "Body", UID: "Body", AttributeExpr: DupAtt(source)}
				}
			case "selected":
				authored = &AttributeExpr{Type: String, Meta: MetaExpr{"origin:attribute": {"value"}}, Description: "local selection"}
			case "Empty":
				authored = &AttributeExpr{Type: Empty}
			}
			r := &HTTPResponseExpr{Status: HTTPStatusSelector{Code: 200}, Body: authored, Headers: NewEmptyMappedAttributeExpr()}
			e := &HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: "read", Result: source}, Service: &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "capture"}}, Responses: []*HTTPResponseExpr{r}}
			require.Nil(t, r.AuthoredBody())
			require.Nil(t, r.StatusDomain())
			e.captureResponseBindings()
			require.Same(t, authored, r.AuthoredBody())
			duplicate := r.Dup()
			require.Nil(t, duplicate.AuthoredBody())
			duplicate.captureBindings(source)
			require.Same(t, duplicate.Body, duplicate.AuthoredBody())
			if authored != nil {
				before := DupAtt(authored)
				r.Finalize(e, source)
				require.Same(t, authored, r.AuthoredBody())
				// Compare equivalent detached snapshots: DupAtt itself records origins.
				require.Equal(t, before, DupAtt(authored))
				if shape != "Empty" {
					require.NotSame(t, authored, r.Body)
				}
			}
			if authored == nil {
				r.Finalize(e, source)
			}
			r.Body = httpResponseBody(e, r)
			require.Same(t, authored, r.AuthoredBody())
			if shape == "implicit" {
				require.NotNil(t, r.Body)
				require.Nil(t, r.AuthoredBody())
			}
		})
	}
}

func TestResponseDomainCaptureBeforeLowering(t *testing.T) {
	source := &AttributeExpr{Type: &Object{
		&NamedAttributeExpr{Name: "status", Attribute: &AttributeExpr{Type: Int}},
		&NamedAttributeExpr{Name: "value", Attribute: &AttributeExpr{Type: String}},
	}, Validation: &ValidationExpr{Required: []string{"status", "value"}}}
	r := &HTTPResponseExpr{Status: HTTPStatusSelector{Class: Status2XX}, StatusField: "status", Headers: NewEmptyMappedAttributeExpr()}
	e := &HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: "read", Result: source}, Service: &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "capture"}}, Responses: []*HTTPResponseExpr{r}}
	e.captureResponseBindings()
	verr := new(eval.ValidationErrors)
	e.validateResponseSelection(verr)
	require.Empty(t, verr.Errors)
	before := r.StatusDomain()
	require.Len(t, before, 99)
	require.NotContains(t, before, 204)
	r.Finalize(e, source)
	r.Body = httpResponseBody(e, r)
	require.Equal(t, before, r.StatusDomain())
	require.Contains(t, r.RequiredFields(""), "value")
	require.True(t, source.IsRequired("value"))
	// A later transport-only replacement cannot recapture authored ownership.
	r.Body = &AttributeExpr{Type: Empty}
	r.captureStatusDomain(e, source)
	require.Equal(t, before, r.StatusDomain())
}
