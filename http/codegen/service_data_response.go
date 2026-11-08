package codegen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/representation"
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
		pkg:      svc.LocationPackageName(method.ResultLoc),
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
		if r.Unary != nil || len(r.ServerBody) > 0 || len(r.Headers) > 0 || len(r.Cookies) > 0 || r.Tag != nil || r.ClientStatusField != nil {
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
		responses = make([]*ResponseData, 0, len(endpointIR.Response.Responses))

		svc    = sd.Service
		md     = svc.Method(endpointIR.Name)
		pkg    = svc.LocationPackageName(md.ResultLoc)
		svcctx = serviceContext(pkg, sd.Service.Scope)
	)
	if viewed {
		svcctx = viewContext(sd.Service.ViewsPkg, sd.Service.ViewScope)
	}
	for _, resp := range endpointIR.Response.Responses {
		responses = append(responses, sds.buildSingleResponseData(endpointIR, resp, result, viewed, md, svcctx, sd))
	}
	return responses
}

func (sds *ServicesData) buildSingleResponseData(
	endpointIR *transportir.Endpoint,
	resp *transportir.ResponseStatus,
	result *expr.AttributeExpr,
	viewed bool,
	md *service.MethodData,
	svcctx *codegen.AttributeContext,
	sd *ServiceData,
) *ResponseData {
	unaryResult, unaryViewed, unaryContext := result, viewed, svcctx
	metadata := resp.Handshake || resp.Selector.Phase() == expr.PhaseInformational
	if metadata {
		result, viewed = resp.Carrier, false
		pkg := sd.Service.LocationPackageName(codegen.UserTypeLocation(result.Type))
		svcctx = serviceContext(pkg, sd.Service.Scope)
	}
	vars := newResponseVarScope(sd)
	headersData := sds.extractHeaders(resp.Headers, result, svcctx, svcctx.Scope.Scope(), vars, sds.examplesFor(sd))
	cookiesData := sds.extractResponseCookies(resp.Cookies, result, svcctx, svcctx.Scope.Scope(), vars, sds.examplesFor(sd))
	field := selectBodyField(result, resp.Body)
	origin := field.Name
	serverBodyData, clientBodyData := sds.buildResponseBodyData(resp, result, origin, viewed, md, endpointIR, sd)
	for _, body := range serverBodyData {
		attachMediaBodyData(body, resp.MediaVariants)
	}
	attachMediaBodyData(clientBodyData, resp.MediaVariants)
	init := sds.buildResponseResultInit(resp, result, field, md, clientBodyData, svcctx, headersData, cookiesData, endpointIR, sd)
	tag := responseTagData(resp.Tag, result, viewed, svcctx)
	data := newResponseData(
		resp,
		ResponseData{
			nullableProjection: newNullableResponseProjection(result, resp.Body, headersData, cookiesData, svcctx, viewed, resp.StatusField != nil),
			serverContext:      svcctx,
			Description:        resp.Description,
			Headers:            headersData,
			Cookies:            cookiesData,
			ContentType:        resp.ContentType,
			ServerBody:         serverBodyData,
			ClientBody:         clientBodyData,
			ResultInit:         init,
			Tag:                tag,
			MustValidate:       responseFieldsNeedValidation(headersData, cookiesData),
			ResultAttr:         field.GoName,
			ViewedResult:       responseViewedResult(md, metadata),
			DeferStatus:        endpointIR.Response.FileResponse,
		},
	)
	if resp.Unary != nil {
		unary := *resp
		unary.Unary, unary.StatusField, unary.Headers, unary.Cookies = nil, nil, nil, nil
		unary.Handshake, unary.Carrier, unary.BodyOrigin = false, unaryResult, ""
		unary.Body, unary.BodyValue, unary.BodyValues = resp.Unary.Body, resp.Unary.BodyValue, resp.Unary.BodyValues
		unary.MediaVariants, unary.ContentType, unary.ContentTypes = resp.Unary.MediaVariants, resp.Unary.ContentType, resp.Unary.ContentTypes
		data.Unary = sds.buildSingleResponseData(endpointIR, &unary, unaryResult, unaryViewed, md, unaryContext, sd)
		data.Unary.DeferStatus = true
	}
	return data
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
	loc := md.ResultLoc
	if resp.Handshake || resp.Selector.Phase() == expr.PhaseInformational {
		loc = codegen.UserTypeLocation(result.Type)
	}
	return sds.buildResponseBodyPair(responseStatusBody(resp), result, loc, endpointIR.Name, sd, resp.BodyValue)
}

