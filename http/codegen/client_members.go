package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

// finalizeClientMembers allocates concrete receiver roles before their names
// are consumed by client rendering, CLI adapters or group package reservations.
func finalizeClientMembers(data *ServiceData) {
	scope := codegen.NewNameScope()
	for _, name := range fixedClientMembers(data) {
		scope.Unique(name)
	}
	for _, endpoint := range data.Endpoints {
		endpoint.EndpointInit = scope.Unique(endpoint.Method.VarName, "Endpoint")
	}
	jsonrpc := len(data.Endpoints) > 0 && data.Endpoints[0].valueTransport.IsJSONRPC
	for _, endpoint := range data.Endpoints {
		preferred := endpoint.RequestInit.Name
		endpoint.RequestInit.Name = scope.Unique(preferred, "Request")
		endpoint.RequestInit.Description = strings.Replace(endpoint.RequestInit.Description, preferred, endpoint.RequestInit.Name, 1)
		if !jsonrpc || IsSSEEndpoint(endpoint) {
			endpoint.ClientDoer = scope.Unique(endpoint.Method.VarName+"Doer", "Doer")
		}
		if !jsonrpc && endpoint.HasMixedResults {
			endpoint.StreamEndpointInit = scope.Unique(endpoint.Method.VarName+"Stream", "Stream")
		}
	}
	if !jsonrpc {
		data.clientGroups = buildClientOperationGroups(data, scope)
	}
}

// fixedClientMembers mirrors fields and methods emitted on the concrete Client.
// Canonical methods are independent of conditional transport lifecycle members.
func fixedClientMembers(data *ServiceData) []string {
	names := []string{"RestoreResponseBody", "scheme", "host", "encoder", "decoder"}
	if len(data.Endpoints) > 0 && data.Endpoints[0].valueTransport.IsJSONRPC {
		names = append(names, "Doer")
		if HasWebSocket(data) {
			names = append(names, "dialer", "webSocketConfig", "connMu", "conn", "closed", "streamConfig", "getConn", "Close", "IsClosed")
		}
	} else if HasWebSocket(data) {
		names = append(names, "dialer", "configurer")
	}
	return names
}
