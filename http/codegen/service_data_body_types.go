package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

type requestBodyTypeDetails struct {
	varName              string
	description          string
	definition           string
	validateDefinition   string
	validateReference    string
	flatFormUnionField   string
	flatFormUnionPointer bool
	flatFormUnionTypeKey string
	flatFormUnionRef     string
}

// buildRequestBodyType builds the TypeData for a request body. The data makes
// it possible to generate a function on the client side that creates the body
// from the service method payload.
func (sds *ServicesData) buildRequestBodyType(body, att *expr.AttributeExpr, endpointName string, formEncoded, multipart, svr bool, sd *ServiceData) *TypeData {
	if body.Type == expr.Empty {
		return nil
	}
	var (
		httpctx = httpContext(sd.Scope, true, svr)
		ep      = sd.Service.Method(endpointName)
		pkg     = service.DefaultPackageName(ep.PayloadLoc, sd.Service.PkgName)
		svcctx  = serviceContext(pkg, sd.Service.Scope)
	)
	httpctx.JSONPresence = svr && !formEncoded && !multipart
	ensureTypeLayoutMaps(sd)
	recordAttributeTypeLayouts(sd, body, svr, httpctx.JSONPresence, httpctx.Pointer, httpctx.UseDefault)
	if svr {
		httpctx.JSONPresenceTypes = sd.ServerJSONPresenceTypes
		httpctx.PresencePointerTypes = sd.ServerPresencePointerTypes
		httpctx.PresenceUseDefaultTypes = sd.ServerPresenceUseDefaultTypes
	} else {
		httpctx.JSONPresenceTypes = sd.ClientJSONPresenceTypes
		httpctx.PresencePointerTypes = sd.ClientPresencePointerTypes
		httpctx.PresenceUseDefaultTypes = sd.ClientPresenceUseDefaultTypes
	}
	applyUserTypeLayout(httpctx, sd, body, svr)
	addMarshalTags(body, make(map[string]struct{}))
	details := buildRequestBodyTypeDetails(body, endpointName, formEncoded, svr, sd, httpctx)
	ref := sd.Scope.GoTypeRef(body)
	init := sds.buildRequestBodyInit(body, att, endpointName, pkg, details.validateDefinition, svr, svcctx, httpctx, sd)
	return &TypeData{
		Name:                 body.Type.Name(),
		VarName:              details.varName,
		Description:          details.description,
		Def:                  details.definition,
		Ref:                  ref,
		Init:                 init,
		ValidateDef:          details.validateDefinition,
		ValidateRef:          details.validateReference,
		Example:              body.Example(sds.examplesFor(sd)),
		FlatFormUnionField:   details.flatFormUnionField,
		FlatFormUnionPointer: details.flatFormUnionPointer,
		FlatFormUnionTypeKey: details.flatFormUnionTypeKey,
		FlatFormUnionRef:     details.flatFormUnionRef,
	}
}

func buildRequestBodyTypeDetails(
	body *expr.AttributeExpr,
	endpointName string,
	formEncoded bool,
	svr bool,
	sd *ServiceData,
	httpctx *codegen.AttributeContext,
) requestBodyTypeDetails {
	if codegen.IsExplicitPresenceType(body) {
		ctx := codegen.NewAttributeContext(false, false, !svr, "", sd.Scope)
		return requestBodyTypeDetails{
			varName:           sd.Scope.GoTypeRef(body),
			description:       body.Description,
			validateReference: codegen.ValidationCode(body, nil, ctx, true, expr.IsAlias(body.Type), false, "body"),
		}
	}
	if userType, ok := body.Type.(expr.UserType); ok {
		return buildUserRequestBodyTypeDetails(body, userType, endpointName, formEncoded, svr, sd, httpctx)
	}
	ctx := codegen.NewAttributeContext(!expr.IsPrimitive(body.Type), false, !svr, "", sd.Scope)
	validateReference := codegen.ValidationCode(body, nil, ctx, true, expr.IsAlias(body.Type), false, "body")
	if svr && expr.IsObject(body.Type) {
		body.Validation = nil
	}
	details := requestBodyTypeDetails{
		varName:           sd.Scope.GoTypeRef(body),
		description:       body.Description,
		validateReference: validateReference,
	}
	return details
}

