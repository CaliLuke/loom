package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/v3/codegen"
	"github.com/CaliLuke/loom/v3/codegen/service"
	"github.com/CaliLuke/loom/v3/expr"
)

// buildPayloadData returns the data structure used to describe the endpoint
// payload including the HTTP request details. It also returns the user types
// used by the request body type recursively if any.
func (sds *ServicesData) buildPayloadData(e *expr.HTTPEndpointExpr, sd *ServiceData) *PayloadData {
	e.Body = makeHTTPType(e.Body)
	var (
		payload    = e.MethodExpr.Payload
		svc        = sd.Service
		body       = e.Body.Type
		ep         = svc.Method(e.MethodExpr.Name)
		httpsvrctx = httpContext(sd.Scope, true, true)
		httpclictx = httpContext(sd.Scope, true, false)
		pkg        = pkgWithDefault(ep.PayloadLoc, svc.PkgName)
		svcctx     = serviceContext(pkg, sd.Service.Scope)

		request       *RequestData
		mapQueryParam *ParamData
	)
	request, mapQueryParam = sds.buildPayloadRequestData(e, payload, svcctx, httpsvrctx, sd)
	init := sds.buildPayloadInit(payload, body, e, ep, pkg, request, svcctx, httpsvrctx, httpclictx, sd)
	request.PayloadInit = init
	name, ref := buildPayloadMetadata(svc, payload, pkg)
	returnValue := buildPayloadDecoderReturnValue(e, init, mapQueryParam)
	data := &PayloadData{
		Name:               name,
		Ref:                ref,
		Request:            request,
		DecoderReturnValue: returnValue,
	}
	data.IDAttribute, data.IDAttributeRequired = buildPayloadIDData(e, payload)
	return data
}

func buildPayloadMetadata(svc *service.Data, payload *expr.AttributeExpr, pkg string) (string, string) {
	if payload.Type == expr.Empty {
		return "", ""
	}
	return svc.Scope.GoFullTypeName(payload, pkg), svc.Scope.GoFullTypeRef(payload, pkg)
}

func buildPayloadDecoderReturnValue(e *expr.HTTPEndpointExpr, init *InitData, mapQueryParam *ParamData) string {
	if init != nil {
		return ""
	}
	if o := expr.AsObject(e.Params.Type); o != nil && len(*o) > 0 {
		return codegen.Goify((*o)[0].Name, false)
	}
	if o := expr.AsObject(e.Headers.Type); o != nil && len(*o) > 0 {
		return codegen.Goify((*o)[0].Name, false)
	}
	if o := expr.AsObject(e.Cookies.Type); o != nil && len(*o) > 0 {
		return codegen.Goify((*o)[0].Name, false)
	}
	if e.MapQueryParams != nil && *e.MapQueryParams == "" && mapQueryParam != nil {
		return mapQueryParam.VarName
	}
	return ""
}

func buildPayloadIDData(e *expr.HTTPEndpointExpr, payload *expr.AttributeExpr) (string, bool) {
	if !e.IsJSONRPC() || e.PayloadIDAttribute == "" {
		return "", false
	}
	return codegen.Goify(e.PayloadIDAttribute, true), payload.IsRequired(e.PayloadIDAttribute)
}

func (sds *ServicesData) buildPayloadInit(
	payload *expr.AttributeExpr,
	body expr.DataType,
	e *expr.HTTPEndpointExpr,
	ep *service.MethodData,
	pkg string,
	request *RequestData,
	svcctx, httpsvrctx, httpclictx *codegen.AttributeContext,
	sd *ServiceData,
) *InitData {
	if !needInit(payload.Type) {
		return nil
	}
	return sds.buildPayloadInitData(e, payload, body, ep, pkg, request, svcctx, httpsvrctx, httpclictx, sd)
}

