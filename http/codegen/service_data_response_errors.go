package codegen

import (
	"fmt"
	"sort"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type errorBuilder struct {
	sds        *ServicesData
	endpoint   *transportir.Endpoint
	sd         *ServiceData
	svc        *service.Data
	method     *service.MethodData
	httpclictx *codegen.AttributeContext
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
		httpclictx: responseHTTPContext(sd.Scope),
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
	errorDesc := service.BuildErrorDescriptor(b.svc, httpError.Name, httpError.Attribute)
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
	if !needInit(httpError.Attribute) && !expr.ContainsNonNullableCollectionElement(responseStatusBody(errorResponse)) {
		return nil
	}
	vars := newResponseVarScope(b.sd)
	headers := b.sds.extractHeaders(errorResponse.Headers, httpError.Attribute, errctx, b.sd.Scope, vars, b.sds.examplesFor(b.sd))
	cookies := b.sds.extractResponseCookies(errorResponse.Cookies, httpError.Attribute, errctx, b.sd.Scope, vars, b.sds.examplesFor(b.sd))
	args := make([]*InitArgData, 0, len(headers)+len(cookies)+1)
	if body != expr.Empty {
		args = append(args, buildBodyInitArg(b.sd.Scope, responseStatusBody(errorResponse), true, b.httpclictx))
	}
	args = append(args, buildHeaderInitArgs(headers)...)
	args = append(args, buildCookieInitArgs(cookies)...)
	code, origin, skipFieldInit := b.buildResultInitCode(errorResponse, errctx, args)
	fieldName, _, fieldPointer := transportFieldBinding(origin, httpError.Attribute, httpError.Attribute, errctx)
	name := fmt.Sprintf("New%s%s", codegen.Goify(b.method.Name, true), codegen.Goify(httpError.Name, true))
	return &InitData{
		Name:                     name,
		Description:              fmt.Sprintf("%s builds a %s service %s endpoint %s error.", name, b.svc.Name, b.endpoint.Name, httpError.Name),
		ClientArgs:               args,
		ReturnTypeName:           b.svc.Scope.GoFullTypeName(httpError.Attribute, pkg),
		ReturnTypeRef:            b.svc.Scope.GoFullTypeRef(httpError.Attribute, pkg),
		ReturnIsStruct:           expr.IsObject(httpError.Type),
		ReturnIsUnionValue:       bodyFieldIsUnionValue(httpError.Attribute, origin),
		ReturnTypeAttribute:      fieldName,
		ReturnIsPrimitivePointer: fieldPointer,
		ReturnTypePkg:            pkg,
		ClientCode:               code,
		SkipFieldInit:            skipFieldInit,
	}
}

func (b *errorBuilder) buildResultInitCode(errorResponse *transportir.ResponseStatus, errctx *codegen.AttributeContext, args []*InitArgData) (string, string, bool) {
	origin := ""
	httpError := errorResponse.Error
	body := responseStatusBody(errorResponse)
	if expr.IsDefaultErrorResult(httpError.Attribute.Type) {
		return b.buildProblemClientResult(errorResponse, args), "", true
	}
	errAtt := httpError.Attribute
	if o, ok := body.Meta["origin:attribute"]; ok {
		origin = o[0]
		_, errAtt = httpError.Attribute.FindAttribute(origin)
		errAtt = serviceFieldTransformAttribute(httpError.Attribute, origin, errAtt)
	}
	code, err := b.sds.buildClientResultTransformCode(body, errAtt, httpError.Attribute, b.endpoint.Request, b.httpclictx, errctx, b.sd)
	if err != nil {
		panic(codegen.NewError(b.sds.Ctx, body, fmt.Errorf("build HTTP error response transform for %s: %w", httpError.Name, err)))
	}
	return code, origin, false
}

func (b *errorBuilder) buildResponseData(errorResponse *transportir.ResponseStatus, errctx *codegen.AttributeContext, init *InitData) *ResponseData {
	httpError := errorResponse.Error
	serverBodyData, clientBodyData := b.buildResponseBodyData(errorResponse)
	vars := newResponseVarScope(b.sd)
	headers := b.sds.extractHeaders(errorResponse.Headers, httpError.Attribute, errctx, b.sd.Scope, vars, b.sds.examplesFor(b.sd))
	cookies := b.sds.extractResponseCookies(errorResponse.Cookies, httpError.Attribute, errctx, b.sd.Scope, vars, b.sds.examplesFor(b.sd))
	contentType := errorResponse.ContentType
	origin, _ := responseOriginAttribute(errorResponse, httpError.Attribute)
	fieldName, _, _ := transportFieldBinding(origin, httpError.Attribute, httpError.Attribute, errctx)
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
			ResultAttr:           fieldName,
			MustValidate:         responseFieldsNeedValidation(headers, cookies),
			SourceVar:            headerSourceVar,
			ProblemTypeOverride:  problemTypeOverride,
			ProblemTitleOverride: problemTitleOverride,
		},
	)
}

func (b *errorBuilder) buildResponseBodyData(errorResponse *transportir.ResponseStatus) ([]*TypeData, *TypeData) {
	httpError := errorResponse.Error
	errorLoc := codegen.UserTypeLocation(httpError.Attribute.Type)
	serverBodyData, clientBodyData := b.sds.buildResponseBodyPair(responseStatusBody(errorResponse), httpError.Attribute, errorLoc, b.endpoint.Name, b.sd, errorResponse.BodyValue)

	if clientBodyData != nil {
		clientBodyData.Description = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body for the %q error.", clientBodyData.VarName, b.svc.Name, b.endpoint.Name, httpError.Name)
		serverBodyData[0].Description = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body for the %q error.", serverBodyData[0].VarName, b.svc.Name, b.endpoint.Name, httpError.Name)
	}
	return serverBodyData, clientBodyData
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
