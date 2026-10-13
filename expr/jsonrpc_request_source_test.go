package expr

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONRPCRequestSourceIdentity(t *testing.T) {
	for _, kind := range []StreamKind{NoStreamKind, ServerStreamKind, ClientStreamKind, BidirectionalStreamKind} {
		t.Run(strconv.Itoa(int(kind)), func(t *testing.T) {
			payload := &AttributeExpr{Type: &Object{{Name: "request", Attribute: &AttributeExpr{Type: String}}}}
			streaming := &AttributeExpr{Type: &Object{{Name: "message", Attribute: &AttributeExpr{Type: String}}}}
			endpoint := &HTTPEndpointExpr{MethodExpr: &MethodExpr{Payload: payload, StreamingPayload: streaming, Stream: kind, Meta: MetaExpr{"jsonrpc": {}}}}
			want := payload
			if kind == ClientStreamKind || kind == BidirectionalStreamKind {
				want = streaming
			}
			require.Same(t, want, endpoint.jsonrpcRequestPayload())
			require.Same(t, payload, endpoint.MethodExpr.Payload)
			require.Same(t, streaming, endpoint.MethodExpr.StreamingPayload)
		})
	}
}

func TestJSONRPCRequestSourceNormalization(t *testing.T) {
	payload := &AttributeExpr{Type: &Object{{Name: "token", Attribute: &AttributeExpr{Type: String}}}}
	endpoint := &HTTPEndpointExpr{
		MethodExpr: &MethodExpr{
			Payload: payload, StreamingPayload: &AttributeExpr{Type: Empty}, Result: &AttributeExpr{Type: String},
			Stream: ServerStreamKind, Meta: MetaExpr{"jsonrpc": {}},
		},
		StreamingBody: payload,
	}
	require.Same(t, payload, endpoint.jsonrpcRequestPayload())
	endpoint.normalizeJSONRPCServerStreamingPayload()
	require.Same(t, payload, endpoint.jsonrpcRequestPayload())
	endpoint.finalizeJSONRPCBodyState()
	require.Same(t, payload, endpoint.MethodExpr.Payload)
	require.Same(t, payload, endpoint.Body)
	endpoint.normalizeJSONRPCServerStreamingPayload()
	endpoint.finalizeJSONRPCBodyState()
	require.Same(t, payload, endpoint.jsonrpcRequestPayload())
}

func TestHTTPWireBindingCredentialRoleMembership(t *testing.T) {
	plan := &httpWireBindingPlan{}
	jwt := httpCredentialRole{tag: "security:token", source: "token"}
	key := httpCredentialRole{tag: "security:apikey:key", source: "token"}
	for _, role := range []httpCredentialRole{jwt, key, jwt} {
		plan.add(&httpWireBinding{location: "body", name: "token", sourceKind: "payload", sources: []string{"token"}, codec: "raw", roles: []httpCredentialRole{role}})
	}
	require.Len(t, plan.bindings, 1)
	require.Equal(t, []httpCredentialRole{jwt, key}, plan.bindings[0].roles)
}
