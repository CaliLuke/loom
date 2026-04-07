package codegen

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type resultBuilder struct {
	sds      *ServicesData
	endpoint *transportir.Endpoint
	sd       *ServiceData
	svc      *service.Data
	method   *service.MethodData
	pkg      string
	result   *expr.AttributeExpr
}

type errorBuilder struct {
	sds        *ServicesData
	endpoint   *transportir.Endpoint
	sd         *ServiceData
	svc        *service.Data
	method     *service.MethodData
	httpclictx *codegen.AttributeContext
}

func (sds *ServicesData) buildResultDataFromIR(endpointIR *transportir.Endpoint, sd *ServiceData) *ResultData {
	return newResultBuilder(sds, endpointIR, sd).build()
}

func newResultBuilder(sds *ServicesData, endpointIR *transportir.Endpoint, sd *ServiceData) *resultBuilder {
	svc := sd.Service
	method := svc.Method(endpointIR.MethodName)
	return &resultBuilder{
		sds:      sds,
		endpoint: endpointIR,
		sd:       sd,
		svc:      svc,
		method:   method,
		pkg:      service.DefaultPackageName(method.ResultLoc, svc.PkgName),
		result:   endpointIR.Response.Result,
	}
}

func (b *resultBuilder) build() *ResultData {
	resultDesc := service.BuildResultDescriptor(b.svc, b.method, b.result)
	responses, mustInit, result := b.buildResponsesData(resultDesc)
	idAtt, idAttRequired := buildResultIDData(b.endpoint.Response, result)
	return &ResultData{
		IsStruct:            expr.IsObject(result.Type),
		Name:                resultDesc.Declared.Name,
		Ref:                 resultDesc.Declared.Ref,
		IDAttribute:         idAtt,
		IDAttributeRequired: idAttRequired,
		Responses:           responses,
		View:                resultDesc.View,
		MustInit:            mustInit,
	}
}

func (b *resultBuilder) buildResponsesData(resultDesc service.ResultDescriptor) ([]*ResponseData, bool, *expr.AttributeExpr) {
	result := resultDesc.Effective.Attribute
	viewed := resultDesc.UsesViewedResult
	responses := b.sds.buildResponsesFromIR(b.endpoint, result, viewed, b.sd)
	mustInit := false
	for _, r := range responses {
		if len(r.ServerBody) > 0 || len(r.Headers) > 0 || len(r.Cookies) > 0 || r.TagName != "" {
			mustInit = true
			break
		}
	}
	return responses, mustInit, result
}

func buildResultIDData(response *transportir.Response, result *expr.AttributeExpr) (string, bool) {
	if response == nil || response.IDAttribute == "" {
		return "", false
	}
	return codegen.Goify(response.IDAttribute, true), result.IsRequired(response.IDAttribute)
}

func (sds *ServicesData) buildErrorsDataFromIR(endpointIR *transportir.Endpoint, sd *ServiceData) []*ErrorGroupData {
	return newErrorBuilder(sds, endpointIR, sd).build()
}

func newErrorBuilder(sds *ServicesData, endpointIR *transportir.Endpoint, sd *ServiceData) *errorBuilder {
	svc := sd.Service
	return &errorBuilder{
		sds:        sds,
		endpoint:   endpointIR,
		sd:         sd,
		svc:        svc,
		method:     svc.Method(endpointIR.Name),
		httpclictx: httpContext(sd.Scope, false, false),
	}
}

func (b *errorBuilder) build() []*ErrorGroupData {
	data := make(map[string][]*ErrorData)
	for _, errorResponse := range b.endpoint.Response.ErrorResponses {
		ref, errorData := b.buildSingle(errorResponse)
		data[ref] = append(data[ref], errorData)
	}
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var vals []*ErrorGroupData
	for _, k := range keys {
		es := data[k]
		for _, e := range es {
			found := false
			for _, eg := range vals {
				if eg.StatusCode == e.Response.StatusCode {
					eg.Errors = append(eg.Errors, e)
					found = true
					break
				}
			}
			if !found {
				vals = append(vals, &ErrorGroupData{StatusCode: e.Response.StatusCode, Errors: []*ErrorData{e}})
			}
		}
	}
	return vals
}

