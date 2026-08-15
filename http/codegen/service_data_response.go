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

type resultBuilder struct {
	sds      *ServicesData
	endpoint *transportir.Endpoint
	sd       *ServiceData
	svc      *service.Data
	method   *service.MethodData
	pkg      string
	result   *expr.AttributeExpr
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
		IsAny:               isAnyType(result.Type),
		Name:                resultDesc.Declared.Name,
		Ref:                 resultDesc.Declared.Ref,
		IDAttribute:         idAtt,
		IDAttributeRequired: idAttRequired,
		Responses:           responses,
		View:                resultDesc.View,
		MustInit:            mustInit,
	}
}

func isAnyType(dataType expr.DataType) bool {
	switch actual := dataType.(type) {
	case expr.Primitive:
		return actual.Kind() == expr.AnyKind
	case expr.UserType:
		return isAnyType(actual.Attribute().Type)
	default:
		return false
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
			Code:         resp.StatusCode,
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
			DeferStatus:  endpointIR.Response.FileResponse,
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
		serverView := v
		clientName := v
		return []*string{&serverView}, &clientName
	}
	views := make([]*string, 0, len(md.ViewedResult.Views))
	for _, view := range md.ViewedResult.Views {
		viewName := view.Name
		views = append(views, &viewName)
	}
	if len(md.ViewedResult.Views) == 1 {
		single := md.ViewedResult.Views[0].Name
		clientView = &single
	}
	return views, clientView
}

// clientResponseViewName returns the response view used by client code
// generation when the design fixes the response to a single view. An empty
// string means the client must keep the unprojected transport body because the
// server may render multiple views.
func clientResponseViewName(methodResult *expr.AttributeExpr, md *service.MethodData) string {
	if md == nil || md.ViewedResult == nil {
		return ""
	}
	if methodResult != nil {
		if v, ok := methodResult.Meta.Last(expr.ViewMetaKey); ok {
			return v
		}
	}
	if len(md.ViewedResult.Views) == 1 {
		return md.ViewedResult.Views[0].Name
	}
	return ""
}

// effectiveClientResponseBody returns the response body shape used by client
// code generation. When the design fixes the response to a single view, the
// returned attribute uses that projected ResultType so type collection, union
// collection, and client decode/init all agree on one transport body.
func effectiveClientResponseBody(body, methodResult *expr.AttributeExpr, md *service.MethodData) *expr.AttributeExpr {
	if body == nil {
		return body
	}
	view := clientResponseViewName(methodResult, md)
	if view == "" {
		return body
	}
	body = expr.DupAtt(body)
	rt, ok := body.Type.(*expr.ResultTypeExpr)
	if !ok {
		return body
	}
	projected, err := expr.Project(rt, view)
	if err != nil {
		panic(codegen.NewError(nil, body, fmt.Errorf("project effective client response body view %q: %w", view, err)))
	}
	body.Type = projected
	return body
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
	if !needInit(result) {
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
	code, pointer, clientArgs := sds.buildResponseResultInitCode(resp, result, resAttr, origin, md, httpclictx, svcctx, headersData, cookiesData, endpointIR, sd)
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
	md *service.MethodData,
	httpclictx *codegen.AttributeContext,
	svcctx *codegen.AttributeContext,
	headersData []*HeaderData,
	cookiesData []*CookieData,
	endpointIR *transportir.Endpoint,
	sd *ServiceData,
) (string, bool, []*InitArgData) {
	body := effectiveClientResponseBody(resp.Body, endpointIR.Response.Result, md)
	clientArgs := buildResponseResultInitArgs(resp, body, httpclictx, headersData, cookiesData, sd)
	code, err := sds.buildClientResultTransformCode(body, resAttr, result, endpointIR.Request, httpclictx, svcctx, sd)
	if err != nil {
		panic(codegen.NewError(nil, body, fmt.Errorf("build HTTP response result transform: %w", err)))
	}
	clientArgs = append(clientArgs, buildHeaderInitArgs(headersData)...)
	clientArgs = append(clientArgs, buildCookieInitArgs(cookiesData)...)
	return code, buildResponseResultPointer(resp, result, origin), clientArgs
}

func buildResponseResultInitArgs(
	resp *transportir.ResponseStatus,
	body *expr.AttributeExpr,
	httpclictx *codegen.AttributeContext,
	headersData []*HeaderData,
	cookiesData []*CookieData,
	sd *ServiceData,
) []*InitArgData {
	clientArgs := make([]*InitArgData, 0, 1+len(headersData)+len(cookiesData))
	if resp.Body.Type == expr.Empty {
		return clientArgs
	}
	bodyArg := buildBodyInitArg(sd.Scope, body, true)
	bodyArg.AttributeData.Validate = validationCodeForBodyArg(body, httpclictx)
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
	data.EncodePlan = newResponseEncodePlan(&data)
	return &data
}

func newResponseEncodePlan(response *ResponseData) *ResponseEncodePlan {
	bodyCount := len(response.ServerBody)
	var firstBody *TypeData
	if bodyCount > 0 {
		firstBody = response.ServerBody[0]
	}
	return &ResponseEncodePlan{
		BodyCount:         bodyCount,
		HasBody:           bodyCount > 0,
		FirstBody:         firstBody,
		HasMultipleBodies: bodyCount > 1,
		UseViewedBodySwitch: bodyCount > 1 &&
			response.ViewedResult != nil,
		NeedsProblemSource: response.HeaderSourceVar == "problem" &&
			(len(response.Headers) > 0 || len(response.Cookies) > 0),
	}
}