func buildUserRequestBodyTypeDetails(
	body *expr.AttributeExpr,
	userType expr.UserType,
	endpointName string,
	formEncoded bool,
	svr bool,
	sd *ServiceData,
	httpctx *codegen.AttributeContext,
) requestBodyTypeDetails {
	varName := codegen.Goify(userType.Name(), true)
	details := requestBodyTypeDetails{
		varName:     varName,
		description: fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP request body.", varName, sd.Service.Name, endpointName),
	}
	if expr.IsUnion(userType.Attribute().Type) {
		return details
	}
	details.definition = goValueTypeDef(sd.Scope, userType.Attribute(), httpctx.Pointer, httpctx.UseDefault, httpctx.JSONPresence)
	details.flatFormUnionField, details.flatFormUnionPointer, details.flatFormUnionTypeKey, details.flatFormUnionRef =
		flatFormUnionMetadata(userType.Attribute(), formEncoded, sd.Scope)
	if svr || containsUnionType(body.Type) {
		details.validateDefinition = codegen.ValidationCode(body, userType, httpctx, true, expr.IsAlias(body.Type), false, "body")
		if details.validateDefinition != "" {
			details.validateReference = fmt.Sprintf("err = Validate%s(&body)", varName)
		}
	}
	return details
}

func flatFormUnionMetadata(
	attribute *expr.AttributeExpr,
	formEncoded bool,
	scope *codegen.NameScope,
) (fieldName string, pointer bool, typeKey, ref string) {
	if !formEncoded {
		return "", false, "", ""
	}
	object := expr.AsObject(attribute.Type)
	if object == nil || len(*object) != 1 || !expr.IsUnion((*object)[0].Attribute.Type) {
		return "", false, "", ""
	}
	field := (*object)[0]
	return codegen.Goify(field.Name, true),
		!attribute.IsRequired(field.Name),
		expr.AsUnion(field.Attribute.Type).GetTypeKey(),
		scope.GoTypeRef(field.Attribute)
}

// buildResponseBodyType builds the TypeData for a response body. The data
// makes it possible to generate a function that creates the server response
// body from the service method result/projected result or error.
func (sds *ServicesData) buildResponseBodyType(body, att *expr.AttributeExpr, loc *codegen.Location, endpointName string, svr bool, view *string, sd *ServiceData) *TypeData {
	if body.Type == expr.Empty {
		return nil
	}
	httpctx := httpContext(sd.Scope, false, svr)
	httpctx.JSONPresence = !svr
	ensureTypeLayoutMaps(sd)
	if svr {
		httpctx.JSONPresenceTypes = sd.ServerJSONPresenceTypes
		httpctx.PresencePointerTypes = sd.ServerPresencePointerTypes
		httpctx.PresenceUseDefaultTypes = sd.ServerPresenceUseDefaultTypes
	} else {
		httpctx.JSONPresenceTypes = sd.ClientJSONPresenceTypes
		httpctx.PresencePointerTypes = sd.ClientPresencePointerTypes
		httpctx.PresenceUseDefaultTypes = sd.ClientPresenceUseDefaultTypes
	}
	pkg := service.DefaultPackageName(loc, sd.Service.PkgName)
	svcctx := serviceContext(pkg, sd.Service.Scope)
	body, viewName := projectResponseBodyView(body, view, svr, sd)
	if svr {
		recordRootTypeLayout(sd, body, true, false, false, true)
		recordAttributeTypeLayouts(sd, body, true, false, false, true)
	} else {
		recordAttributeTypeLayouts(sd, body, false, true, true, false)
	}
	applyUserTypeLayout(httpctx, sd, body, svr)
	data := initResponseBodyTypeData(body, att, sd)
	addMarshalTags(body, make(map[string]struct{}))

	switch ut := body.Type.(type) {
	case expr.UserType:
		applyUserResponseBodyTypeData(data, body, ut, endpointName, httpctx, sd, svr, view == nil)
	default:
		switch {
		case expr.IsUnion(body.Type):
			applyUnionResponseBodyTypeData(data, body, endpointName, httpctx, sd)
		case !expr.IsPrimitive(body.Type) && data.mustInit:
			applyStructuredResponseBodyTypeData(data, body, endpointName, httpctx, sd, svr)
		default:
			applyPrimitiveResponseBodyTypeData(data, body, sd)
		}
	}
	if svr {
		collectServerResponseBodyTypes(sds, body, data.name, sd)
	}
	init := sds.buildResponseBodyInit(body, att, pkg, endpointName, view, data.mustInit, data.validateDef, svr, svcctx, httpctx, sd)
	return &TypeData{
		Name:        data.name,
		VarName:     data.varName,
		Description: data.desc,
		Def:         data.def,
		Ref:         data.ref,
		ValueRef:    bodyValueRef(sd.Scope, body, data.varName),
		Init:        init,
		ValidateDef: data.validateDef,
		ValidateRef: data.validateRef,
		Example:     body.Example(sds.examplesFor(sd)),
		View:        viewName,
	}
}