func (b *errorBuilder) buildSingle(errorResponse *transportir.ResponseStatus) (string, *ErrorData) {
	httpError := errorResponse.Error
	errorDesc := service.BuildErrorDescriptor(b.svc, b.method, httpError.Name, httpError.Attribute)
	pkg := errorDesc.Type.Package
	errctx := serviceContext(pkg, b.sd.Service.Scope)
	init := b.buildResultInit(errorResponse, pkg, errctx)
	responseData := b.buildResponseData(errorResponse, errctx, init)
	ref := errorDesc.Type.Ref
	return ref, &ErrorData{Name: httpError.Name, Response: responseData, Ref: ref}
}

func (b *errorBuilder) buildResultInit(errorResponse *transportir.ResponseStatus, pkg string, errctx *codegen.AttributeContext) *InitData {
	httpError := errorResponse.Error
	body := responseStatusBody(errorResponse).Type
	if !needInit(httpError.Type) {
		return nil
	}
	headers := b.sds.extractHeaders(errorResponse.Headers, httpError.Attribute, errctx, b.sd.Scope)
	cookies := b.sds.extractResponseCookies(errorResponse.Cookies, httpError.Attribute, errctx, b.sd.Scope)
	args := make([]*InitArgData, 0, len(headers)+len(cookies)+1)
	if body != expr.Empty {
		args = append(args, buildBodyInitArg(b.sd.Scope, responseStatusBody(errorResponse), true))
	}
	args = append(args, buildHeaderInitArgs(headers)...)
	args = append(args, buildCookieInitArgs(cookies)...)
	code, origin, skipFieldInit := b.buildResultInitCode(errorResponse, errctx, args)
	name := fmt.Sprintf("New%s%s", codegen.Goify(b.method.Name, true), codegen.Goify(httpError.Name, true))
	return &InitData{
		Name:                name,
		Description:         fmt.Sprintf("%s builds a %s service %s endpoint %s error.", name, b.svc.Name, b.endpoint.Name, httpError.Name),
		ClientArgs:          args,
		ReturnTypeName:      b.svc.Scope.GoFullTypeName(httpError.Attribute, pkg),
		ReturnTypeRef:       b.svc.Scope.GoFullTypeRef(httpError.Attribute, pkg),
		ReturnIsStruct:      expr.IsObject(httpError.Type),
		ReturnTypeAttribute: codegen.Goify(origin, true),
		ReturnTypePkg:       pkg,
		ClientCode:          code,
		SkipFieldInit:       skipFieldInit,
	}
}

func (b *errorBuilder) buildResultInitCode(errorResponse *transportir.ResponseStatus, errctx *codegen.AttributeContext, args []*InitArgData) (string, string, bool) {
	origin := ""
	httpError := errorResponse.Error
	body := responseStatusBody(errorResponse)
	if expr.IsDefaultErrorResult(httpError.Attribute.Type) {
		return buildProblemClientResultTransformCode(errorResponse, body.Type != expr.Empty, args), "", true
	}
	errAtt := httpError.Attribute
	if o, ok := body.Meta["origin:attribute"]; ok {
		origin = o[0]
		errAtt = expr.AsObject(httpError.Type).Attribute(origin)
	}
	code, err := b.sds.buildClientResultTransformCode(body, errAtt, httpError.Attribute, b.endpoint.Request, b.httpclictx, errctx, b.sd)
	if err != nil {
		fmt.Println(err.Error())
	}
	return code, origin, false
}

func (b *errorBuilder) buildResponseData(errorResponse *transportir.ResponseStatus, errctx *codegen.AttributeContext, init *InitData) *ResponseData {
	httpError := errorResponse.Error
	serverBodyData, clientBodyData := b.buildResponseBodyData(errorResponse)
	headers := b.sds.extractHeaders(errorResponse.Headers, httpError.Attribute, errctx, b.sd.Scope)
	cookies := b.sds.extractResponseCookies(errorResponse.Cookies, httpError.Attribute, errctx, b.sd.Scope)
	contentType := errorResponse.ContentType
	headerSourceVar := "res"
	problemTypeOverride := ""
	problemTitleOverride := ""
	if expr.IsDefaultErrorResult(httpError.Attribute.Type) {
		headerSourceVar = "problem"
		problemTypeOverride = quotedMetaValue(httpError.Attribute.Meta, "http:problem:type")
		problemTitleOverride = quotedMetaValue(httpError.Attribute.Meta, "http:problem:title")
	}
	return newResponseData(
		"",
		ResponseData{
			StatusCode:           statusCodeToHTTPConst(errorResponse.StatusCode),
			Code:                 errorResponse.StatusCode,
			Headers:              headers,
			ContentType:          contentType,
			Cookies:              cookies,
			ErrorHeader:          httpError.Name,
			ServerBody:           serverBodyData,
			ClientBody:           clientBodyData,
			ResultInit:           init,
			MustValidate:         responseFieldsNeedValidation(headers, cookies),
			HeaderSourceVar:      headerSourceVar,
			ProblemTypeOverride:  problemTypeOverride,
			ProblemTitleOverride: problemTitleOverride,
		},
	)
}

