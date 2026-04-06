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
	pkg := pkgWithDefault(ep.PayloadLoc, svc.PkgName)
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
	name, ref := buildPayloadMetadata(b.svc, b.payload, b.pkg)
	returnValue := buildPayloadDecoderReturnValue(b.endpointIR.Request, init, mapQueryParam)
	data := &PayloadData{
		Name:               name,
		Ref:                ref,
		Request:            request,
		DecoderReturnValue: returnValue,
	}
	data.IDAttribute, data.IDAttributeRequired = buildPayloadIDData(b.endpointIR.Request, b.payload)
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

func (b *payloadBuilder) buildInit(request *RequestData) *InitData {
	if !needInit(b.payload.Type) {
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
	return &RequestData{
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
	}, mapQueryParam
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
		fmt.Println(err.Error())
	}
	return serverCode, clientCode, origin, pointer
}
