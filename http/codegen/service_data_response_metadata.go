package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func (sds *ServicesData) buildInformationalData(endpoint *transportir.Endpoint, sd *ServiceData) []*InformationalData {
	method := sd.Service.Method(endpoint.MethodName)
	data := make([]*InformationalData, 0, len(endpoint.Response.Informational))
	for _, info := range endpoint.Response.Informational {
		data = append(data, &InformationalData{
			Name:       info.Name,
			Send:       httpInformationalSymbol(sd.Scope, method.VarName, info.Name, "send"),
			Observe:    httpInformationalSymbol(sd.Scope, method.VarName, info.Name, "observe"),
			ObserveKey: httpInformationalSymbol(sd.Scope, method.VarName, info.Name, "observe-key"),
			Decode:     httpInformationalSymbol(sd.Scope, method.VarName, info.Name, "decode-info"),
			Response:   sds.buildSingleResponseData(endpoint, info.Response, info.Response.Carrier, false, method, nil, sd),
		})
	}
	return data
}

func responseViewedResult(method *service.MethodData, metadata bool) *service.ViewedResultTypeData {
	if metadata {
		return nil
	}
	return method.ViewedResult
}

// appendResponseStatusArg supplies the actual HTTP response status after the
// existing body/header/cookie/error-name inputs through their native field owner.
func appendResponseStatusArg(args []*InitArgData, response *transportir.ResponseStatus, carrier *expr.AttributeExpr, context *codegen.AttributeContext, sd *ServiceData) []*InitArgData {
	field := response.StatusField
	if field == nil {
		return args
	}
	vars := newResponseVarScope(sd)
	for _, arg := range args {
		vars.reserve(arg.VarName)
	}
	name, _ := vars.allocate("statusCode", nil)
	fieldName, fieldAttribute, pointer := transportFieldBinding(field.Name, field.Attribute, carrier, context)
	argument := &InitArgData{
		responseStatus: true,
		attribute:      &expr.AttributeExpr{Type: expr.Int},
		fieldAttribute: fieldAttribute,
		context:        context,
		Ref:            "resp.StatusCode",
		AttributeData: &AttributeData{
			Name: field.Name, VarName: name,
			FieldName: fieldName, FieldType: fieldAttribute.Type, FieldPointer: pointer,
			Type: expr.Int, TypeName: sd.Scope.GoTypeName(&expr.AttributeExpr{Type: expr.Int}),
			TypeRef: sd.Scope.GoTypeRef(&expr.AttributeExpr{Type: expr.Int}), Required: true,
		},
	}
	return append(args, argument)
}
