package service

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// methodSymbols records declarations before nominal types occupy the package
// scope. Method, client-member and transport-adapter names have separate owners.
type (
	methodSymbols struct {
		name, endpoint, event                    string
		serverInterface, clientInterface, input  string
		serverAdapter, clientAdapter             string
		serverWrapper                            string
		clientInterception                       *clientInterception
		serverWrappedStream, clientWrappedStream string
		transport                                MethodTransportData
		security                                 MethodSecurityData
		interceptors                             map[string]*interceptorSymbols
	}

	interceptorSymbols struct {
		server                                             string
		payload, result, streamingPayload, streamingResult string
	}
)

const (
	apiNameSymbol            = "APIName"
	apiVersionSymbol         = "APIVersion"
	serviceNameSymbol        = "ServiceName"
	methodNamesSymbol        = "MethodNames"
	authorizerSymbol         = "Authorizer"
	streamSymbol             = "Stream"
	eventSymbol              = "Event"
	serverInterceptorsSymbol = "ServerInterceptors"
	clientInterceptorsSymbol = "ClientInterceptors"
	newClientSymbol          = "New" + clientStructName
	newEndpointsSymbol       = "New" + endpointsStructName
)

func (d *ServicesData) serviceSymbols(svc *expr.ServiceExpr, scope *codegen.NameScope) map[string]*methodSymbols {
	methods := make(map[string]*methodSymbols, len(svc.Methods))
	var secure, stream, websocket, event bool
	for _, m := range svc.Methods {
		n := &methodSymbols{
			clientInterception: newClientInterception(m),
			transport:          d.buildMethodTransportData(m),
			security:           buildMethodSecurityData(m, nil, nil),
			interceptors:       make(map[string]*interceptorSymbols),
		}
		methods[m.Name] = n
		secure = secure || len(n.security.Schemes) > 0
		stream = stream || n.transport.IsJSONRPC && (m.IsStreaming() || m.HasMixedResults())
		websocket = websocket || n.transport.IsJSONRPCWebSocket
		event = event || n.transport.IsJSONRPCSSE && m.Result.Type != expr.Empty
	}
	for _, name := range []string{serviceInterfaceName, clientStructName, endpointsStructName, newClientSymbol, newEndpointsSymbol, apiNameSymbol, apiVersionSymbol, serviceNameSymbol, methodNamesSymbol} {
		scope.Unique(name)
	}
	for _, item := range []struct {
		name    string
		emitted bool
	}{
		{authorizerSymbol, secure}, {streamSymbol, stream}, {eventSymbol, event && !websocket},
		{serverInterceptorsSymbol, len(d.serviceInterceptors(svc, true)) > 0},
		{clientInterceptorsSymbol, len(d.serviceInterceptors(svc, false)) > 0},
	} {
		if item.emitted {
			scope.Unique(item.name)
		}
	}
	members := codegen.NewNameScope()
	members.Unique("Use")
	if websocket {
		members.Unique("HandleStream")
	}
	adapters := codegen.NewNameScope()
	for _, m := range svc.Methods {
		n := methods[m.Name]
		n.name = members.Unique(codegen.Goify(m.Name, true), "Endpoint")
	}
	for _, m := range svc.Methods {
		methods[m.Name].allocatePackageSymbols(m, scope, adapters)
	}
	return methods
}

func (n *methodSymbols) allocatePackageSymbols(m *expr.MethodExpr, scope, adapters *codegen.NameScope) {
	n.endpoint = scope.Unique("New" + n.name + "Endpoint")
	if n.transport.SkipRequestBodyEncodeDecode {
		n.transport.RequestStruct = scope.Unique(n.name + "RequestData")
	}
	if n.transport.SkipResponseBodyEncodeDecode || n.transport.FileResponse {
		n.transport.ResponseStruct = scope.Unique(n.name + "ResponseData")
	}
	if n.transport.FileResponse {
		n.transport.FileResponseStruct = scope.Unique(n.name + "FileResponseData")
	}
	if m.IsStreaming() || m.HasMixedResults() {
		n.serverInterface = scope.Unique(n.name + "ServerStream")
		n.clientInterface = scope.Unique(n.name + "ClientStream")
		if !(n.transport.IsJSONRPCWebSocket && m.Stream == expr.ClientStreamKind) {
			n.input = scope.Unique(n.name + "EndpointInput")
		}
		n.serverAdapter = adapters.Unique(codegen.Goify(m.Name, true) + "ServerStream")
		n.clientAdapter = adapters.Unique(codegen.Goify(m.Name, true) + "ClientStream")
		if n.transport.IsJSONRPCSSE {
			n.event = scope.Unique(n.name + "Event")
		}
	}
	for _, server := range []bool{true, false} {
		ints := m.ServerInterceptors
		if !server {
			ints = m.ClientInterceptors
		}
		if len(ints) == 0 {
			continue
		}
		if server {
			n.serverWrapper = scope.Unique("Wrap" + n.name + "Endpoint")
		} else {
			n.clientInterception.allocateEndpoints(scope, n.name)
		}
		for _, in := range ints {
			n.allocateInterceptor(in, scope, server)
		}
	}
}

func (n *methodSymbols) allocateInterceptor(in *expr.InterceptorExpr, scope *codegen.NameScope, server bool) {
	name := codegen.Goify(in.Name, true)
	s := n.interceptors[name]
	if s == nil {
		s = &interceptorSymbols{}
		n.interceptors[name] = s
		for _, access := range []struct {
			enabled bool
			suffix  string
			dest    *string
		}{
			{in.ReadPayload != nil || in.WritePayload != nil, "Payload", &s.payload},
			{in.ReadResult != nil || in.WriteResult != nil, "Result", &s.result},
			{in.ReadStreamingPayload != nil || in.WriteStreamingPayload != nil, "StreamingPayload", &s.streamingPayload},
			{in.ReadStreamingResult != nil || in.WriteStreamingResult != nil, "StreamingResult", &s.streamingResult},
		} {
			if access.enabled {
				*access.dest = scope.Unique(codegen.Goify(in.Name, false) + n.name + access.suffix)
			}
		}
	}
	if server {
		s.server = scope.Unique("wrap" + n.name + name)
	} else {
		n.clientInterception.allocateBindings(scope, n.name, name)
	}
	if s.streamingPayload == "" && s.streamingResult == "" {
		return
	}
	if server && n.serverWrappedStream == "" {
		n.serverWrappedStream = scope.Unique("wrapped" + n.serverInterface)
	}
	if !server && n.clientWrappedStream == "" {
		n.clientWrappedStream = scope.Unique("wrapped" + n.clientInterface)
	}
}
