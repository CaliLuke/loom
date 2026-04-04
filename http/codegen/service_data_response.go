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

func (sds *ServicesData) buildResponsesFromIR(endpointIR *transportir.Endpoint, result *expr.AttributeExpr, viewed bool, sd *ServiceData) []*ResponseData {
	var (
		responses []*ResponseData

		svc        = sd.Service
		md         = svc.Method(endpointIR.Name)
		pkg        = pkgWithDefault(md.ResultLoc, svc.PkgName)
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

// buildErrorsData builds the error data for all the error responses in the
// endpoint expression. The response headers, cookies and body for each response
// are inferred from the method's error expression if not specified explicitly.
func (sds *ServicesData) buildErrorsData(e *expr.HTTPEndpointExpr, sd *ServiceData) []*ErrorGroupData {
	return sds.buildErrorsDataFromIR(transportir.BuildEndpoint(e), sd)
}

func (sds *ServicesData) buildErrorsDataFromIR(endpointIR *transportir.Endpoint, sd *ServiceData) []*ErrorGroupData {
	var (
		svc        = sd.Service
		ep         = svc.Method(endpointIR.Name)
		httpclictx = httpContext(sd.Scope, false, false)
	)

	data := make(map[string][]*ErrorData)
	for _, errorResponse := range endpointIR.Response.ErrorResponses {
		ref, errorData := sds.buildSingleErrorDataFromIR(endpointIR, errorResponse, ep, svc, httpclictx, sd)
		data[ref] = append(data[ref], errorData)
	}
	keys := make([]string, len(data))
	i := 0
	for k := range data {
		keys[i] = k
		i++
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

func (sds *ServicesData) buildSingleErrorDataFromIR(
	endpointIR *transportir.Endpoint,
	errorResponse *transportir.ResponseStatus,
	ep *service.MethodData,
	svc *service.Data,
	httpclictx *codegen.AttributeContext,
	sd *ServiceData,
) (string, *ErrorData) {
	httpError := errorResponse.Error
	pkg := pkgWithDefault(ep.ErrorLocs[httpError.Name], svc.PkgName)
	errctx := serviceContext(pkg, sd.Service.Scope)
	init := sds.buildErrorResultInit(endpointIR, errorResponse, ep, pkg, httpclictx, errctx, svc, sd)
	responseData := sds.buildErrorResponseData(endpointIR, errorResponse, ep, errctx, init, svc, sd)
	ref := svc.Scope.GoFullTypeRef(httpError.AttributeExpr, pkg)
	return ref, &ErrorData{Name: httpError.Name, Response: responseData, Ref: ref}
}

func (sds *ServicesData) buildErrorResultInit(
	endpointIR *transportir.Endpoint,
	errorResponse *transportir.ResponseStatus,
	ep *service.MethodData,
	pkg string,
	httpclictx *codegen.AttributeContext,
	errctx *codegen.AttributeContext,
	svc *service.Data,
	sd *ServiceData,
) *InitData {
	httpError := errorResponse.Error
	body := responseStatusBody(errorResponse).Type
	if !needInit(httpError.Type) {
		return nil
	}
	headers := sds.extractHeaders(errorResponse.Headers, httpError.AttributeExpr, errctx, sd.Scope)
	cookies := sds.extractResponseCookies(errorResponse.Cookies, httpError.AttributeExpr, errctx, sd.Scope)
	args := make([]*InitArgData, 0, len(headers)+len(cookies)+1)
	if body != expr.Empty {
		args = append(args, buildBodyInitArg(sd.Scope, responseStatusBody(errorResponse), true))
	}
	args = append(args, buildHeaderInitArgs(headers)...)
	args = append(args, buildCookieInitArgs(cookies)...)
	code, origin := sds.buildErrorResultInitCode(endpointIR, errorResponse, httpclictx, errctx, sd)
	name := fmt.Sprintf("New%s%s", codegen.Goify(ep.Name, true), codegen.Goify(httpError.ErrorExpr.Name, true))
	return &InitData{
		Name:                name,
		Description:         fmt.Sprintf("%s builds a %s service %s endpoint %s error.", name, svc.Name, endpointIR.Name, httpError.ErrorExpr.Name),
		ClientArgs:          args,
		ReturnTypeName:      svc.Scope.GoFullTypeName(httpError.AttributeExpr, pkg),
		ReturnTypeRef:       svc.Scope.GoFullTypeRef(httpError.AttributeExpr, pkg),
		ReturnIsStruct:      expr.IsObject(httpError.Type),
		ReturnTypeAttribute: codegen.Goify(origin, true),
		ReturnTypePkg:       pkg,
		ClientCode:          code,
	}
}

func (sds *ServicesData) buildErrorResultInitCode(
	endpointIR *transportir.Endpoint,
	errorResponse *transportir.ResponseStatus,
	httpclictx *codegen.AttributeContext,
	errctx *codegen.AttributeContext,
	sd *ServiceData,
) (string, string) {
	origin := ""
	httpError := errorResponse.Error
	body := responseStatusBody(errorResponse)
	errAtt := httpError.AttributeExpr
	if o, ok := body.Meta["origin:attribute"]; ok {
		origin = o[0]
		errAtt = expr.AsObject(httpError.ErrorExpr.Type).Attribute(origin)
	}
	code, err := sds.buildClientResultTransformCode(body, errAtt, httpError.AttributeExpr, endpointIR.Request, httpclictx, errctx, sd)
	if err != nil {
		fmt.Println(err.Error())
	}
	return code, origin
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

func (sds *ServicesData) buildErrorResponseData(
	endpointIR *transportir.Endpoint,
	errorResponse *transportir.ResponseStatus,
	ep *service.MethodData,
	errctx *codegen.AttributeContext,
	init *InitData,
	svc *service.Data,
	sd *ServiceData,
) *ResponseData {
	httpError := errorResponse.Error
	serverBodyData, clientBodyData := sds.buildErrorResponseBodyData(endpointIR, errorResponse, ep, svc, sd)
	headers := sds.extractHeaders(errorResponse.Headers, httpError.AttributeExpr, errctx, sd.Scope)
	cookies := sds.extractResponseCookies(errorResponse.Cookies, httpError.AttributeExpr, errctx, sd.Scope)
	contentType := ""
	if errorResponse.ContentType != expr.ErrorResultIdentifier {
		contentType = errorResponse.ContentType
	}
	return newResponseData(
		"",
		ResponseData{
			StatusCode:   statusCodeToHTTPConst(errorResponse.StatusCode),
			Code:         errorResponse.StatusCode,
			Headers:      headers,
			ContentType:  contentType,
			Cookies:      cookies,
			ErrorHeader:  httpError.Name,
			ServerBody:   serverBodyData,
			ClientBody:   clientBodyData,
			ResultInit:   init,
			MustValidate: responseFieldsNeedValidation(headers, cookies),
		},
	)
}

func (sds *ServicesData) buildErrorResponseBodyData(
	endpointIR *transportir.Endpoint,
	errorResponse *transportir.ResponseStatus,
	ep *service.MethodData,
	svc *service.Data,
	sd *ServiceData,
) ([]*TypeData, *TypeData) {
	httpError := errorResponse.Error
	errorLoc := ep.ErrorLocs[httpError.ErrorExpr.Name]
	serverBodyData, clientBodyData := sds.buildResponseBodyPair(responseStatusBody(errorResponse), httpError.AttributeExpr, errorLoc, endpointIR.Name, sd)
	if clientBodyData != nil {
		clientBodyData.Description = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body for the %q error.", clientBodyData.VarName, svc.Name, endpointIR.Name, httpError.Name)
		serverBodyData[0].Description = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body for the %q error.", serverBodyData[0].VarName, svc.Name, endpointIR.Name, httpError.Name)
	}
	return serverBodyData, clientBodyData
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
