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

func (sds *ServicesData) buildPayloadDataFromIR(endpointIR *transportir.Endpoint, sd *ServiceData) *PayloadData {
	var (
		payload                  = endpointIR.Request.Payload
		svc                      = sd.Service
		bodyAttr                 = endpointIR.Request.Body
		body       expr.DataType = expr.Empty
		ep                       = svc.Method(endpointIR.MethodName)
		httpsvrctx               = httpContext(sd.Scope, true, true)
		httpclictx               = httpContext(sd.Scope, true, false)
		pkg                      = pkgWithDefault(ep.PayloadLoc, svc.PkgName)
		svcctx                   = serviceContext(pkg, sd.Service.Scope)

		request       *RequestData
		mapQueryParam *ParamData
	)
	if bodyAttr != nil {
		body = bodyAttr.Type
	}
	request, mapQueryParam = sds.buildPayloadRequestData(endpointIR, bodyAttr, payload, svcctx, httpsvrctx, sd)
	init := sds.buildPayloadInit(endpointIR, payload, body, ep, pkg, request, svcctx, httpsvrctx, httpclictx, sd)
	request.PayloadInit = init
	name, ref := buildPayloadMetadata(svc, payload, pkg)
	returnValue := buildPayloadDecoderReturnValue(endpointIR.Request, init, mapQueryParam)
	data := &PayloadData{
		Name:               name,
		Ref:                ref,
		Request:            request,
		DecoderReturnValue: returnValue,
	}
	data.IDAttribute, data.IDAttributeRequired = buildPayloadIDData(endpointIR.Request, payload)
	return data
}

