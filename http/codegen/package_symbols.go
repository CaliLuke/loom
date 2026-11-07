package codegen

import (
	"strconv"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/unionnames"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type httpPackageSymbol string

func (s httpPackageSymbol) Hash() string {
	return "http-package-symbol:" + string(s)
}

// preparePackageSymbols establishes declaration ownership before any transport
// reference is cached. The server and client share nominal transport identities;
// receiver members remain in their independent scopes.
func (sd *ServiceData) preparePackageSymbols(endpoints []*transportir.Endpoint) {
	for _, name := range []string{sd.ServerStruct, sd.MountPointStruct, sd.ServerInit, sd.MountServer, sd.ClientStruct, "New" + sd.ClientStruct} {
		sd.Scope.Unique(name)
	}
	hasContracts, hasWebSocket := false, false
	for _, endpoint := range endpoints {
		method := sd.Service.Method(endpoint.MethodName)
		for _, role := range []string{"handler", "mount", "decode-request", "encode-response", "encode-error", "encode-request", "decode-response", "contracts"} {
			httpEndpointSymbol(sd.Scope, method.VarName, role)
		}
		cases, _ := buildResponseContractCaseData(endpoint)
		hasContracts = hasContracts || len(cases) > 0
		for i := range endpoint.Routes {
			httpPathSymbol(sd.Scope, method.VarName, sd.Service.StructName, i)
		}
		if endpoint.Stream != nil && endpoint.Stream.IsStreaming {
			if method.ServerStream != nil {
				sd.Scope.Unique(method.ServerStream.VarName)
			}
			if method.ClientStream != nil {
				sd.Scope.Unique(method.ClientStream.VarName)
			}
			if endpoint.Stream.IsSSE && !endpoint.IsJSONRPC {
				sd.Scope.Unique(method.VarName + "ClientStream")
				sd.Scope.Unique(method.VarName + "StreamImpl")
				sd.Scope.Unique("New" + method.VarName + "Stream")
			}
			hasWebSocket = hasWebSocket || endpoint.Stream.IsWebSocket && !endpoint.IsJSONRPC
		}
		if endpoint.Request.Multipart && (endpoint.Request.MultipartEncoding == nil || endpoint.Request.MultipartEncoding.Custom) {
			for _, role := range []string{"DecoderFunc", "EncoderFunc", "Decoder", "Encoder"} {
				httpMultipartSymbol(sd.Scope, sd.Service.StructName, method.VarName, role)
			}
		}
	}
	if hasContracts {
		sd.Scope.Unique("ResponseContractCases")
	}
	if hasWebSocket {
		sd.Scope.Unique("ConnConfigurer")
		sd.Scope.Unique("NewConnConfigurer")
	}
	sd.Scope.SetTypeFamilies(func(typ expr.DataType, name string) []string {
		if union := expr.AsUnion(typ); union != nil {
			return unionnames.PackageDeclarations(union, name).Names
		}
		return []string{"Validate" + name}
	})
}

func httpEndpointSymbol(scope *codegen.NameScope, method, role string) string {
	var name string
	switch role {
	case "handler":
		name = "New" + method + "Handler"
	case "mount":
		name = "Mount" + method + "Handler"
	case "decode-request":
		name = "Decode" + method + "Request"
	case "encode-response":
		name = "Encode" + method + "Response"
	case "encode-error":
		name = "Encode" + method + "Error"
	case "encode-request":
		name = "Encode" + method + "Request"
	case "decode-response":
		name = "Decode" + method + "Response"
	case "contracts":
		name = method + "ResponseContractCases"
	}
	return scope.HashedUnique(httpPackageSymbol("endpoint:"+method+":"+role), name)
}

func httpPathSymbol(scope *codegen.NameScope, method, service string, index int) string {
	suffix := ""
	if index > 0 {
		suffix = strconv.Itoa(index + 1)
	}
	return scope.HashedUnique(httpPackageSymbol("path:"+method+":"+suffix), method+service+"Path"+suffix)
}

func httpMultipartSymbol(scope *codegen.NameScope, service, method, role string) string {
	name := service + method + role
	if role == "Decoder" || role == "Encoder" {
		name = "New" + name
	}
	return scope.HashedUnique(httpPackageSymbol("multipart:"+method+":"+role), name)
}

func endpointRequestEncoderSymbol(scope *codegen.NameScope, method *service.MethodData, payload *PayloadData, basicScheme *service.SchemeData) string {
	if payload.Request.JSONRPCID == nil && payload.Request.ClientBody == nil &&
		len(payload.Request.Headers) == 0 && len(payload.Request.QueryParams) == 0 &&
		len(payload.Request.Cookies) == 0 && basicScheme == nil {
		return ""
	}
	return httpEndpointSymbol(scope, method.VarName, "encode-request")
}