func (sds *ServicesData) buildPayloadRequestData(e *expr.HTTPEndpointExpr, payload *expr.AttributeExpr, svcctx, httpsvrctx *codegen.AttributeContext, sd *ServiceData) (*RequestData, *ParamData) {
	serverBodyData, clientBodyData := sds.buildPayloadRequestBodies(e, payload, sd)
	paramsData, queryData, headersData, cookiesData := sds.buildPayloadRequestElements(e, payload, svcctx, sd)
	multipartGen, multipartFiles := generatedMultipartRequestData(e)
	mapQueryParam := sds.buildMapQueryParam(e, payload, httpsvrctx, sd)
	if mapQueryParam != nil {
		queryData = append(queryData, mapQueryParam)
	}
	registerRequestBodyTypeNames(serverBodyData, sd)
	origin, mustHaveBody := buildPayloadRequestBodyRequirements(e, payload)
	return &RequestData{
		PathParams:          paramsData,
		QueryParams:         queryData,
		Headers:             headersData,
		Cookies:             cookiesData,
		ServerBody:          serverBodyData,
		ClientBody:          clientBodyData,
		PayloadAttr:         codegen.Goify(origin, true),
		PayloadType:         e.MethodExpr.Payload.Type,
		MustHaveBody:        mustHaveBody,
		MustValidate:        payloadRequestNeedsValidation(paramsData, queryData, headersData, cookiesData),
		Multipart:           e.MultipartRequest,
		MultipartGenerated:  multipartGen,
		MultipartFileFields: multipartFiles,
		FormEncoded:         e.FormRequest,
	}, mapQueryParam
}

func (sds *ServicesData) buildPayloadRequestBodies(e *expr.HTTPEndpointExpr, payload *expr.AttributeExpr, sd *ServiceData) (*TypeData, *TypeData) {
	return sds.buildRequestBodyType(e.Body, payload, e, true, sd), sds.buildRequestBodyType(e.Body, payload, e, false, sd)
}

func (sds *ServicesData) buildPayloadRequestElements(
	e *expr.HTTPEndpointExpr,
	payload *expr.AttributeExpr,
	svcctx *codegen.AttributeContext,
	sd *ServiceData,
) ([]*ParamData, []*ParamData, []*HeaderData, []*CookieData) {
	return sds.extractPathParams(e.PathParams(), payload, sd.Scope),
		sds.extractQueryParams(e.QueryParams(), payload, sd.Scope),
		sds.extractHeaders(e.Headers, payload, svcctx, sd.Scope),
		sds.extractCookies(e.Cookies, payload, svcctx, sd.Scope)
}

func registerRequestBodyTypeNames(serverBodyData *TypeData, sd *ServiceData) {
	if serverBodyData == nil {
		return
	}
	sd.ServerTypeNames[serverBodyData.Name] = false
	sd.ClientTypeNames[serverBodyData.Name] = false
}

func buildPayloadRequestBodyRequirements(e *expr.HTTPEndpointExpr, payload *expr.AttributeExpr) (string, bool) {
	origin := ""
	mustHaveBody := true
	if e.Body.Type == expr.Empty {
		return origin, mustHaveBody
	}
	if e.OptionalRequestBody {
		mustHaveBody = false
	}
	if o, ok := e.Body.Meta["origin:attribute"]; ok {
		origin = o[0]
		if !payload.IsRequired(o[0]) {
			mustHaveBody = false
		}
	}
	return origin, mustHaveBody
}

