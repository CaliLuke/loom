package codegen

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type payloadBuilder struct {
	sds        *ServicesData
	endpointIR *transportir.Endpoint
	sd         *ServiceData
	payload    *expr.AttributeExpr
	bodyAttr   *expr.AttributeExpr
	body       expr.DataType
	ep         *service.MethodData
	svc        *service.Data
	httpsvrctx *codegen.AttributeContext
	httpclictx *codegen.AttributeContext
	svcctx     *codegen.AttributeContext
	pkg        string
}

func (sds *ServicesData) buildPayloadDataFromIR(endpointIR *transportir.Endpoint, sd *ServiceData) *PayloadData {
	return newPayloadBuilder(sds, endpointIR, sd).build()
}

func newPayloadBuilder(sds *ServicesData, endpointIR *transportir.Endpoint, sd *ServiceData) *payloadBuilder {
	payload := endpointIR.Request.Payload
	svc := sd.Service
	ep := svc.Method(endpointIR.MethodName)
	pkg := service.DefaultPackageName(ep.PayloadLoc, svc.PkgName)
	bodyAttr := endpointIR.Request.Body
	body := expr.DataType(expr.Empty)
	if bodyAttr != nil {
		body = bodyAttr.Type
	}
	return &payloadBuilder{
		sds:        sds,
		endpointIR: endpointIR,
		sd:         sd,
		payload:    payload,
		bodyAttr:   bodyAttr,
		body:       body,
		ep:         ep,
		svc:        svc,
		httpsvrctx: httpContext(sd.Scope, true, true),
		httpclictx: httpContext(sd.Scope, true, false),
		svcctx:     serviceContext(pkg, sd.Service.Scope),
		pkg:        pkg,
	}
}

func (b *payloadBuilder) build() *PayloadData {
	request, mapQueryParam := b.buildRequestData()
	init := b.buildInit(request)
	request.PayloadInit = init
	payloadDesc := service.BuildPayloadDescriptor(b.svc, b.ep, b.payload)
	returnValue := buildPayloadDecoderReturnValue(b.endpointIR.Request, init, mapQueryParam)
	data := &PayloadData{
		Name:               payloadDesc.Name,
		Ref:                payloadDesc.Ref,
		Request:            request,
		DecoderReturnValue: returnValue,
	}
	data.IDAttribute, data.IDAttributeRequired = buildPayloadIDData(b.endpointIR.Request, b.payload)
	return data
}

func buildPayloadDecoderReturnValue(request *transportir.Request, init *InitData, mapQueryParam *ParamData) string {
	if init != nil {
		return ""
	}
	if len(request.PathParams) > 0 {
		return codegen.Goify(request.PathParams[0].Name, false)
	}
	for _, query := range request.QueryParams {
		if query.MapQueryParams != nil {
			continue
		}
		return codegen.Goify(query.Name, false)
	}
	if len(request.Headers) > 0 {
		return codegen.Goify(request.Headers[0].Name, false)
	}
	if len(request.Cookies) > 0 {
		return codegen.Goify(request.Cookies[0].Name, false)
	}
	if request.MapQueryParams != nil && *request.MapQueryParams == "" && mapQueryParam != nil {
		return mapQueryParam.VarName
	}
	return ""
}

func buildPayloadIDData(request *transportir.Request, payload *expr.AttributeExpr) (string, bool) {
	if request == nil || request.IDAttribute == "" {
		return "", false
	}
	return codegen.Goify(request.IDAttribute, true), payload.IsRequired(request.IDAttribute)
}

func (b *payloadBuilder) buildInit(request *RequestData) *InitData {
	if !needInit(b.payload) {
		return nil
	}
	return b.buildInitData(request)
}