func (sds *ServicesData) buildViewedResponseBodyData(
	resp *transportir.ResponseStatus,
	result *expr.AttributeExpr,
	origin string,
	md *service.MethodData,
	endpointIR *transportir.Endpoint,
	sd *ServiceData,
) ([]*TypeData, *TypeData) {
	serverViews, clientView := viewedResponseBodyViews(origin, md.ViewedResult, endpointIR.Response.Result)
	serverBodyData := make([]*TypeData, 0, len(serverViews))
	for _, view := range serverViews {
		viewName := view
		if sbd := sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, endpointIR.Name, true, viewName, sd, resp.BodyValue); sbd != nil {
			serverBodyData = append(serverBodyData, sbd)
		}
	}
	clientBodyData := sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, endpointIR.Name, false, clientView, sd, resp.BodyValue)
	registerClientBodyType(clientBodyData, sd)
	return serverBodyData, clientBodyData
}

func viewedResponseBodyViews(origin string, viewed *service.ViewedResultTypeData, methodResult *expr.AttributeExpr) ([]*string, *string) {
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
	views := make([]*string, 0, len(viewed.Views))
	for _, view := range viewed.Views {
		viewName := view.Name
		views = append(views, &viewName)
	}
	if len(viewed.Views) == 1 {
		single := viewed.Views[0].Name
		clientView = &single
	}
	return views, clientView
}

func (sds *ServicesData) buildResponseBodyPair(
	body, target *expr.AttributeExpr,
	loc *codegen.Location,
	endpointName string,
	sd *ServiceData,
	value *transportir.ValueTarget,
) ([]*TypeData, *TypeData) {
	var serverBodyData []*TypeData
	if sbd := sds.buildResponseBodyType(body, target, loc, endpointName, true, nil, sd, value); sbd != nil {
		serverBodyData = append(serverBodyData, sbd)
	}
	clientBodyData := sds.buildResponseBodyType(body, target, loc, endpointName, false, nil, sd, value)
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
	field selectedServiceField,
	md *service.MethodData,
	clientBody *TypeData,
	svcctx *codegen.AttributeContext,
	headersData []*HeaderData,
	cookiesData []*CookieData,
	endpointIR *transportir.Endpoint,
	sd *ServiceData,
) *InitData {
	if !needInit(result) && !expr.ContainsNonNullableCollectionElement(responseStatusBody(resp)) {
		return nil
	}
	scope := svcctx.Scope.Scope()
	tname := scope.GoFullTypeName(result, svcctx.DefaultPkg)
	tref := scope.GoFullTypeRef(result, svcctx.DefaultPkg)
	status := codegen.Goify(resp.Selector.NameSuffix(), true)
	n := codegen.Goify(md.Name, true)
	resultName := md.Result
	if resp.Handshake || resp.Selector.Phase() == expr.PhaseInformational {
		resultName = scope.GoTypeName(result)
	}
	r := codegen.Goify(resultName, true)
	if strings.HasPrefix(r, n) {
		r = scope.HashedUnique(result.Type, r)
	}
	name := fmt.Sprintf("New%s%s%s", n, r, status)
	if strings.HasPrefix(codegen.Goify(resultName, true), n) {
		name = fmt.Sprintf("New%s%s", r, status)
	}
	code, pointer, clientArgs := sds.buildResponseResultInitCode(resp, result, field.Attribute, field.Name, clientBody, svcctx, headersData, cookiesData, sd)
	return &InitData{
		returnContext:            svcctx,
		Name:                     name,
		Description:              fmt.Sprintf("%s builds a %q service %q endpoint result from a HTTP %q response.", name, sd.Service.Name, endpointIR.Name, status),
		ClientArgs:               clientArgs,
		ReturnTypeName:           tname,
		ReturnTypeRef:            tref,
		ReturnIsStruct:           expr.IsObject(result.Type),
		ReturnTypeAttribute:      field.GoName,
		ReturnTypePkg:            svcctx.DefaultPkg,
		ReturnIsPrimitivePointer: pointer,
		ReturnIsUnionValue:       bodyFieldIsUnionValue(result, field.Name),
		ClientCode:               code.Code,
		ClientErrorAware:         code.ErrorAware,
	}
}