func (sds *ServicesData) buildMapQueryParam(e *expr.HTTPEndpointExpr, payload *expr.AttributeExpr, httpsvrctx *codegen.AttributeContext, sd *ServiceData) *ParamData {
	if e.MapQueryParams == nil {
		return nil
	}
	fieldName := ""
	name := "query"
	required := true
	pAtt := payload
	if n := *e.MapQueryParams; n != "" {
		pAtt = expr.AsObject(payload.Type).Attribute(n)
		required = payload.IsRequired(n)
		name = n
		fieldName = codegen.Goify(name, true)
	}
	varName := codegen.Goify(name, false)
	return &ParamData{
		MapQueryParams: e.MapQueryParams,
		Map:            expr.AsMap(payload.Type) != nil,
		Element: &Element{
			HTTPName: name,
			AttributeData: &AttributeData{
				Name:         name,
				VarName:      varName,
				FieldName:    fieldName,
				FieldType:    pAtt.Type,
				Required:     required,
				Type:         pAtt.Type,
				TypeName:     sd.Scope.GoTypeName(pAtt),
				TypeRef:      sd.Scope.GoTypeRef(pAtt),
				Validate:     codegen.AttributeValidationCode(pAtt, nil, httpsvrctx, required, expr.IsAlias(pAtt.Type), varName, name),
				DefaultValue: pAtt.DefaultValue,
				Example:      pAtt.Example(sds.Root.API.ExampleGenerator),
			},
		},
	}
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
	e *expr.HTTPEndpointExpr,
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
	serverArgs, clientArgs := sds.buildPayloadBodyArgs(e, body, argsCap, httpclictx, httpsvrctx, sd)
	args := buildPayloadFieldArgs(request)
	serverArgs = append(serverArgs, args...)
	clientArgs = append(clientArgs, args...)
	serverCode, clientCode, origin, pointer := sds.buildPayloadTransformCode(e, payload, body, svcctx, httpsvrctx, httpclictx, sd)
	return &InitData{
		Name:                     name,
		Description:              fmt.Sprintf("%s builds a %s service %s endpoint payload.", name, svc.Name, e.Name()),
		ServerArgs:               serverArgs,
		ClientArgs:               clientArgs,
		CLIArgs:                  buildBasicAuthCLIArgs(ep, e, svc, httpsvrctx, sds.Root.API.ExampleGenerator),
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
	e *expr.HTTPEndpointExpr,
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
			TypeName: sd.Scope.GoTypeName(e.Body),
			TypeRef:  sd.Scope.GoTypeRef(e.Body),
			Type:     body,
			Required: true,
			Example:  e.Body.Example(sds.Root.API.ExampleGenerator),
			Validate: svcode,
		},
	})
	clientArgs = append(clientArgs, &InitArgData{
		Ref: sd.Scope.GoVar("body", body),
		AttributeData: &AttributeData{
			Name:     "body",
			VarName:  "body",
			TypeName: sd.Scope.GoTypeNameWithDefaults(e.Body),
			TypeRef:  sd.Scope.GoTypeRefWithDefaults(e.Body),
			Type:     body,
			Required: true,
			Example:  e.Body.Example(sds.Root.API.ExampleGenerator),
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
		appendField(cookie.VarName, cookie.Name, cookie.VarName, "", cookie.FieldName, cookie.FieldPointer, cookie.FieldType, cookie.TypeName, cookie.TypeRef, cookie.Type, cookie.Pointer, cookie.Required, cookie.DefaultValue, cookie.Validate, cookie.Example)
	}
	return args
}

func buildBasicAuthCLIArgs(ep *service.MethodData, e *expr.HTTPEndpointExpr, svc *service.Data, httpsvrctx *codegen.AttributeContext, generator *expr.ExampleGenerator) []*InitArgData {
	for _, requirement := range ep.Requirements {
		for _, scheme := range requirement.Schemes {
			if scheme.Type != "Basic" {
				continue
			}
			uatt := e.MethodExpr.Payload.Find(scheme.UsernameAttr)
			uref := svc.Scope.GoTypeRef(uatt)
			if scheme.UsernamePointer {
				uref = "*" + uref
			}
			patt := e.MethodExpr.Payload.Find(scheme.PasswordAttr)
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
	e *expr.HTTPEndpointExpr,
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
		if o, ok := e.Body.Meta["origin:attribute"]; ok {
			origin = o[0]
			pAtt = expr.AsObject(payload.Type).Attribute(origin)
			pointer = !payload.IsRequired(o[0]) && expr.IsPrimitive(pAtt.Type)
		}
		var helpers []*codegen.TransformFunctionData
		serverCode, helpers, err = unmarshal(e.Body, pAtt, "body", httpsvrctx, svcctx)
		if err == nil {
			sd.ServerTransformHelpers = codegen.AppendHelpers(sd.ServerTransformHelpers, helpers)
		}
		clientCode, helpers, err = marshal(e.Body, pAtt, "body", "v", httpclictx, svcctx)
		if err == nil {
			sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
		}
	} else if expr.IsArray(payload.Type) || expr.IsMap(payload.Type) {
		if params := expr.AsObject(e.Params.Type); len(*params) > 0 {
			var helpers []*codegen.TransformFunctionData
			source := codegen.Goify((*params)[0].Name, false)
			serverCode, helpers, err = unmarshal((*params)[0].Attribute, payload, source, httpsvrctx, svcctx)
			if err == nil {
				sd.ServerTransformHelpers = codegen.AppendHelpers(sd.ServerTransformHelpers, helpers)
			}
			clientCode, helpers, err = marshal((*params)[0].Attribute, payload, source, "v", httpclictx, svcctx)
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
