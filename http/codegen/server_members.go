package codegen

import (
	"go/token"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// finalizeServerMembers assigns identities in the concrete server namespace,
// separate from service methods and package declarations. WebSocket roles are
// copied into the service stream, so both receiving namespaces participate.
func finalizeServerMembers(data *ServiceData) {
	scope := codegen.NewNameScope()
	for _, name := range fixedServerMembers(data) {
		scope.Unique(name)
	}
	jsonrpc := len(data.Endpoints) > 0 && data.Endpoints[0].valueTransport.IsJSONRPC
	for _, endpoint := range data.Endpoints {
		preferred := endpoint.Method.VarName
		if jsonrpc && IsWebSocketEndpoint(endpoint) {
			_, size := utf8.DecodeRuneInString(preferred)
			preferred = strings.ToLower(preferred[:size]) + preferred[size:]
			if !token.IsIdentifier(preferred) {
				preferred = codegen.Goify(preferred, false)
			}
		}
		endpoint.ServerHandler = scope.Unique(preferred, "Handler")
		if jsonrpc && IsWebSocketEndpoint(endpoint) && endpoint.Method.ServerStream != nil &&
			(endpoint.Method.ServerStream.Kind == expr.ServerStreamKind || endpoint.Method.ServerStream.Kind == expr.BidirectionalStreamKind) {
			endpoint.ServerEndpoint = scope.Unique(preferred+"Endpoint", "Endpoint")
		}
	}
	for _, fs := range data.FileServers {
		fs.VarName = scope.Unique(fs.VarName)
	}
}

// fixedServerMembers mirrors concrete receiver declarations, including methods.
// Endpoint and file fields are allocated afterward; canonical method names are
// never changed to accommodate a transport receiver.
func fixedServerMembers(data *ServiceData) []string {
	names := []string{data.ServerService, data.MountServer, "Use", "MethodNames"}
	if data.CORS != nil && data.CORS.Runtime {
		names = append(names, "corsPolicy")
	}
	if len(data.Endpoints) == 0 || !data.Endpoints[0].valueTransport.IsJSONRPC {
		return append(names, "Mounts")
	}
	names = append(names, "Handler", "Methods", "ServeHTTP", "decoder", "encoder", "errhandler")
	if HasWebSocket(data) {
		names = append(names, "StreamHandler", "upgrader", "webSocketConfig", "streamWritePolicy", "serveHTTP",
			"cancel", "r", "w", "conn", "Recv", "Close", "SendError", "send", "sendError", "matchesMethod", "processRequest")
		for _, endpoint := range data.Endpoints {
			if endpoint.Result == nil || endpoint.Result.Ref == "" {
				continue
			}
			names = append(names, "Send"+endpoint.Method.VarName+"Notification", "Send"+endpoint.Method.VarName+"Response")
		}
		return names
	}
	hasSSE, hasHTTP := false, false
	for _, endpoint := range data.Endpoints {
		hasSSE = hasSSE || (endpoint.Method.ServerStream != nil && IsSSEEndpoint(endpoint))
		hasHTTP = hasHTTP || endpoint.Method.ServerStream == nil
	}
	names = append(names, "encodeJSONRPCError")
	if hasSSE {
		names = append(names, "sseHandlerSpec", "dispatchSSE", "sendSSEError", "streamWritePolicy")
		if !hasHTTP {
			names = append(names, "handleSSE")
		}
	}
	if !hasSSE || hasHTTP {
		names = append(names, "serveHTTP", "handleHTTP", "httpHandlerSpec", "dispatchHTTP")
	}
	return names
}