func (b *payloadBuilder) buildRequestData() (*RequestData, *ParamData) {
	serverBodyData, clientBodyData := b.buildRequestBodies()
	paramsData, queryData, headersData, cookiesData, mapQueryParam := b.buildRequestElements()
	multipartGen, multipartFiles := generatedMultipartRequestData(b.endpointIR.Request)
	registerRequestBodyTypeNames(serverBodyData, b.sd)
	origin, mustHaveBody := buildPayloadRequestBodyRequirements(b.endpointIR.Request)
	request := &RequestData{
		PathParams:          paramsData,
		QueryParams:         queryData,
		Headers:             headersData,
		Cookies:             cookiesData,
		ServerBody:          serverBodyData,
		ClientBody:          clientBodyData,
		PayloadAttr:         codegen.Goify(origin, true),
		PayloadType:         b.endpointIR.Request.Payload.Type,
		MustHaveBody:        mustHaveBody,
		MustValidate:        payloadRequestNeedsValidation(paramsData, queryData, headersData, cookiesData),
		Multipart:           b.endpointIR.Request.Multipart,
		MultipartGenerated:  multipartGen,
		MultipartFileFields: multipartFiles,
		FormEncoded:         b.endpointIR.Request.FormEncoded,
	}
	request.DecodePlan = newRequestDecodePlan(request)
	return request, mapQueryParam
}

func newRequestDecodePlan(request *RequestData) *RequestDecodePlan {
	hasPathParams := len(request.PathParams) > 0
	hasQueryParams := len(request.QueryParams) > 0
	hasHeaders := len(request.Headers) > 0
	hasCookies := len(request.Cookies) > 0
	return &RequestDecodePlan{
		HasElements:    hasPathParams || hasQueryParams || hasHeaders || hasCookies,
		HasPathParams:  hasPathParams,
		HasQueryParams: hasQueryParams,
		HasHeaders:     hasHeaders,
		HasCookies:     hasCookies,
		MustValidate:   request.MustValidate,
	}
}

func (b *payloadBuilder) buildRequestBodies() (*TypeData, *TypeData) {
	return b.sds.buildRequestBodyType(b.bodyAttr, b.payload, b.endpointIR.Name, b.endpointIR.Request.FormEncoded, true, b.sd),
		b.sds.buildRequestBodyType(b.bodyAttr, b.payload, b.endpointIR.Name, b.endpointIR.Request.FormEncoded, false, b.sd)
}

func (b *payloadBuilder) buildRequestElements() ([]*ParamData, []*ParamData, []*HeaderData, []*CookieData, *ParamData) {
	request := b.endpointIR.Request
	paramsData := b.buildPathParams(request.PathParams)
	queryData := b.buildQueryParams(request.QueryParams)
	headersData := b.buildHeaders(request.Headers)
	cookiesData := b.buildCookies(request.Cookies)
	mapQueryParam := b.buildMapQueryParam()
	if mapQueryParam != nil {
		queryData = append(queryData, mapQueryParam)
	}
	return paramsData, queryData, headersData, cookiesData, mapQueryParam
}

func registerRequestBodyTypeNames(serverBodyData *TypeData, sd *ServiceData) {
	if serverBodyData == nil {
		return
	}
	sd.ServerTypeNames[serverBodyData.Name] = false
	sd.ClientTypeNames[serverBodyData.Name] = false
}

func buildPayloadRequestBodyRequirements(request *transportir.Request) (string, bool) {
	if request == nil {
		return "", true
	}
	return request.BodyOrigin, request.MustHaveBody
}