func (sds *ServicesData) buildResponseResultInitCode(
	resp *transportir.ResponseStatus,
	result *expr.AttributeExpr,
	resAttr *expr.AttributeExpr,
	origin string,
	clientBody *TypeData,
	svcctx *codegen.AttributeContext,
	headersData []*HeaderData,
	cookiesData []*CookieData,
	sd *ServiceData,
) (codegen.TransformPlan, bool, []*InitArgData) {
	clientArgs := make([]*InitArgData, 0, 1+len(headersData)+len(cookiesData))
	if clientBody != nil {
		bodyArg := buildBodyInitArg(clientBody)
		bodyArg.Validate = validationCodeForBodyArg(bodyArg.attribute, bodyArg.context)
		clientArgs = append(clientArgs, bodyArg)
	}
	clientArgs = append(clientArgs, buildHeaderInitArgs(headersData)...)
	clientArgs = append(clientArgs, buildCookieInitArgs(cookiesData)...)
	clientArgs = appendResponseStatusArg(clientArgs, resp, result, svcctx, sd)
	code, err := sds.buildClientResultTransformCode(clientBody, resAttr, result, clientArgs, svcctx, sd)
	if err != nil {
		panic(codegen.NewError(nil, resp.Body, fmt.Errorf("build HTTP response result transform: %w", err)))
	}
	return code, buildResponseResultPointer(resp, result, origin, svcctx), clientArgs
}

func buildResponseResultPointer(resp *transportir.ResponseStatus, result *expr.AttributeExpr, origin string, targetctx *codegen.AttributeContext) bool {
	if responseStatusBody(resp).Type == expr.Empty || expr.IsObject(responseStatusBody(resp).Type) {
		return false
	}
	if origin == "" {
		return false
	}
	return targetctx.IsPrimitivePointer(origin, result)
}

func validationCodeForBodyArg(body *expr.AttributeExpr, httpclictx *codegen.AttributeContext) string {
	ut, ok := body.Type.(expr.UserType)
	if !ok || ut.Attribute().Validation == nil {
		return ""
	}
	return codegen.ValidationCode(ut.Attribute(), ut, httpclictx, true, expr.IsAlias(ut), false, "body")
}

func (sds *ServicesData) buildClientResultTransformCode(
	body *TypeData,
	bodyTarget *expr.AttributeExpr,
	fallbackTarget *expr.AttributeExpr,
	args []*InitArgData,
	targetctx *codegen.AttributeContext,
	sd *ServiceData,
) (codegen.TransformPlan, error) {
	if body != nil {
		return sds.buildClientResultUnmarshalCode(body.attribute, bodyTarget, "body", body.context, targetctx, sd)
	}
	if !expr.IsArray(fallbackTarget.Type) && !expr.IsMap(fallbackTarget.Type) {
		return codegen.TransformPlan{}, nil
	}
	if len(args) != 1 {
		return codegen.TransformPlan{}, nil
	}
	// A bodyless collection comes from its sole decoded native binding, whose
	// physical representation is independent of JSON body layout.
	argument := args[0]
	return sds.buildClientResultUnmarshalCode(argument.attribute, fallbackTarget, argument.VarName, argument.context, targetctx, sd)
}

func (sds *ServicesData) buildClientResultUnmarshalCode(
	source, target *expr.AttributeExpr,
	sourceVar string,
	sourcectx, targetctx *codegen.AttributeContext,
	sd *ServiceData,
) (codegen.TransformPlan, error) {
	var plan codegen.TransformPlan
	var err error
	if nullableResponseRequiresConcrete(target, source) && selectBodyField(target, source).Name == "" {
		concrete := representation.ConcreteAttribute(target)
		plan, err = codegen.GoTransform(source, concrete, sourceVar, "responseValue", sourcectx.Dup(), targetctx, "unmarshal", true)
		if err == nil {
			ref := targetctx.Scope.Ref(target, targetctx.DefaultPkg)
			value := "responseValue"
			guard := ""
			if strings.HasPrefix(targetctx.Scope.Ref(concrete, targetctx.DefaultPkg), "*") {
				value = "*responseValue"
				guard = "if responseValue == nil {\n*transformErr = loom.ErrAbsentNullable\n} else "
			}
			plan.Code += fmt.Sprintf("\nvar v %s\nif *transformErr == nil {\n%s{\nv.SetValue(%s)\n}\n}\n", ref, guard, value)
			plan.ErrorAware = true
		}
	} else {
		plan, err = unmarshal(source, target, sourceVar, sourcectx.Dup(), targetctx)
	}
	if err != nil {
		return codegen.TransformPlan{}, err
	}
	sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, plan.Helpers)
	return plan, nil
}