func buildPayloadMetadata(svc *service.Data, payload *expr.AttributeExpr, pkg string) (string, string) {
	if payload.Type == expr.Empty {
		return "", ""
	}
	return svc.Scope.GoFullTypeName(payload, pkg), svc.Scope.GoFullTypeRef(payload, pkg)
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

func (sds *ServicesData) buildPayloadInit(
	endpointIR *transportir.Endpoint,
	payload *expr.AttributeExpr,
	body expr.DataType,
	ep *service.MethodData,
	pkg string,
	request *RequestData,
	svcctx, httpsvrctx, httpclictx *codegen.AttributeContext,
	sd *ServiceData,
) *InitData {
	if !needInit(payload.Type) {
		return nil
	}
	return sds.buildPayloadInitData(endpointIR, payload, body, ep, pkg, request, svcctx, httpsvrctx, httpclictx, sd)
}

func (sds *ServicesData) buildPayloadRequestData(endpointIR *transportir.Endpoint, body *expr.AttributeExpr, payload *expr.AttributeExpr, svcctx, httpsvrctx *codegen.AttributeContext, sd *ServiceData) (*RequestData, *ParamData) {
	serverBodyData, clientBodyData := sds.buildPayloadRequestBodies(body, endpointIR, payload, sd)
	paramsData, queryData, headersData, cookiesData, mapQueryParam := sds.buildPayloadRequestElements(endpointIR, payload, svcctx, httpsvrctx, sd)
	multipartGen, multipartFiles := generatedMultipartRequestData(endpointIR.Request)
	registerRequestBodyTypeNames(serverBodyData, sd)
	origin, mustHaveBody := buildPayloadRequestBodyRequirements(endpointIR.Request)
	return &RequestData{
		PathParams:          paramsData,
		QueryParams:         queryData,
		Headers:             headersData,
		Cookies:             cookiesData,
		ServerBody:          serverBodyData,
		ClientBody:          clientBodyData,
		PayloadAttr:         codegen.Goify(origin, true),
		PayloadType:         endpointIR.Request.Payload.Type,
		MustHaveBody:        mustHaveBody,
		MustValidate:        payloadRequestNeedsValidation(paramsData, queryData, headersData, cookiesData),
		Multipart:           endpointIR.Request.Multipart,
		MultipartGenerated:  multipartGen,
		MultipartFileFields: multipartFiles,
		FormEncoded:         endpointIR.Request.FormEncoded,
	}, mapQueryParam
}

func (sds *ServicesData) buildPayloadRequestBodies(body *expr.AttributeExpr, endpointIR *transportir.Endpoint, payload *expr.AttributeExpr, sd *ServiceData) (*TypeData, *TypeData) {
	return sds.buildRequestBodyType(body, payload, endpointIR.Name, endpointIR.Request.FormEncoded, true, sd), sds.buildRequestBodyType(body, payload, endpointIR.Name, endpointIR.Request.FormEncoded, false, sd)
}

func (sds *ServicesData) buildPayloadRequestElements(
	endpointIR *transportir.Endpoint,
	payload *expr.AttributeExpr,
	svcctx *codegen.AttributeContext,
	httpsvrctx *codegen.AttributeContext,
	sd *ServiceData,
) ([]*ParamData, []*ParamData, []*HeaderData, []*CookieData, *ParamData) {
	request := endpointIR.Request
	paramsData := sds.buildPathParamsFromIR(request.PathParams, payload, sd.Scope)
	queryData := sds.buildQueryParamsFromIR(request.QueryParams, payload, sd.Scope)
	headersData := sds.buildHeadersFromIR(request.Headers, payload, svcctx, sd.Scope)
	cookiesData := sds.buildCookiesFromIR(request.Cookies, payload, svcctx, sd.Scope)
	mapQueryParam := sds.buildMapQueryParamFromIR(endpointIR.Request, payload, httpsvrctx, sd)
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

func (sds *ServicesData) buildMapQueryParamFromIR(request *transportir.Request, payload *expr.AttributeExpr, httpsvrctx *codegen.AttributeContext, sd *ServiceData) *ParamData {
	for _, param := range request.QueryParams {
		if param.MapQueryParams == nil {
			continue
		}
		fieldName := ""
		attr := payload
		if param.Name != "query" || (param.MapQueryParams != nil && *param.MapQueryParams != "") {
			fieldName = codegen.Goify(param.Name, true)
			if object := expr.AsObject(payload.Type); object != nil {
				if payloadAttr := object.Attribute(param.Name); payloadAttr != nil {
					attr = payloadAttr
				}
			}
		}
		varName := codegen.Goify(param.Name, false)
		return &ParamData{
			MapQueryParams: param.MapQueryParams,
			Map:            expr.AsMap(payload.Type) != nil,
			Element: &Element{
				HTTPName: param.HTTPName,
				AttributeData: &AttributeData{
					Name:         param.Name,
					VarName:      varName,
					FieldName:    fieldName,
					FieldType:    attr.Type,
					Required:     param.Required,
					Type:         attr.Type,
					TypeName:     sd.Scope.GoTypeName(attr),
					TypeRef:      sd.Scope.GoTypeRef(attr),
					Validate:     codegen.AttributeValidationCode(attr, nil, httpsvrctx, param.Required, expr.IsAlias(attr.Type), varName, param.Name),
					DefaultValue: attr.DefaultValue,
					Example:      attr.Example(sds.Root.API.ExampleGenerator),
				},
			},
		}
	}
	return nil
}

func (sds *ServicesData) buildPathParamsFromIR(params []*transportir.Parameter, service *expr.AttributeExpr, scope *codegen.NameScope) []*ParamData {
	data := make([]*ParamData, 0, len(params))
	ctx := serviceContext("", scope)
	for _, param := range params {
		attr := makeHTTPType(param.Attribute)
		stringSlice := transportStringSlice(rawServiceField(service, param.Name, param.Attribute))
		fieldName, fieldType, fieldPointer := transportFieldBinding(param.Name, attr, service, nil)
		data = append(data, &ParamData{
			Map:            false,
			MapStringSlice: false,
			Element:        sds.buildTransportElement(param.Name, param.HTTPName, attr, stringSlice, true, false, fieldName, fieldType, fieldPointer, ctx, scope),
		})
	}
	return data
}

func (sds *ServicesData) buildQueryParamsFromIR(params []*transportir.Parameter, service *expr.AttributeExpr, scope *codegen.NameScope) []*ParamData {
	data := make([]*ParamData, 0, len(params))
	ctx := serviceContext("", scope)
	for _, param := range params {
		if param.MapQueryParams != nil {
			continue
		}
		attr := makeHTTPType(param.Attribute)
		mp := expr.AsMap(attr.Type)
		stringSlice := transportStringSlice(rawServiceField(service, param.Name, param.Attribute))
		fieldName, fieldType, fieldPointer := transportFieldBinding(param.Name, attr, service, nil)
		data = append(data, &ParamData{
			Map: mp != nil,
			MapStringSlice: mp != nil &&
				mp.KeyType.Type.Kind() == expr.StringKind &&
				mp.ElemType.Type.Kind() == expr.ArrayKind &&
				expr.AsArray(mp.ElemType.Type).ElemType.Type.Kind() == expr.StringKind,
			Element: sds.buildTransportElement(param.Name, param.HTTPName, attr, stringSlice, param.Required, param.PrimitivePointer, fieldName, fieldType, fieldPointer, ctx, scope),
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

func (sds *ServicesData) buildHeadersFromIR(params []*transportir.Parameter, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) []*HeaderData {
	headers := make([]*HeaderData, 0, len(params))
	for _, param := range params {
		attr := svcAtt.Find(param.Name)
		if attr == nil {
			attr = svcAtt
		}
		stringSlice := transportStringSlice(attr)
		hattr := makeHTTPType(attr)
		fieldName, fieldType, fieldPointer := transportFieldBinding(param.Name, attr, svcAtt, svcCtx)
		headers = append(headers, &HeaderData{
			CanonicalName: http.CanonicalHeaderKey(param.HTTPName),
			Element:       sds.buildTransportElement(param.Name, param.HTTPName, hattr, stringSlice, param.Required, param.PrimitivePointer, fieldName, fieldType, fieldPointer, svcCtx, scope),
		})
	}
	return headers
}

func (sds *ServicesData) buildCookiesFromIR(params []*transportir.Parameter, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) []*CookieData {
	cookies := make([]*CookieData, 0, len(params))
	for _, param := range params {
		if _, ok := param.Attribute.Meta["loom:transport-only-session-cookie"]; ok {
			continue
		}
		cookies = append(cookies, sds.cookieData(param.Name, param.HTTPName, param.Required, param.PrimitivePointer, param.Attribute, svcAtt, svcCtx, scope))
	}
	return cookies
}

func payloadRequestNeedsValidation(paramsData []*ParamData, queryData []*ParamData, headersData []*HeaderData, cookiesData []*CookieData) bool {
	for _, cookie := range cookiesData {
		if cookie.Required || cookie.Validate != "" || needConversion(cookie.Type) {
			return true
		}
	}
	for _, param := range paramsData {
		if param.Validate != "" || needConversion(param.Type) {
			return true
		}
	}
	for _, query := range queryData {
		if query.Map || query.Validate != "" || query.Required || needConversion(query.Type) {
			return true
		}
	}
	for _, header := range headersData {
		if header.Validate != "" || header.Required || needConversion(header.Type) {
			return true
		}
	}
	return false
}

func (sds *ServicesData) buildPayloadInitData(
	endpointIR *transportir.Endpoint,
	payload *expr.AttributeExpr,
	body expr.DataType,
	ep *service.MethodData,
	pkg string,
	request *RequestData,
	svcctx, httpsvrctx, httpclictx *codegen.AttributeContext,
	sd *ServiceData,
) *InitData {
	svc := sd.Service
	argsCap := len(request.PathParams) + len(request.QueryParams) + len(request.Headers) + len(request.Cookies)
	n := codegen.Goify(ep.Name, true)
	p := codegen.Goify(ep.Payload, true)
	name := ""
	if strings.HasPrefix(p, n) {
		p = svc.Scope.HashedUnique(payload.Type, p)
		name = fmt.Sprintf("New%s", p)
	} else {
		name = fmt.Sprintf("New%s%s", n, p)
	}
	serverArgs, clientArgs := sds.buildPayloadBodyArgs(endpointIR.Request.Body, body, argsCap, httpclictx, httpsvrctx, sd)
	args := buildPayloadFieldArgs(request)
	serverArgs = append(serverArgs, args...)
	clientArgs = append(clientArgs, args...)
	serverCode, clientCode, origin, pointer := sds.buildPayloadTransformCode(endpointIR.Request, payload, body, svcctx, httpsvrctx, httpclictx, sd)
	return &InitData{
		Name:                     name,
		Description:              fmt.Sprintf("%s builds a %s service %s endpoint payload.", name, svc.Name, endpointIR.Name),
		ServerArgs:               serverArgs,
		ClientArgs:               clientArgs,
		CLIArgs:                  buildBasicAuthCLIArgs(ep, endpointIR.Request.Payload, svc, httpsvrctx, sds.Root.API.ExampleGenerator),
		ReturnTypeName:           svc.Scope.GoFullTypeName(payload, pkg),
		ReturnTypeRef:            svc.Scope.GoFullTypeRef(payload, pkg),
		ReturnIsStruct:           expr.IsObject(payload.Type),
		ReturnTypeAttribute:      codegen.Goify(origin, true),
		ReturnTypePkg:            pkg,
		ServerCode:               serverCode,
		ClientCode:               clientCode,
		ReturnIsPrimitivePointer: pointer,
	}
}

func (sds *ServicesData) buildPayloadBodyArgs(
	bodyAttr *expr.AttributeExpr,
	body expr.DataType,
	argsCap int,
	httpclictx, httpsvrctx *codegen.AttributeContext,
	sd *ServiceData,
) ([]*InitArgData, []*InitArgData) {
	serverArgs := make([]*InitArgData, 0, argsCap+1)
	clientArgs := make([]*InitArgData, 0, argsCap+1)
	if body == expr.Empty {
		return serverArgs, clientArgs
	}
	svcode := ""
	cvcode := ""
	if ut, ok := body.(expr.UserType); ok {
		if val := ut.Attribute().Validation; val != nil {
			svcode = codegen.ValidationCode(ut.Attribute(), ut, httpsvrctx, true, expr.IsAlias(ut), false, "body")
			cvcode = codegen.ValidationCode(ut.Attribute(), ut, httpclictx, true, expr.IsAlias(ut), false, "body")
		}
	}
	serverArgs = append(serverArgs, &InitArgData{
		Ref: sd.Scope.GoVar("body", body),
		AttributeData: &AttributeData{
			Name:     "body",
			VarName:  "body",
			TypeName: sd.Scope.GoTypeName(bodyAttr),
			TypeRef:  sd.Scope.GoTypeRef(bodyAttr),
			Type:     body,
			Required: true,
			Example:  bodyAttr.Example(sds.Root.API.ExampleGenerator),
			Validate: svcode,
		},
	})
	clientArgs = append(clientArgs, &InitArgData{
		Ref: sd.Scope.GoVar("body", body),
		AttributeData: &AttributeData{
			Name:     "body",
			VarName:  "body",
			TypeName: sd.Scope.GoTypeNameWithDefaults(bodyAttr),
			TypeRef:  sd.Scope.GoTypeRefWithDefaults(bodyAttr),
			Type:     body,
			Required: true,
			Example:  bodyAttr.Example(sds.Root.API.ExampleGenerator),
			Validate: cvcode,
		},
	})
	return serverArgs, clientArgs
}

func buildPayloadFieldArgs(request *RequestData) []*InitArgData {
	args := make([]*InitArgData, 0, len(request.PathParams)+len(request.QueryParams)+len(request.Headers)+len(request.Cookies))
	appendField := func(
		ref string,
		name string,
		varName string,
		description string,
		fieldName string,
		fieldPointer bool,
		fieldType expr.DataType,
		typeName string,
		typeRef string,
		typ expr.DataType,
		pointer bool,
		required bool,
		defaultValue any,
		validate string,
		example any,
	) {
		args = append(args, &InitArgData{
			Ref: ref,
			AttributeData: &AttributeData{
				Name:         name,
				VarName:      varName,
				Description:  description,
				FieldName:    fieldName,
				FieldPointer: fieldPointer,
				FieldType:    fieldType,
				TypeName:     typeName,
				TypeRef:      typeRef,
				Type:         typ,
				Pointer:      pointer,
				Required:     required,
				DefaultValue: defaultValue,
				Validate:     validate,
				Example:      example,
			},
		})
	}
	for _, param := range request.PathParams {
		appendField(param.VarName, param.Name, param.VarName, param.Description, param.FieldName, param.FieldPointer, param.FieldType, param.TypeName, param.TypeRef, param.Type, param.Pointer, param.Required, nil, param.Validate, param.Example)
	}
	for _, param := range request.QueryParams {
		appendField(param.VarName, param.Name, param.VarName, "", param.FieldName, param.FieldPointer, param.FieldType, param.TypeName, param.TypeRef, param.Type, param.Pointer, param.Required, param.DefaultValue, param.Validate, param.Example)
	}
	for _, header := range request.Headers {
		appendField(header.VarName, header.Name, header.VarName, "", header.FieldName, header.FieldPointer, header.FieldType, header.TypeName, header.TypeRef, header.Type, header.Pointer, header.Required, header.DefaultValue, header.Validate, header.Example)
	}
	for _, cookie := range request.Cookies {
		if cookie.FieldName == "" {
			continue
		}
		appendField(cookie.VarName, cookie.Name, cookie.VarName, "", cookie.FieldName, cookie.FieldPointer, cookie.FieldType, cookie.TypeName, cookie.TypeRef, cookie.Type, cookie.Pointer, cookie.Required, cookie.DefaultValue, cookie.Validate, cookie.Example)
	}
	return args
}

func buildBasicAuthCLIArgs(ep *service.MethodData, payload *expr.AttributeExpr, svc *service.Data, httpsvrctx *codegen.AttributeContext, generator *expr.ExampleGenerator) []*InitArgData {
	for _, requirement := range ep.Requirements {
		for _, scheme := range requirement.Schemes {
			if scheme.Type != "Basic" {
				continue
			}
			uatt := payload.Find(scheme.UsernameAttr)
			uref := svc.Scope.GoTypeRef(uatt)
			if scheme.UsernamePointer {
				uref = "*" + uref
			}
			patt := payload.Find(scheme.PasswordAttr)
			pref := svc.Scope.GoTypeRef(patt)
			if scheme.PasswordPointer {
				pref = "*" + pref
			}
			return []*InitArgData{
				{
					Ref: scheme.UsernameAttr,
					AttributeData: &AttributeData{
						Name:         scheme.UsernameAttr,
						VarName:      scheme.UsernameAttr,
						FieldName:    scheme.UsernameField,
						FieldPointer: scheme.UsernamePointer,
						FieldType:    uatt.Type,
						Description:  uatt.Description,
						Required:     scheme.UsernameRequired,
						TypeName:     svc.Scope.GoTypeName(uatt),
						TypeRef:      uref,
						Type:         uatt.Type,
						Pointer:      scheme.UsernamePointer,
						Validate:     codegen.ValidationCode(uatt, nil, httpsvrctx, scheme.UsernameRequired, expr.IsAlias(uatt.Type), false, scheme.UsernameAttr),
						Example:      uatt.Example(generator),
					},
				},
				{
					Ref: scheme.PasswordAttr,
					AttributeData: &AttributeData{
						Name:         scheme.PasswordAttr,
						VarName:      scheme.PasswordAttr,
						FieldName:    scheme.PasswordField,
						FieldPointer: scheme.PasswordPointer,
						FieldType:    patt.Type,
						Description:  patt.Description,
						Required:     scheme.PasswordRequired,
						TypeName:     svc.Scope.GoTypeName(patt),
						TypeRef:      pref,
						Type:         patt.Type,
						Pointer:      scheme.PasswordPointer,
						Validate:     codegen.ValidationCode(patt, nil, httpsvrctx, scheme.PasswordRequired, expr.IsAlias(patt.Type), false, scheme.PasswordAttr),
						Example:      patt.Example(generator),
					},
				},
			}
		}
	}
	return nil
}

func buildBodyInitArg(scope *codegen.NameScope, body *expr.AttributeExpr, addressObject bool) *InitArgData {
	ref := "body"
	if addressObject && expr.IsObject(body.Type) {
		ref = "&body"
	}
	return &InitArgData{
		Ref: ref,
		AttributeData: &AttributeData{
			Name:    "body",
			VarName: "body",
			TypeRef: scope.GoTypeRef(body),
		},
	}
}

func buildHeaderInitArgs(headers []*HeaderData) []*InitArgData {
	args := make([]*InitArgData, 0, len(headers))
	for _, header := range headers {
		args = append(args, &InitArgData{
			Ref: header.VarName,
			AttributeData: &AttributeData{
				Name:         header.Name,
				VarName:      header.VarName,
				FieldName:    header.FieldName,
				FieldPointer: header.FieldPointer,
				FieldType:    header.FieldType,
				Required:     header.Required,
				Pointer:      header.Pointer,
				TypeRef:      header.TypeRef,
				Type:         header.Type,
				Validate:     header.Validate,
				Example:      header.Example,
			},
		})
	}
	return args
}

func buildCookieInitArgs(cookies []*CookieData) []*InitArgData {
	args := make([]*InitArgData, 0, len(cookies))
	for _, cookie := range cookies {
		args = append(args, &InitArgData{
			Ref: cookie.VarName,
			AttributeData: &AttributeData{
				Name:         cookie.Name,
				VarName:      cookie.VarName,
				FieldName:    cookie.FieldName,
				FieldPointer: cookie.FieldPointer,
				FieldType:    cookie.FieldType,
				Required:     cookie.Required,
				Pointer:      cookie.Pointer,
				TypeRef:      cookie.TypeRef,
				Type:         cookie.Type,
				Validate:     cookie.Validate,
				Example:      cookie.Example,
			},
		})
	}
	return args
}

func responseFieldsNeedValidation(headers []*HeaderData, cookies []*CookieData) bool {
	for _, header := range headers {
		if header.Validate != "" || header.Required || needConversion(header.Type) {
			return true
		}
	}
	for _, cookie := range cookies {
		if cookie.Validate != "" || cookie.Required || needConversion(cookie.Type) {
			return true
		}
	}
	return false
}

func (sds *ServicesData) buildPayloadTransformCode(
	request *transportir.Request,
	payload *expr.AttributeExpr,
	body expr.DataType,
	svcctx, httpsvrctx, httpclictx *codegen.AttributeContext,
	sd *ServiceData,
) (string, string, string, bool) {
	serverCode := ""
	clientCode := ""
	origin := ""
	pointer := false
	var err error
	pAtt := payload
	if body != expr.Empty {
		if o, ok := request.Body.Meta["origin:attribute"]; ok {
			origin = o[0]
			pAtt = expr.AsObject(payload.Type).Attribute(origin)
			pointer = !payload.IsRequired(o[0]) && expr.IsPrimitive(pAtt.Type)
		}
		var helpers []*codegen.TransformFunctionData
		serverCode, helpers, err = unmarshal(request.Body, pAtt, "body", httpsvrctx, svcctx)
		if err == nil {
			sd.ServerTransformHelpers = codegen.AppendHelpers(sd.ServerTransformHelpers, helpers)
		}
		clientCode, helpers, err = marshal(request.Body, pAtt, "body", "v", httpclictx, svcctx)
		if err == nil {
			sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
		}
	} else if expr.IsArray(payload.Type) || expr.IsMap(payload.Type) {
		if len(request.PathParams) > 0 {
			var helpers []*codegen.TransformFunctionData
			sourceParam := request.PathParams[0]
			source := codegen.Goify(sourceParam.Name, false)
			serverCode, helpers, err = unmarshal(sourceParam.Attribute, payload, source, httpsvrctx, svcctx)
			if err == nil {
				sd.ServerTransformHelpers = codegen.AppendHelpers(sd.ServerTransformHelpers, helpers)
			}
			clientCode, helpers, err = marshal(sourceParam.Attribute, payload, source, "v", httpclictx, svcctx)
			if err == nil {
				sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
			}
		}
	}
	if err != nil {
		fmt.Println(err.Error())
	}
	return serverCode, clientCode, origin, pointer
}
