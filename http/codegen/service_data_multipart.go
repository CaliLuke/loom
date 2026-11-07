package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func (sds *ServicesData) initEndpointMultipartData(endpoint *EndpointData, endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data, scope *codegen.NameScope) {
	if endpointIR.Request.Multipart && !endpoint.Payload.Request.MultipartGenerated {
		endpoint.MultipartRequestDecoder = &MultipartData{
			FuncName:    httpMultipartSymbol(scope, svc.StructName, method.VarName, "DecoderFunc"),
			InitName:    httpMultipartSymbol(scope, svc.StructName, method.VarName, "Decoder"),
			VarName:     fmt.Sprintf("%s%sDecoderFn", svc.VarName, method.VarName),
			ServiceName: svc.Name,
			MethodName:  method.Name,
			Payload:     endpoint.Payload,
		}
		endpoint.MultipartRequestEncoder = &MultipartData{
			FuncName:    httpMultipartSymbol(scope, svc.StructName, method.VarName, "EncoderFunc"),
			InitName:    httpMultipartSymbol(scope, svc.StructName, method.VarName, "Encoder"),
			VarName:     fmt.Sprintf("%s%sEncoderFn", svc.VarName, method.VarName),
			ServiceName: svc.Name,
			MethodName:  method.Name,
			Payload:     endpoint.Payload,
		}
	}
}
