package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/grpc/codegen/internal/transportir"
)

// reserveProtoServiceSymbols records declarations emitted by the pinned
// protoc-gen-go-grpc service generator in the protobuf package. The existing
// scope allocates inferred message names around them; message collection checks
// explicit names against the same inventory instead of overriding authored names.
func reserveProtoServiceSymbols(sd *ServiceData, service *transportir.Service) {
	name := sd.GoName
	symbols := map[string]string{
		name + "Client": "client interface",
		strings.ToLower(name[:1]) + name[1:] + "Client": "private client implementation",
		"New" + name + "Client":                         "client constructor",
		name + "Server":                                 "server interface",
		"Unimplemented" + name + "Server":               "unimplemented server",
		"Unsafe" + name + "Server":                      "unsafe server interface",
		"Register" + name + "Server":                    "server registration",
		name + "_ServiceDesc":                           "service descriptor",
	}
	for _, endpoint := range service.Endpoints {
		method := protoBufIdentifier(protoServiceName(sd.Service.Method(endpoint.Name).VarName), true, true)
		prefix := name + "_" + method
		symbols[prefix+"_FullMethodName"] = "full method name"
		symbols["_"+prefix+"_Handler"] = "method handler"
		if endpoint.Stream.IsStreaming {
			symbols[prefix+"Client"] = "client stream alias"
			symbols[prefix+"Server"] = "server stream alias"
		}
	}
	for symbol := range symbols {
		sd.Scope.Unique(symbol)
	}
	sd.protoServiceSymbols = symbols
}