func applyUnionResponseBodyTypeData(data *responseBodyTypeData, body *expr.AttributeExpr, endpointName string, httpctx *codegen.AttributeContext, sd *ServiceData) {
	data.varName = sd.Scope.GoTypeName(body)
	data.desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body.", data.varName, sd.Service.Name, endpointName)
	data.validateRef = codegen.ValidationCode(body, nil, httpctx, true, false, false, "body")
}

type responseBodyTypeData struct {
	name        string
	varName     string
	desc        string
	def         string
	ref         string
	validateDef string
	validateRef string
	mustInit    bool
}

func projectResponseBodyView(body *expr.AttributeExpr, view *string, svr bool, sd *ServiceData) (*expr.AttributeExpr, string) {
	if view == nil || *view == "" {
		return body, ""
	}
	viewName := *view
	body = expr.DupAtt(body)
	if rt, ok := body.Type.(*expr.ResultTypeExpr); ok {
		var err error
		rt, err = expr.Project(rt, *view)
		if err != nil {
			panic(codegen.NewError(nil, body, fmt.Errorf("project generated response body view %q: %w", *view, err)))
		}
		body.Type = rt
		body.Validation = rt.Validation
		if svr {
			sd.ServerTypeNames[rt.Name()] = false
		} else {
			sd.ClientTypeNames[rt.Name()] = false
		}
	}
	return body, viewName
}

func initResponseBodyTypeData(body, att *expr.AttributeExpr, sd *ServiceData) *responseBodyTypeData {
	return &responseBodyTypeData{
		name:     body.Type.Name(),
		ref:      bodyTypeRef(sd.Scope, body),
		mustInit: att.Type != expr.Empty && needInit(body),
	}
}

func applyUserResponseBodyTypeData(data *responseBodyTypeData, body *expr.AttributeExpr, ut expr.UserType, endpointName string, httpctx *codegen.AttributeContext, sd *ServiceData, svr, allowValidateDef bool) {
	data.varName = codegen.Goify(ut.Name(), true)
	if !expr.IsUnion(ut.Attribute().Type) {
		data.def = goValueTypeDef(sd.Scope, ut.Attribute(), !svr, svr, !svr)
	}
	data.desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body.", data.varName, sd.Service.Name, endpointName)
	serverRequestValidation := svr && (sd.ServerRequestValidationTypes[ut.ID()] || sd.ServerRequestValidationTypes[ut.Name()])
	if allowValidateDef && (!svr || serverRequestValidation) {
		data.validateDef = codegen.ValidationCode(body, ut, httpctx, true, expr.IsAlias(body.Type), false, "body")
		if data.validateDef == "" && serverRequestValidation {
			data.validateDef = "// no validations"
		}
		if data.validateDef == "" {
			return
		}
		target := "&body"
		if expr.IsArray(ut) || expr.IsNullable(body) {
			target = "body"
		}
		data.validateRef = fmt.Sprintf("err = Validate%s(%s)", data.varName, target)
	}
}

func applyStructuredResponseBodyTypeData(data *responseBodyTypeData, body *expr.AttributeExpr, endpointName string, httpctx *codegen.AttributeContext, sd *ServiceData, svr bool) {
	if svr {
		data.name = codegen.Goify(endpointName, true) + "ResponseBody"
		data.varName = data.name
		data.desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body.", data.varName, sd.Service.Name, endpointName)
		data.def = goTypeDef(sd.Scope, body, !svr, svr, !svr)
	} else {
		data.varName = sd.Scope.GoTypeRef(body)
		data.desc = body.Description
	}
	data.validateRef = codegen.ValidationCode(body, nil, httpctx, true, expr.IsAlias(body.Type), false, "body")
}

func applyPrimitiveResponseBodyTypeData(data *responseBodyTypeData, body *expr.AttributeExpr, sd *ServiceData) {
	httpctx := httpContext(sd.Scope, false, true)
	data.validateRef = codegen.ValidationCode(body, nil, httpctx, true, expr.IsAlias(body.Type), false, "body")
	data.varName = sd.Scope.GoTypeRef(body)
	data.desc = body.Description
}

func collectServerResponseBodyTypes(sds *ServicesData, body *expr.AttributeExpr, name string, sd *ServiceData) {
	sd.ServerTypeNames[name] = false
	collectUserTypes(body.Type, func(ut expr.UserType) {
		if d := sds.attributeTypeData(ut, false, false, true, false, sd); d != nil {
			sd.ServerBodyAttributeTypes = append(sd.ServerBodyAttributeTypes, d)
		}
	})
}