func (b *errorBuilder) buildResponseBodyData(errorResponse *transportir.ResponseStatus) ([]*TypeData, *TypeData) {
	httpError := errorResponse.Error
	errorLoc := b.method.ErrorLocs[httpError.Name]
	serverBodyData, clientBodyData := b.sds.buildResponseBodyPair(responseStatusBody(errorResponse), httpError.Attribute, errorLoc, b.endpoint.Name, b.sd)
	if expr.IsDefaultErrorResult(httpError.Attribute.Type) && len(serverBodyData) > 0 && serverBodyData[0] != nil && serverBodyData[0].Init != nil {
		serverBodyData[0].Init.ServerCode = buildProblemServerResponseBodyCode(serverBodyData[0].Ref, errorResponse)
	}
	if clientBodyData != nil {
		clientBodyData.Description = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body for the %q error.", clientBodyData.VarName, b.svc.Name, b.endpoint.Name, httpError.Name)
		serverBodyData[0].Description = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body for the %q error.", serverBodyData[0].VarName, b.svc.Name, b.endpoint.Name, httpError.Name)
	}
	return serverBodyData, clientBodyData
}

func (sds *ServicesData) buildResponsesFromIR(endpointIR *transportir.Endpoint, result *expr.AttributeExpr, viewed bool, sd *ServiceData) []*ResponseData {
	var (
		responses []*ResponseData

		svc        = sd.Service
		md         = svc.Method(endpointIR.Name)
		pkg        = service.DefaultPackageName(md.ResultLoc, svc.PkgName)
		httpclictx = httpContext(sd.Scope, false, false)
		scope      = svc.Scope
		svcctx     = serviceContext(pkg, sd.Service.Scope)
	)
	{
		if viewed {
			scope = svc.ViewScope
			svcctx = viewContext(sd.Service.ViewsPkg, sd.Service.ViewScope)
		}
		notag := -1
		for i, resp := range endpointIR.Response.Responses {
			if resp.TagName == "" {
				if notag > -1 {
					continue
				}
				notag = i
			}
			responses = append(responses, sds.buildSingleResponseData(endpointIR, resp, result, viewed, md, pkg, httpclictx, scope, svcctx, sd))
		}
		count := len(responses)
		if notag >= 0 && notag < count-1 {
			responses[notag], responses[count-1] = responses[count-1], responses[notag]
		}
	}
	return responses
}

func (sds *ServicesData) buildSingleResponseData(
	endpointIR *transportir.Endpoint,
	resp *transportir.ResponseStatus,
	result *expr.AttributeExpr,
	viewed bool,
	md *service.MethodData,
	pkg string,
	httpclictx *codegen.AttributeContext,
	scope *codegen.NameScope,
	svcctx *codegen.AttributeContext,
	sd *ServiceData,
) *ResponseData {
	headersData := sds.extractHeaders(resp.Headers, result, svcctx, scope)
	cookiesData := sds.extractResponseCookies(resp.Cookies, result, svcctx, scope)
	origin, resAttr := responseOriginAttribute(resp, result)
	serverBodyData, clientBodyData := sds.buildResponseBodyData(resp, result, origin, viewed, md, endpointIR, sd)
	init := sds.buildResponseResultInit(resp, result, resAttr, origin, viewed, md, pkg, httpclictx, scope, svcctx, headersData, cookiesData, endpointIR, sd)
	tagName, tagValue, tagPointer := responseTagData(resp, result, viewed)
	return newResponseData(
		resp.Description,
		ResponseData{
			StatusCode:   statusCodeToHTTPConst(resp.StatusCode),
			Headers:      headersData,
			Cookies:      cookiesData,
			ContentType:  resp.ContentType,
			ServerBody:   serverBodyData,
			ClientBody:   clientBodyData,
			ResultInit:   init,
			TagName:      tagName,
			TagValue:     tagValue,
			TagPointer:   tagPointer,
			MustValidate: responseFieldsNeedValidation(headersData, cookiesData),
			ResultAttr:   codegen.Goify(origin, true),
			ViewedResult: md.ViewedResult,
		},
	)
}