func (b *payloadBuilder) buildMapQueryParam() *ParamData {
	for _, param := range b.endpointIR.Request.QueryParams {
		if param.MapQueryParams == nil {
			continue
		}
		fieldName := ""
		attr := b.payload
		if param.Name != "query" || (param.MapQueryParams != nil && *param.MapQueryParams != "") {
			fieldName = codegen.Goify(param.Name, true)
			if object := expr.AsObject(b.payload.Type); object != nil {
				if payloadAttr := object.Attribute(param.Name); payloadAttr != nil {
					attr = payloadAttr
				}
			}
		}
		varName := codegen.Goify(param.Name, false)
		return &ParamData{
			MapQueryParams: param.MapQueryParams,
			Map:            expr.AsMap(b.payload.Type) != nil,
			Element: &Element{
				HTTPName: param.HTTPName,
				AttributeData: &AttributeData{
					Name:         param.Name,
					VarName:      varName,
					FieldName:    fieldName,
					FieldType:    attr.Type,
					Required:     param.Required,
					Type:         attr.Type,
					TypeName:     b.sd.Scope.GoTypeName(attr),
					TypeRef:      b.sd.Scope.GoTypeRef(attr),
					Validate:     codegen.AttributeValidationCode(attr, nil, b.httpsvrctx, param.Required, expr.IsAlias(attr.Type), varName, param.Name),
					DefaultValue: attr.DefaultValue,
					Example:      attr.Example(b.sds.Root.API.ExampleGenerator),
				},
			},
		}
	}
	return nil
}

func (b *payloadBuilder) buildPathParams(params []*transportir.Parameter) []*ParamData {
	data := make([]*ParamData, 0, len(params))
	ctx := serviceContext("", b.sd.Scope)
	for _, param := range params {
		attr := makeHTTPType(param.Attribute)
		stringSlice := transportStringSlice(rawServiceField(b.payload, param.Name, param.Attribute))
		fieldName, fieldType, fieldPointer := transportFieldBinding(param.Name, attr, b.payload, nil)
		data = append(data, &ParamData{
			Map:            false,
			MapStringSlice: false,
			Element:        b.sds.buildTransportElement(param.Name, param.HTTPName, attr, stringSlice, true, false, fieldName, fieldType, fieldPointer, ctx, b.sd.Scope),
		})
	}
	return data
}

func (b *payloadBuilder) buildQueryParams(params []*transportir.Parameter) []*ParamData {
	data := make([]*ParamData, 0, len(params))
	ctx := serviceContext("", b.sd.Scope)
	for _, param := range params {
		if param.MapQueryParams != nil {
			continue
		}
		attr := makeHTTPType(param.Attribute)
		mp := expr.AsMap(attr.Type)
		stringSlice := transportStringSlice(rawServiceField(b.payload, param.Name, param.Attribute))
		fieldName, fieldType, fieldPointer := transportFieldBinding(param.Name, attr, b.payload, nil)
		data = append(data, &ParamData{
			Map: mp != nil,
			MapStringSlice: mp != nil &&
				mp.KeyType.Type.Kind() == expr.StringKind &&
				mp.ElemType.Type.Kind() == expr.ArrayKind &&
				expr.AsArray(mp.ElemType.Type).ElemType.Type.Kind() == expr.StringKind,
			Element: b.sds.buildTransportElement(param.Name, param.HTTPName, attr, stringSlice, param.Required, param.PrimitivePointer, fieldName, fieldType, fieldPointer, ctx, b.sd.Scope),
		})
	}
	return data
}

func rawServiceField(service *expr.AttributeExpr, name string, fallback *expr.AttributeExpr) *expr.AttributeExpr {
	if service != nil {
		if field := service.Find(name); field != nil {
			return field
		}
	}
	return fallback
}

func (b *payloadBuilder) buildHeaders(params []*transportir.Parameter) []*HeaderData {
	headers := make([]*HeaderData, 0, len(params))
	for _, param := range params {
		attr := b.payload.Find(param.Name)
		if attr == nil {
			attr = b.payload
		}
		stringSlice := transportStringSlice(attr)
		hattr := makeHTTPType(attr)
		fieldName, fieldType, fieldPointer := transportFieldBinding(param.Name, attr, b.payload, b.svcctx)
		headers = append(headers, &HeaderData{
			CanonicalName: http.CanonicalHeaderKey(param.HTTPName),
			Element:       b.sds.buildTransportElement(param.Name, param.HTTPName, hattr, stringSlice, param.Required, param.PrimitivePointer, fieldName, fieldType, fieldPointer, b.svcctx, b.sd.Scope),
		})
	}
	return headers
}

