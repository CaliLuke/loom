package expr

import "github.com/CaliLuke/loom/eval"

// jsonrpcRequestPayload selects the message source used by both admission and
// finalization. Server-stream normalization moves this same attribute into
// StreamingPayload without changing its identity.
func (e *HTTPEndpointExpr) jsonrpcRequestPayload() *AttributeExpr {
	if e.MethodExpr.IsPayloadStreaming() {
		return e.MethodExpr.StreamingPayload
	}
	return e.MethodExpr.Payload
}

func (e *HTTPEndpointExpr) validateJSONRPCWebSocketCredentials(verr *eval.ValidationErrors) {
	if !e.MethodExpr.IsStreaming() || e.SSE != nil {
		return
	}
	request := e.jsonrpcRequestPayload()
	for _, binding := range e.ensureWireBindings().bindings {
		if len(binding.schemes) == 0 {
			continue
		}
		if binding.location != "body" {
			verr.Add(e, "JSON-RPC WebSocket credential binding for %s uses unsupported %s %q; credentials must be carried in the message body", binding.description(), binding.location, binding.name)
			continue
		}
		for _, role := range binding.roles {
			if request == nil || TaggedAttribute(request, role.tag) != role.source {
				verr.Add(e, "JSON-RPC WebSocket credential %q must retain role %q in the effective request payload", role.source, role.tag)
				continue
			}
			_, attribute := request.FindAttribute(role.source)
			if attribute == nil || !validCredentialType(attribute) {
				verr.Add(e, "JSON-RPC WebSocket credential %q in the effective request payload must use String or a named string type without nullability or a custom Go type", role.source)
			}
		}
	}
}