func responseOriginAttribute(resp *transportir.ResponseStatus, result *expr.AttributeExpr) (string, *expr.AttributeExpr) {
	if resp.Body == nil || resp.Body.Type == expr.Empty {
		return "", result
	}
	if resp.BodyOrigin != "" {
		return resp.BodyOrigin, expr.AsObject(result.Type).Attribute(resp.BodyOrigin)
	}
	return "", result
}

func (sds *ServicesData) buildResponseBodyData(
	resp *transportir.ResponseStatus,
	result *expr.AttributeExpr,
	origin string,
	viewed bool,
	md *service.MethodData,
	endpointIR *transportir.Endpoint,
	sd *ServiceData,
) ([]*TypeData, *TypeData) {
	if viewed {
		return sds.buildViewedResponseBodyData(resp, result, origin, md, endpointIR, sd)
	}
	return sds.buildResponseBodyPair(resp.Body, result, md.ResultLoc, endpointIR.Name, sd)
}

func (sds *ServicesData) buildViewedResponseBodyData(
	resp *transportir.ResponseStatus,
	result *expr.AttributeExpr,
	origin string,
	md *service.MethodData,
	endpointIR *transportir.Endpoint,
	sd *ServiceData,
) ([]*TypeData, *TypeData) {
	serverViews, clientView := viewedResponseBodyViews(origin, md, endpointIR.Response.Result)
	serverBodyData := make([]*TypeData, 0, len(serverViews))
	for _, view := range serverViews {
		viewName := view
		if sbd := sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, endpointIR.Name, true, viewName, sd); sbd != nil {
			serverBodyData = append(serverBodyData, sbd)
		}
	}
	clientBodyData := sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, endpointIR.Name, false, clientView, sd)
	registerClientBodyType(clientBodyData, sd)
	return serverBodyData, clientBodyData
}

func viewedResponseBodyViews(origin string, md *service.MethodData, methodResult *expr.AttributeExpr) ([]*string, *string) {
	vname := ""
	clientView := &vname
	if origin != "" {
		return []*string{&vname}, clientView
	}
	if v, ok := methodResult.Meta.Last(expr.ViewMetaKey); ok {
		return []*string{&v}, clientView
	}
	views := make([]*string, 0, len(md.ViewedResult.Views))
	for _, view := range md.ViewedResult.Views {
		viewName := view.Name
		views = append(views, &viewName)
	}
	return views, clientView
}

func (sds *ServicesData) buildResponseBodyPair(
	body, target *expr.AttributeExpr,
	loc *codegen.Location,
	endpointName string,
	sd *ServiceData,
) ([]*TypeData, *TypeData) {
	var serverBodyData []*TypeData
	if sbd := sds.buildResponseBodyType(body, target, loc, endpointName, true, nil, sd); sbd != nil {
		serverBodyData = append(serverBodyData, sbd)
	}
	clientBodyData := sds.buildResponseBodyType(body, target, loc, endpointName, false, nil, sd)
	registerClientBodyType(clientBodyData, sd)
	return serverBodyData, clientBodyData
}

func registerClientBodyType(clientBodyData *TypeData, sd *ServiceData) {
	if clientBodyData == nil || clientBodyData.Def == "" {
		return
	}
	sd.ClientTypeNames[clientBodyData.Name] = false
}

func (sds *ServicesData) buildResponseResultInit(
	resp *transportir.ResponseStatus,
	result *expr.AttributeExpr,
	resAttr *expr.AttributeExpr,
	origin string,
	viewed bool,
	md *service.MethodData,
	pkg string,
	httpclictx *codegen.AttributeContext,
	scope *codegen.NameScope,
	svcctx *codegen.AttributeContext,
	headersData []*HeaderData,
	cookiesData []*CookieData,
	endpointIR *transportir.Endpoint,
	sd *ServiceData,
) *InitData {
	if !needInit(result.Type) {
		return nil
	}
	tname := sd.Service.Scope.GoFullTypeName(result, pkg)
	tref := sd.Service.Scope.GoFullTypeRef(result, pkg)
	if viewed {
		tname = sd.Service.ViewScope.GoFullTypeName(result, sd.Service.ViewsPkg)
		tref = sd.Service.ViewScope.GoFullTypeRef(result, sd.Service.ViewsPkg)
	}
	status := codegen.Goify(http.StatusText(resp.StatusCode), true)
	n := codegen.Goify(md.Name, true)
	r := codegen.Goify(md.Result, true)
	if strings.HasPrefix(r, n) {
		r = scope.HashedUnique(result.Type, r)
	}
	name := fmt.Sprintf("New%s%s%s", n, r, status)
	if strings.HasPrefix(codegen.Goify(md.Result, true), n) {
		name = fmt.Sprintf("New%s%s", r, status)
	}
	code, pointer, clientArgs := sds.buildResponseResultInitCode(resp, result, resAttr, origin, httpclictx, svcctx, headersData, cookiesData, endpointIR, sd)
	return &InitData{
		Name:                     name,
		Description:              fmt.Sprintf("%s builds a %q service %q endpoint result from a HTTP %q response.", name, sd.Service.Name, endpointIR.Name, status),
		ClientArgs:               clientArgs,
		ReturnTypeName:           tname,
		ReturnTypeRef:            tref,
		ReturnIsStruct:           expr.IsObject(result.Type),
		ReturnTypeAttribute:      codegen.Goify(origin, true),
		ReturnTypePkg:            pkg,
		ReturnIsPrimitivePointer: pointer,
		ClientCode:               code,
	}
}