func (b *payloadBuilder) buildCookies(params []*transportir.Parameter) []*CookieData {
	cookies := make([]*CookieData, 0, len(params))
	for _, param := range params {
		if _, ok := param.Attribute.Meta["loom:transport-only-session-cookie"]; ok {
			continue
		}
		cookies = append(cookies, b.sds.cookieData(param.Name, param.HTTPName, param.Required, param.PrimitivePointer, param.Attribute, b.payload, b.svcctx, b.sd.Scope))
	}
	return cookies
}

func payloadRequestNeedsValidation(paramsData []*ParamData, queryData []*ParamData, headersData []*HeaderData, cookiesData []*CookieData) bool {
	for _, cookie := range cookiesData {
		if cookie.Required || cookie.Validate != "" || needConversion(cookie.Type) || cookie.IsTextUnmarshaler {
			return true
		}
	}
	for _, param := range paramsData {
		if param.Validate != "" || needConversion(param.Type) || param.IsTextUnmarshaler {
			return true
		}
	}
	for _, query := range queryData {
		if query.Map || query.Validate != "" || query.Required || needConversion(query.Type) || query.IsTextUnmarshaler {
			return true
		}
	}
	for _, header := range headersData {
		if header.Validate != "" || header.Required || needConversion(header.Type) || header.IsTextUnmarshaler {
			return true
		}
	}
	return false
}

func (b *payloadBuilder) buildInitData(request *RequestData) *InitData {
	argsCap := len(request.PathParams) + len(request.QueryParams) + len(request.Headers) + len(request.Cookies)
	n := codegen.Goify(b.ep.Name, true)
	p := codegen.Goify(b.ep.Payload, true)
	name := ""
	if strings.HasPrefix(p, n) {
		p = b.svc.Scope.HashedUnique(b.payload.Type, p)
		name = fmt.Sprintf("New%s", p)
	} else {
		name = fmt.Sprintf("New%s%s", n, p)
	}
	serverArgs, clientArgs := b.buildPayloadBodyArgs(argsCap)
	args := buildPayloadFieldArgs(request)
	serverArgs = append(serverArgs, args...)
	clientArgs = append(clientArgs, args...)
	serverCode, clientCode, origin, pointer := b.buildTransformCode()
	return &InitData{
		Name:                     name,
		Description:              fmt.Sprintf("%s builds a %s service %s endpoint payload.", name, b.svc.Name, b.endpointIR.Name),
		ServerArgs:               serverArgs,
		ClientArgs:               clientArgs,
		CLIArgs:                  buildBasicAuthCLIArgs(b.ep, b.endpointIR.Request.Payload, b.svc, b.httpsvrctx, b.sds.Root.API.ExampleGenerator),
		ReturnTypeName:           b.svc.Scope.GoFullTypeName(b.payload, b.pkg),
		ReturnTypeRef:            b.svc.Scope.GoFullTypeRef(b.payload, b.pkg),
		ReturnIsStruct:           expr.IsObject(b.payload.Type),
		ReturnTypeAttribute:      codegen.Goify(origin, true),
		ReturnTypePkg:            b.pkg,
		ServerCode:               serverCode,
		ClientCode:               clientCode,
		ReturnIsPrimitivePointer: pointer,
	}
}