func (sds *ServicesData) buildRequestBodyInit(
	body, att *expr.AttributeExpr,
	endpointName string,
	pkg, validateDef string,
	svr bool,
	svcctx, httpctx *codegen.AttributeContext,
	sd *ServiceData,
) *InitData {
	if svr || att.Type == expr.Empty || !needInit(body) {
		return nil
	}

	const sourceVar = "p"

	initName := fmt.Sprintf("New%s", codegen.Goify(sd.Scope.GoTypeName(body), true))
	initDesc := fmt.Sprintf("%s builds the HTTP request body from the payload of the %q endpoint of the %q service.",
		initName, endpointName, sd.Service.Name)
	srcAtt, src, origin := serviceBodyTransformSource(att, body, sourceVar)
	code, helpers, err := marshal(srcAtt, body, src, "body", svcctx, httpctx)
	if err != nil {
		panic(codegen.NewError(nil, body, fmt.Errorf("build HTTP request body transform: %w", err)))
	}
	sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)

	arg := &InitArgData{
		Ref: sourceVar,
		AttributeData: &AttributeData{
			Name:     "payload",
			VarName:  sourceVar,
			TypeRef:  sd.Service.Scope.GoFullTypeRef(att, pkg),
			Type:     att.Type,
			Validate: validateDef,
			Example:  att.Example(sds.examplesFor(sd)),
		},
	}
	return &InitData{
		Name:                initName,
		Description:         initDesc,
		ReturnTypeRef:       sd.Scope.GoTypeRef(body),
		ReturnTypeAttribute: codegen.Goify(origin, true),
		ClientCode:          code,
		ClientArgs:          []*InitArgData{arg},
	}
}

func (sds *ServicesData) buildResponseBodyInit(
	body, att *expr.AttributeExpr,
	pkg string,
	endpointName string,
	view *string,
	mustInit bool,
	validateDef string,
	svr bool,
	svcctx, httpctx *codegen.AttributeContext,
	sd *ServiceData,
) *InitData {
	if !svr || !mustInit {
		return nil
	}

	const sourceVar = "res"

	rtname := codegen.Goify(sd.Scope.GoValueTypeName(body), true)
	rtref := bodyTypeRef(sd.Scope, body)
	if _, ok := body.Type.(expr.UserType); !ok && !expr.IsPrimitive(body.Type) && !expr.IsUnion(body.Type) {
		rtname = codegen.Goify(endpointName, true) + "ResponseBody"
		rtref = rtname
	}
	initName := fmt.Sprintf("New%s", rtname)
	initDesc := fmt.Sprintf("%s builds the HTTP response body from the result of the %q endpoint of the %q service.",
		initName, endpointName, sd.Service.Name)
	if view != nil {
		svcctx = viewContext(sd.Service.ViewsPkg, sd.Service.ViewScope)
	}

	srcAtt, src, origin := serviceBodyTransformSource(att, body, sourceVar)
	code, helpers, err := marshal(srcAtt, body, src, "body", svcctx, httpctx)
	if err != nil {
		panic(codegen.NewError(nil, body, fmt.Errorf("build HTTP response body transform: %w", err)))
	}
	sd.ServerTransformHelpers = codegen.AppendHelpers(sd.ServerTransformHelpers, helpers)

	argRef := sourceVar
	argTypeRef := sd.Service.Scope.GoFullTypeRef(att, pkg)
	if view != nil {
		argRef += ".Projected"
		argTypeRef = sd.Service.ViewScope.GoFullTypeRef(att, sd.Service.ViewsPkg)
	}
	arg := &InitArgData{
		Ref: argRef,
		AttributeData: &AttributeData{
			Name:     "result",
			VarName:  sourceVar,
			TypeRef:  argTypeRef,
			Type:     att.Type,
			Validate: validateDef,
			Example:  att.Example(sds.examplesFor(sd)),
		},
	}
	return &InitData{
		Name:                initName,
		Description:         initDesc,
		ReturnTypeRef:       rtref,
		ReturnTypeAttribute: codegen.Goify(origin, true),
		ServerCode:          code,
		ServerArgs:          []*InitArgData{arg},
	}
}

func bodyTypeRef(scope *codegen.NameScope, body *expr.AttributeExpr) string {
	if expr.IsNullable(body) {
		return scope.GoTypeName(body)
	}
	return scope.GoTypeRef(body)
}

func bodyValueRef(scope *codegen.NameScope, body *expr.AttributeExpr, varName string) string {
	if expr.IsNullable(body) {
		return scope.GoTypeName(body)
	}
	return varName
}

func serviceBodyTransformSource(att, body *expr.AttributeExpr, sourceVar string) (*expr.AttributeExpr, string, string) {
	origin, ok := body.Meta["origin:attribute"]
	if !ok || len(origin) == 0 {
		return att, sourceVar, ""
	}
	name := origin[0]
	attribute := expr.AsObject(att.Type).Attribute(name)
	attribute = serviceFieldTransformAttribute(att, name, attribute)
	return attribute, sourceVar + "." + codegen.Goify(name, true), name
}
