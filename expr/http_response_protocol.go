package expr

import "github.com/CaliLuke/loom/eval"

// IsJSONRPCScope reports whether parent is a JSON-RPC declaration scope.
// Service scopes are resolved within the active DSL root; endpoint scopes retain
// their own protocol provenance, including before attachment to a root.
func IsJSONRPCScope(parent eval.Expression) bool {
	switch parent := parent.(type) {
	case *JSONRPCExpr:
		return true
	case *HTTPServiceExpr:
		return parent.Root == &Root.API.JSONRPC.HTTPExpr
	case *HTTPEndpointExpr:
		return parent.IsJSONRPC()
	default:
		return false
	}
}

func (r *HTTPResponseExpr) validateProtocol(rpcError bool, errors *eval.ValidationErrors) {
	if rpcError {
		if r.JSONRPCError == nil || r.Status != (HTTPStatusSelector{}) {
			errors.Add(r, "JSON-RPC error response requires a JSON-RPC code and no HTTP status")
		}
		return
	}
	if r.JSONRPCError != nil {
		errors.Add(r, "HTTP response cannot define a JSON-RPC error code")
	}
	r.validateFinalPhase(errors)
}
