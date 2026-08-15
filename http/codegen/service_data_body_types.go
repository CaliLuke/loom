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
func (sds *ServicesData) buildRequestBodyType(body, att *expr.AttributeExpr, endpointName string, formEncoded bool, svr bool, sd *ServiceData) *TypeData {
	if body.Type == expr.Empty {
		return nil
	}
	var (
		httpctx = httpContext(sd.Scope, true, svr)
		ep      = sd.Service.Method(endpointName)
		pkg     = service.DefaultPackageName(ep.PayloadLoc, sd.Service.PkgName)
		svcctx  = serviceContext(pkg, sd.Service.Scope)
	)
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
		Example:              body.Example(sds.Root.API.ExampleGenerator),
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
		definition:  goTypeDef(sd.Scope, userType.Attribute(), svr, !svr),
	}
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
	pkg := service.DefaultPackageName(loc, sd.Service.PkgName)
	svcctx := serviceContext(pkg, sd.Service.Scope)
	body, viewName := projectResponseBodyView(body, view, svr, sd)
	data := initResponseBodyTypeData(body, att, sd)
	addMarshalTags(body, make(map[string]struct{}))

	if ut, ok := body.Type.(expr.UserType); ok {
		applyUserResponseBodyTypeData(data, body, ut, endpointName, httpctx, sd, svr, view == nil)
	} else if !expr.IsPrimitive(body.Type) && data.mustInit {
		applyStructuredResponseBodyTypeData(data, body, endpointName, httpctx, sd, svr)
	} else {
		applyPrimitiveResponseBodyTypeData(data, body, sd)
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
		Init:        init,
		ValidateDef: data.validateDef,
		ValidateRef: data.validateRef,
		Example:     body.Example(sds.Root.API.ExampleGenerator),
		View:        viewName,
	}
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
		ref:      sd.Scope.GoTypeRef(body),
		mustInit: att.Type != expr.Empty && needInit(body),
	}
}

func applyUserResponseBodyTypeData(data *responseBodyTypeData, body *expr.AttributeExpr, ut expr.UserType, endpointName string, httpctx *codegen.AttributeContext, sd *ServiceData, svr, allowValidateDef bool) {
	data.varName = codegen.Goify(ut.Name(), true)
	data.def = goTypeDef(sd.Scope, ut.Attribute(), !svr, svr)
	data.desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body.", data.varName, sd.Service.Name, endpointName)
	if !svr && allowValidateDef {
		data.validateDef = codegen.ValidationCode(body, ut, httpctx, true, expr.IsAlias(body.Type), false, "body")
		if data.validateDef == "" {
			return
		}
		target := "&body"
		if expr.IsArray(ut) {
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
		data.def = goTypeDef(sd.Scope, body, !svr, svr)
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
		if d := sds.attributeTypeData(ut, false, false, true, sd); d != nil {
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
	src := sourceVar
	srcAtt := att
	origin := ""
	if o, ok := body.Meta["origin:attribute"]; ok {
		srcObj := expr.AsObject(att.Type)
		origin = o[0]
		srcAtt = srcObj.Attribute(origin)
		src += "." + codegen.Goify(origin, true)
	}
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
			Example:  att.Example(sds.Root.API.ExampleGenerator),
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

	rtname := codegen.Goify(sd.Scope.GoTypeName(body), true)
	rtref := sd.Scope.GoTypeRef(body)
	if _, ok := body.Type.(expr.UserType); !ok && !expr.IsPrimitive(body.Type) {
		rtname = codegen.Goify(endpointName, true) + "ResponseBody"
		rtref = rtname
	}
	initName := fmt.Sprintf("New%s", rtname)
	initDesc := fmt.Sprintf("%s builds the HTTP response body from the result of the %q endpoint of the %q service.",
		initName, endpointName, sd.Service.Name)
	if view != nil {
		svcctx = viewContext(sd.Service.ViewsPkg, sd.Service.ViewScope)
	}

	src := sourceVar
	srcAtt := att
	origin := ""
	if o, ok := body.Meta["origin:attribute"]; ok {
		srcObj := expr.AsObject(att.Type)
		origin = o[0]
		srcAtt = srcObj.Attribute(origin)
		src += "." + codegen.Goify(origin, true)
	}
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
			Example:  att.Example(sds.Root.API.ExampleGenerator),
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