func (sds *ServicesData) buildResponseResultInitCode(
	resp *transportir.ResponseStatus,
	result *expr.AttributeExpr,
	resAttr *expr.AttributeExpr,
	origin string,
	httpclictx *codegen.AttributeContext,
	svcctx *codegen.AttributeContext,
	headersData []*HeaderData,
	cookiesData []*CookieData,
	endpointIR *transportir.Endpoint,
	sd *ServiceData,
) (string, bool, []*InitArgData) {
	clientArgs := buildResponseResultInitArgs(resp, httpclictx, headersData, cookiesData, sd)
	code, err := sds.buildClientResultTransformCode(resp.Body, resAttr, result, endpointIR.Request, httpclictx, svcctx, sd)
	if err != nil {
		fmt.Println(err.Error())
	}
	clientArgs = append(clientArgs, buildHeaderInitArgs(headersData)...)
	clientArgs = append(clientArgs, buildCookieInitArgs(cookiesData)...)
	return code, buildResponseResultPointer(resp, result, origin), clientArgs
}

func buildResponseResultInitArgs(
	resp *transportir.ResponseStatus,
	httpclictx *codegen.AttributeContext,
	headersData []*HeaderData,
	cookiesData []*CookieData,
	sd *ServiceData,
) []*InitArgData {
	clientArgs := make([]*InitArgData, 0, 1+len(headersData)+len(cookiesData))
	if resp.Body.Type == expr.Empty {
		return clientArgs
	}
	bodyArg := buildBodyInitArg(sd.Scope, resp.Body, true)
	bodyArg.AttributeData.Validate = validationCodeForBodyArg(resp.Body, httpclictx)
	return append(clientArgs, bodyArg)
}

func buildResponseResultPointer(resp *transportir.ResponseStatus, result *expr.AttributeExpr, origin string) bool {
	if resp.Body.Type == expr.Empty || expr.IsObject(resp.Body.Type) {
		return false
	}
	if origin == "" {
		return false
	}
	return result.IsPrimitivePointer(origin, true)
}

func responseTagData(resp *transportir.ResponseStatus, result *expr.AttributeExpr, viewed bool) (string, string, bool) {
	if resp.TagName == "" {
		return "", "", false
	}
	return codegen.Goify(resp.TagName, true), resp.TagValue, viewed || result.IsPrimitivePointer(resp.TagName, true)
}

func validationCodeForBodyArg(body *expr.AttributeExpr, httpclictx *codegen.AttributeContext) string {
	ut, ok := body.Type.(expr.UserType)
	if !ok || ut.Attribute().Validation == nil {
		return ""
	}
	return codegen.ValidationCode(ut.Attribute(), ut, httpclictx, true, expr.IsAlias(ut), false, "body")
}

func (sds *ServicesData) buildClientResultTransformCode(
	body *expr.AttributeExpr,
	bodyTarget *expr.AttributeExpr,
	fallbackTarget *expr.AttributeExpr,
	request *transportir.Request,
	httpclictx *codegen.AttributeContext,
	targetctx *codegen.AttributeContext,
	sd *ServiceData,
) (string, error) {
	if body.Type != expr.Empty {
		return sds.buildClientBodyUnmarshalCode(body, bodyTarget, httpclictx, targetctx, sd)
	}
	if !expr.IsArray(fallbackTarget.Type) && !expr.IsMap(fallbackTarget.Type) {
		return "", nil
	}
	fallbackSource, fallbackVar := buildQueryFallbackSource(request)
	if fallbackSource == nil {
		return "", nil
	}
	return sds.buildClientBodyUnmarshalCode(fallbackSource, fallbackTarget, httpclictx, targetctx, sd, fallbackVar)
}

