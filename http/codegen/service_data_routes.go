package codegen

import (
	"fmt"
	"strconv"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func (sds *ServicesData) buildEndpointRoutes(endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data, sd *ServiceData) []*RouteData {
	routes := make([]*RouteData, 0, len(endpointIR.Routes))
	for index, route := range endpointIR.Routes {
		routes = append(routes, &RouteData{
			Verb:     route.Method,
			Path:     route.Path,
			PathInit: sds.buildPathInitData(endpointIR, method, svc, sd, route.Path, index),
		})
	}
	return routes
}

func (sds *ServicesData) buildPathInitData(endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data, sd *ServiceData, path string, pathCount int) *InitData {
	params := expr.ExtractHTTPWildcards(path)
	initArgs := make([]*InitArgData, len(params))
	pathParamsObj := pathParametersObject(endpointIR.Request.PathParams)
	suffix := ""
	if pathCount > 0 {
		suffix = strconv.Itoa(pathCount + 1)
	}
	name := fmt.Sprintf("%s%sPath%s", method.VarName, svc.StructName, suffix)
	for j, arg := range params {
		patt := parameterAttributeByName(endpointIR.Request.PathParams, arg)
		att := makeHTTPType(patt)
		pointer := parameterPrimitivePointerByName(endpointIR.Request.PathParams, arg)
		if payloadPointer := payloadPrimitivePointerByName(endpointIR.Request.Payload, arg); payloadPointer {
			pointer = true
		}
		varName := sd.Scope.Name(codegen.Goify(arg, false))
		validate := ""
		if att.Validation != nil {
			ctx := httpContext(sd.Scope, true, false)
			validate = codegen.AttributeValidationCode(att, nil, ctx, true, expr.IsAlias(att.Type), varName, arg)
		}
		initArgs[j] = &InitArgData{
			Ref: varName,
			AttributeData: &AttributeData{
				Name:        arg,
				VarName:     varName,
				Description: att.Description,
				FieldName:   codegen.Goify(arg, true),
				FieldType:   patt.Type,
				TypeName:    sd.Scope.GoTypeName(att),
				TypeRef:     sd.Scope.GoTypeRef(att),
				Type:        att.Type,
				Pointer:     pointer,
				Required:    true,
				Example:     att.Example(sds.Root.API.ExampleGenerator),
				Validate:    validate,
			},
		}
	}
	code := renderPathInitCode(initArgs, pathParamsObj, expr.HTTPWildcardRegex.ReplaceAllString(path, "/%v"))
	return &InitData{
		Name:           name,
		Description:    fmt.Sprintf("%s returns the URL path to the %s service %s HTTP endpoint. ", name, svc.Name, method.Name),
		ServerArgs:     initArgs,
		ClientArgs:     initArgs,
		ReturnTypeName: "string",
		ReturnTypeRef:  "string",
		ServerCode:     code,
		ClientCode:     code,
	}
}

func (sds *ServicesData) buildRequirementSchemes(endpointIR *transportir.Endpoint) (service.RequirementsData, service.SchemesData, service.SchemesData, service.SchemesData, *service.SchemeData) {
	reqs, allSchemes := service.BuildRequirementsData(endpointIR.Security.Requirements, &expr.MethodExpr{Payload: endpointIR.Request.Payload})
	basicScheme, grouped, bodySchemes := service.PartitionSchemesByIn(allSchemes)
	headerSchemes := grouped["header"]
	querySchemes := grouped["query"]
	return reqs, headerSchemes, bodySchemes, querySchemes, basicScheme
}

func endpointRequestEncoderName(method *service.MethodData, payload *PayloadData, basicScheme *service.SchemeData) string {
	if payload.Request.ClientBody == nil &&
		len(payload.Request.Headers) == 0 &&
		len(payload.Request.QueryParams) == 0 &&
		len(payload.Request.Cookies) == 0 &&
		basicScheme == nil {
		return ""
	}
	return fmt.Sprintf("Encode%sRequest", method.VarName)
}

func (sds *ServicesData) buildClientRequestInit(endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data, routes []*RouteData) *InitData {
	name := fmt.Sprintf("Build%sRequest", method.VarName)
	scope := codegen.NewNameScope()
	scope.Unique("c")
	args := make([]*InitArgData, 0, len(routes[0].PathInit.ClientArgs))
	for _, arg := range routes[0].PathInit.ClientArgs {
		if arg.FieldName == "" {
			continue
		}
		arg.VarName = scope.Unique(arg.VarName)
		arg.Ref = arg.VarName
		_, arg.IsAliased = arg.FieldType.(expr.UserType)
		if arg.IsAliased {
			if svcData := sds.ServicesData.Get(svc.Name); svcData != nil {
				arg.ServiceTypeRef = svcData.Scope.GoTypeRef(&expr.AttributeExpr{Type: arg.Type})
			} else {
				arg.ServiceTypeRef = codegen.Goify(arg.FieldType.Name(), true)
			}
		}
		args = append(args, arg)
	}
	caps := service.DescribeMethodCapabilities(method)
	pkg := service.DefaultPackageName(method.PayloadLoc, svc.PkgName)
	payloadRef := ""
	if len(routes[0].PathInit.ClientArgs) > 0 && endpointIR.Request.Payload.Type != expr.Empty {
		payloadRef = svc.Scope.GoFullTypeRef(endpointIR.Request.Payload, pkg)
	}
	requestStruct := ""
	if caps.HasRequestStruct {
		requestStruct = pkg + "." + method.RequestStruct
	}
	code := renderRequestInitCode(
		payloadRef,
		expr.IsObject(endpointIR.Request.Payload.Type),
		svc.Name,
		method.Name,
		args,
		routes[0].PathInit,
		routes[0].Verb,
		endpointIR.Stream.IsStreaming && !endpointIR.Stream.IsSSE,
		requestStruct,
	)
	return &InitData{
		Name:        name,
		Description: fmt.Sprintf("%s instantiates a HTTP request object with method and path set to call the %q service %q endpoint", name, svc.Name, method.Name),
		ClientCode:  code,
		ClientArgs:  []*InitArgData{{Ref: "v", AttributeData: &AttributeData{Name: "payload", VarName: "v", TypeRef: "any"}}},
	}
}

func parameterAttributeByName(params []*transportir.Parameter, name string) *expr.AttributeExpr {
	for _, param := range params {
		if param.Name == name {
			return param.Attribute
		}
	}
	return nil
}

func parameterPrimitivePointerByName(params []*transportir.Parameter, name string) bool {
	for _, param := range params {
		if param.Name == name {
			return param.PrimitivePointer
		}
	}
	return false
}

func payloadPrimitivePointerByName(payload *expr.AttributeExpr, name string) bool {
	if payload == nil || !expr.IsObject(payload.Type) {
		return false
	}
	return payload.IsPrimitivePointer(name, true)
}

func pathParametersObject(params []*transportir.Parameter) *expr.Object {
	object := make(expr.Object, 0, len(params))
	for _, param := range params {
		object = append(object, &expr.NamedAttributeExpr{
			Name:      param.Name,
			Attribute: param.Attribute,
		})
	}
	return &object
}