func responseStatusBody(resp *transportir.ResponseStatus) *expr.AttributeExpr {
	if resp == nil || resp.Body == nil {
		return &expr.AttributeExpr{Type: expr.Empty}
	}
	return resp.Body
}

func newResponseData(response *transportir.ResponseStatus, data ResponseData) *ResponseData {
	data.Selector, data.Domain = response.Selector, slices.Clone(response.Domain)
	data.Carrier, data.Handshake = response.Carrier, response.Handshake
	data.statusBinding = response.StatusField
	data.Required = make(map[string][]string, len(response.Required))
	for view, required := range response.Required {
		data.Required[view] = slices.Clone(required)
	}
	if response.StatusField != nil {
		if data.ResultInit == nil || len(data.ResultInit.ClientArgs) == 0 {
			panic("HTTP status binding has no native constructor argument")
		}
		data.ClientStatusField = data.ResultInit.ClientArgs[len(data.ResultInit.ClientArgs)-1].AttributeData
	}
	data.Description = response.Description
	data.nullableProjection.prepareConstructor(data.ResultInit, data.Headers, data.Cookies)
	data.EncodePlan = newResponseEncodePlan(&data)
	return &data
}

func newResponseEncodePlan(response *ResponseData) *ResponseEncodePlan {
	bodyCount := len(response.ServerBody)
	var firstBody *TypeData
	if bodyCount > 0 {
		firstBody = response.ServerBody[0]
	}
	values := make(map[*TypeData]string, bodyCount)
	for _, body := range response.ServerBody {
		if body.Init != nil {
			args := make([]string, len(body.Init.ServerArgs))
			for i, arg := range body.Init.ServerArgs {
				args[i] = arg.Ref
			}
			values[body] = fmt.Sprintf("%s(%s)", body.Init.Name, strings.Join(args, ", "))
			continue
		}
		source := "res"
		if response.SourceVar == "problem" {
			source = "problem"
		} else if response.ViewedResult != nil {
			source += ".Projected"
		}
		if response.ResultAttr != "" {
			if response.nullableProjection != nil {
				source = "responseValue"
			}
			source += "." + response.ResultAttr
		}
		values[body] = source
	}
	fieldSource, guard := response.nullableProjection.encoderFields()
	return &ResponseEncodePlan{
		sourceProjection: response.nullableProjection, bodies: response.ServerBody, FieldSource: fieldSource, MetadataGuard: guard,
		BodyValues:        values,
		BodyCount:         bodyCount,
		HasBody:           bodyCount > 0,
		FirstBody:         firstBody,
		HasMultipleBodies: bodyCount > 1,
		UseViewedBodySwitch: bodyCount > 1 &&
			response.ViewedResult != nil,
		NeedsProblemSource: response.SourceVar == "problem" &&
			(bodyCount > 0 || len(response.Headers) > 0 || len(response.Cookies) > 0),
		NeedsErrorSource: (response.SourceVar != "problem" &&
			(bodyCount > 0 || len(response.Headers) > 0 || len(response.Cookies) > 0 || response.ClientStatusField != nil)) ||
			(response.ErrorHeader != "" && !response.ErrorNameFromStatus),
	}
}

// LiteralCode returns the exact HTTP code for a literal-only emitter. Dynamic
// selector consumers must use Selector and Domain instead.
func (r *ResponseData) LiteralCode() int {
	code, ok := r.Selector.Exact()
	if !ok {
		panic("HTTP literal emission requires an exact status selector")
	}
	return code
}

// LiteralStatus renders an exact HTTP code using its standard library constant
// when available. It rejects dynamic selectors at the literal emission boundary.
func (r *ResponseData) LiteralStatus() string {
	return statusCodeToHTTPConst(r.LiteralCode())
}