func (sds *ServicesData) buildClientBodyUnmarshalCode(
	source *expr.AttributeExpr,
	target *expr.AttributeExpr,
	httpclictx *codegen.AttributeContext,
	targetctx *codegen.AttributeContext,
	sd *ServiceData,
	sourceVar ...string,
) (string, error) {
	src := "body"
	if len(sourceVar) > 0 && sourceVar[0] != "" {
		src = sourceVar[0]
	}
	code, helpers, err := unmarshal(source, target, src, httpclictx, targetctx)
	if err != nil {
		return "", err
	}
	sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
	return code, nil
}

func buildQueryFallbackSource(request *transportir.Request) (*expr.AttributeExpr, string) {
	if request == nil || len(request.QueryParams) == 0 {
		return nil, ""
	}
	query := request.QueryParams[0]
	return query.Attribute, codegen.Goify(query.Name, false)
}

func responseStatusBody(resp *transportir.ResponseStatus) *expr.AttributeExpr {
	if resp == nil || resp.Body == nil {
		return &expr.AttributeExpr{Type: expr.Empty}
	}
	return resp.Body
}

func newResponseData(description string, data ResponseData) *ResponseData {
	data.Description = description
	return &data
}

func buildProblemServerResponseBodyCode(responseBodyRef string, errorResponse *transportir.ResponseStatus) string {
	problemType := quotedMetaValue(errorResponse.Error.Attribute.Meta, "http:problem:type")
	problemTitle := quotedMetaValue(errorResponse.Error.Attribute.Meta, "http:problem:title")
	responseBodyLiteralType := strings.TrimPrefix(responseBodyRef, "*")
	return fmt.Sprintf(`problemType, problemTitle := loomhttp.ResolveProblemTypeAndTitle(res.Name, %d, %s, %s)
body := &%s{
	Type:     problemType,
	Title:    problemTitle,
	Status:   %d,
	Detail:   loom.ErrorSafeMessage(res),
	Instance: loomhttp.ProblemInstanceURI(res.ID),
	Code:     res.Name,
}
if retryHint := loom.ErrorRetryHint(res); retryHint != "" {
	body.RetryHint = &retryHint
}`, errorResponse.StatusCode, problemType, problemTitle, responseBodyLiteralType, errorResponse.StatusCode)
}

func buildProblemClientResultTransformCode(errorResponse *transportir.ResponseStatus, hasBody bool, args []*InitArgData) string {
	codeExpr := `""`
	detailExpr := `""`
	instanceExpr := `""`
	retryHintExpr := `nil`
	if hasBody {
		return fmt.Sprintf(`code := ""
if body.Code != nil {
	code = *body.Code
}
detail := ""
if body.Detail != nil {
	detail = *body.Detail
}
instance := ""
if body.Instance != nil {
	instance = *body.Instance
}
v := loomhttp.ProblemErrorFromBody(code, %d, detail, instance, body.RetryHint)`, errorResponse.StatusCode)
	} else {
		if v, ok := findInitArgVar(args, "code"); ok {
			codeExpr = v
		}
		if v, ok := findInitArgVar(args, "detail"); ok {
			detailExpr = v
		}
		if v, ok := findInitArgVar(args, "instance"); ok {
			instanceExpr = v
		}
		if v, ok := findInitArgVar(args, "retry_hint"); ok {
			retryHintExpr = v
		}
	}
	return fmt.Sprintf(`v := loomhttp.ProblemErrorFromBody(%s, %d, %s, %s, %s)`, codeExpr, errorResponse.StatusCode, detailExpr, instanceExpr, retryHintExpr)
}

func findInitArgVar(args []*InitArgData, names ...string) (string, bool) {
	for _, arg := range args {
		for _, name := range names {
			goName := codegen.Goify(name, false)
			goField := codegen.Goify(name, true)
			if arg.Name == name || arg.VarName == name || arg.VarName == goName || arg.FieldName == goField {
				return arg.VarName, true
			}
		}
	}
	return "", false
}

func quotedMetaValue(meta expr.MetaExpr, key string) string {
	if meta == nil {
		return `""`
	}
	values := meta[key]
	if len(values) == 0 {
		return `""`
	}
	return fmt.Sprintf("%q", values[0])
}