func (b *payloadBuilder) buildPayloadBodyArgs(argsCap int) ([]*InitArgData, []*InitArgData) {
	serverArgs := make([]*InitArgData, 0, argsCap+1)
	clientArgs := make([]*InitArgData, 0, argsCap+1)
	if b.body == expr.Empty {
		return serverArgs, clientArgs
	}
	svcode := ""
	cvcode := ""
	if ut, ok := b.body.(expr.UserType); ok {
		if val := ut.Attribute().Validation; val != nil {
			svcode = codegen.ValidationCode(ut.Attribute(), ut, b.httpsvrctx, true, expr.IsAlias(ut), false, "body")
			cvcode = codegen.ValidationCode(ut.Attribute(), ut, b.httpclictx, true, expr.IsAlias(ut), false, "body")
		}
	}
	serverArgs = append(serverArgs, &InitArgData{
		Ref: b.sd.Scope.GoVar("body", b.body),
		AttributeData: &AttributeData{
			Name:     "body",
			VarName:  "body",
			TypeName: b.sd.Scope.GoTypeName(b.bodyAttr),
			TypeRef:  b.sd.Scope.GoTypeRef(b.bodyAttr),
			Type:     b.body,
			Required: true,
			Example:  b.bodyAttr.Example(b.sds.Root.API.ExampleGenerator),
			Validate: svcode,
		},
	})
	clientArgs = append(clientArgs, &InitArgData{
		Ref: b.sd.Scope.GoVar("body", b.body),
		AttributeData: &AttributeData{
			Name:     "body",
			VarName:  "body",
			TypeName: b.sd.Scope.GoTypeNameWithDefaults(b.bodyAttr),
			TypeRef:  b.sd.Scope.GoTypeRefWithDefaults(b.bodyAttr),
			Type:     b.body,
			Required: true,
			Example:  b.bodyAttr.Example(b.sds.Root.API.ExampleGenerator),
			Validate: cvcode,
		},
	})
	return serverArgs, clientArgs
}

func (b *payloadBuilder) buildTransformCode() (string, string, string, bool) {
	serverCode := ""
	clientCode := ""
	origin := ""
	pointer := false
	var err error
	pAtt := b.payload
	request := b.endpointIR.Request
	if b.body != expr.Empty {
		if o, ok := request.Body.Meta["origin:attribute"]; ok {
			origin = o[0]
			pAtt = expr.AsObject(b.payload.Type).Attribute(origin)
			pointer = !b.payload.IsRequired(o[0]) && expr.IsPrimitive(pAtt.Type)
		}
		var helpers []*codegen.TransformFunctionData
		serverCode, helpers, err = unmarshal(request.Body, pAtt, "body", b.httpsvrctx, b.svcctx)
		if err == nil {
			b.sd.ServerTransformHelpers = codegen.AppendHelpers(b.sd.ServerTransformHelpers, helpers)
		}
		clientCode, helpers, err = marshal(request.Body, pAtt, "body", "v", b.httpclictx, b.svcctx)
		if err == nil {
			b.sd.ClientTransformHelpers = codegen.AppendHelpers(b.sd.ClientTransformHelpers, helpers)
		}
	} else if expr.IsArray(b.payload.Type) || expr.IsMap(b.payload.Type) {
		if len(request.PathParams) > 0 {
			var helpers []*codegen.TransformFunctionData
			sourceParam := request.PathParams[0]
			source := codegen.Goify(sourceParam.Name, false)
			serverCode, helpers, err = unmarshal(sourceParam.Attribute, b.payload, source, b.httpsvrctx, b.svcctx)
			if err == nil {
				b.sd.ServerTransformHelpers = codegen.AppendHelpers(b.sd.ServerTransformHelpers, helpers)
			}
			clientCode, helpers, err = marshal(sourceParam.Attribute, b.payload, source, "v", b.httpclictx, b.svcctx)
			if err == nil {
				b.sd.ClientTransformHelpers = codegen.AppendHelpers(b.sd.ClientTransformHelpers, helpers)
			}
		}
	}
	if err != nil {
		panic(codegen.NewError(b.sds.Ctx, b.bodyAttr, fmt.Errorf("build HTTP payload transform for %s: %w", b.endpointIR.MethodName, err)))
	}
	return serverCode, clientCode, origin, pointer
}
