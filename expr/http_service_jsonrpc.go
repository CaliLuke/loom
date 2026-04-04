package expr

import "github.com/CaliLuke/loom/eval"

// validateTransports validates transport compatibility and JSON-RPC constraints
func (svc *HTTPServiceExpr) validateTransports(verr *eval.ValidationErrors) {
	var (
		hasPureHTTPWebSocket bool
		hasJSONRPCWebSocket  bool
	)
	for _, e := range svc.HTTPEndpoints {
		usesWebSocket := e.MethodExpr.IsStreaming() && e.SSE == nil
		if e.IsJSONRPC() {
			if usesWebSocket {
				hasJSONRPCWebSocket = true
			}
		} else if usesWebSocket {
			hasPureHTTPWebSocket = true
		}
	}
	if hasJSONRPCWebSocket && hasPureHTTPWebSocket {
		verr.Add(svc, "Service cannot mix JSON-RPC WebSocket endpoints with pure HTTP WebSocket endpoints. JSON-RPC uses a single WebSocket connection for all methods, while pure HTTP WebSocket creates individual connections per endpoint.")
	}
	if hasJSONRPCWebSocket {
		svc.validateJSONRPCWebSocketConstraints(verr)
	}
	if svc.ServiceExpr.Meta != nil && svc.ServiceExpr.Meta["jsonrpc:service"] != nil {
		svc.validateJSONRPCTransportConsistency(verr)
		svc.validateJSONRPCRoutes(verr)
	}
}

// validateJSONRPCWebSocketConstraints validates constraints for JSON-RPC WebSocket endpoints
func (svc *HTTPServiceExpr) validateJSONRPCWebSocketConstraints(verr *eval.ValidationErrors) {
	for _, e := range svc.HTTPEndpoints {
		name := e.MethodExpr.Name
		if !e.Headers.IsEmpty() {
			verr.Add(e, "JSON-RPC endpoint %q using WebSocket cannot have header mappings", name)
		}
		if !e.Cookies.IsEmpty() {
			verr.Add(e, "JSON-RPC endpoint %q using WebSocket cannot have cookie mappings", name)
		}
		if !e.Params.IsEmpty() {
			verr.Add(e, "JSON-RPC endpoint %q using WebSocket cannot have parameter mappings", name)
		}
	}
}

// prepareJSONRPCRoutes creates routes for all JSON-RPC endpoints.
// All JSON-RPC methods share the same route.
func (svc *HTTPServiceExpr) prepareJSONRPCRoutes() {
	hasJSONRPC := false
	for _, e := range svc.HTTPEndpoints {
		if e.IsJSONRPC() {
			hasJSONRPC = true
			break
		}
	}
	if !hasJSONRPC {
		return
	}

	var route *RouteExpr
	if svc.JSONRPCRoute != nil {
		route = svc.JSONRPCRoute
	} else {
		path := "/"
		if len(svc.Paths) > 0 {
			path = svc.Paths[0]
		}
		method := "POST"
		for _, e := range svc.HTTPEndpoints {
			if e.IsJSONRPC() && e.MethodExpr.IsStreaming() && e.SSE == nil {
				method = "GET"
				break
			}
		}
		route = &RouteExpr{Method: method, Path: path}
	}

	for _, e := range svc.HTTPEndpoints {
		if e.IsJSONRPC() {
			e.Routes = []*RouteExpr{{
				Method:   route.Method,
				Path:     route.Path,
				Endpoint: e,
			}}
		}
	}
}

// validateJSONRPCTransportConsistency validates JSON-RPC transport combinations.
// WebSocket cannot be mixed with other transports, but HTTP and SSE can coexist.
func (svc *HTTPServiceExpr) validateJSONRPCTransportConsistency(verr *eval.ValidationErrors) {
	var hasWebSocket, hasSSE, hasRegular bool
	for _, e := range svc.HTTPEndpoints {
		if e.IsJSONRPC() {
			if e.MethodExpr.IsStreaming() {
				if e.SSE != nil {
					hasSSE = true
				} else {
					hasWebSocket = true
				}
			} else {
				hasRegular = true
			}
		}
	}
	if hasWebSocket && (hasSSE || hasRegular) {
		verr.Add(svc, "JSON-RPC service %q cannot mix WebSocket with other transports (SSE or regular HTTP). WebSocket requires a single persistent connection for all methods.", svc.Name())
	}
}

// validateJSONRPCRoutes validates that JSON-RPC routes use the correct HTTP method.
func (svc *HTTPServiceExpr) validateJSONRPCRoutes(verr *eval.ValidationErrors) {
	for _, e := range svc.HTTPEndpoints {
		if e.IsJSONRPC() {
			for _, r := range e.Routes {
				if e.MethodExpr.IsStreaming() && e.SSE == nil {
					if r.Method != "GET" {
						verr.Add(r, "JSON-RPC WebSocket endpoint must use GET method, got %q", r.Method)
					}
				} else if r.Method != "POST" {
					verr.Add(r, "JSON-RPC endpoint must use POST method, got %q", r.Method)
				}
			}
		}
	}
}
