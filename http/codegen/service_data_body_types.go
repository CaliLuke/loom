package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/v3/codegen"
	"github.com/CaliLuke/loom/v3/expr"
)

// buildRequestBodyType builds the TypeData for a request body. The data makes
// it possible to generate a function on the client side that creates the body
// from the service method payload.
func (sds *ServicesData) buildRequestBodyType(body, att *expr.AttributeExpr, e *expr.HTTPEndpointExpr, svr bool, sd *ServiceData) *TypeData {
	if body.Type == expr.Empty {
		return nil
	}
	var (
		name               string
		varname            string
		desc               string
		def                string
		ref                string
		validateDef        string
		validateRef        string
		flatFormUnionField string

		svc     = sd.Service
		httpctx = httpContext(sd.Scope, true, svr)
		ep      = sd.Service.Method(e.Name())
		pkg     = pkgWithDefault(ep.PayloadLoc, sd.Service.PkgName)
		svcctx  = serviceContext(pkg, sd.Service.Scope)
	)
	name = body.Type.Name()
	ref = sd.Scope.GoTypeRef(body)

	addMarshalTags(body, make(map[string]struct{}))

	if ut, ok := body.Type.(expr.UserType); ok {
		varname = codegen.Goify(ut.Name(), true)
		def = goTypeDef(sd.Scope, ut.Attribute(), svr, !svr)
		desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP request body.",
			varname, svc.Name, e.Name())
		if e.FormRequest {
			if obj := expr.AsObject(ut.Attribute().Type); obj != nil && len(*obj) == 1 && expr.IsUnion((*obj)[0].Attribute.Type) {
				flatFormUnionField = codegen.Goify((*obj)[0].Name, true)
			}
		}
		if svr || containsUnionType(body.Type) {
			validateDef = codegen.ValidationCode(body, ut, httpctx, true, expr.IsAlias(body.Type), false, "body")
			if validateDef != "" {
				validateRef = fmt.Sprintf("err = Validate%s(&body)", varname)
			}
		}
	} else {
		ctx := codegen.NewAttributeContext(!expr.IsPrimitive(body.Type), false, !svr, "", sd.Scope)
		validateRef = codegen.ValidationCode(body, nil, ctx, true, expr.IsAlias(body.Type), false, "body")
		if svr && expr.IsObject(body.Type) {
			body.Validation = nil
		}
		varname = sd.Scope.GoTypeRef(body)
		desc = body.Description
	}
	init := sds.buildRequestBodyInit(body, att, e, pkg, validateDef, svr, svcctx, httpctx, sd)
	return &TypeData{
		Name:               name,
		VarName:            varname,
		Description:        desc,
		Def:                def,
		Ref:                ref,
		Init:               init,
		ValidateDef:        validateDef,
		ValidateRef:        validateRef,
		Example:            body.Example(sds.Root.API.ExampleGenerator),
		FlatFormUnionField: flatFormUnionField,
	}
}

// buildResponseBodyType builds the TypeData for a response body. The data
// makes it possible to generate a function that creates the server response
// body from the service method result/projected result or error.
func (sds *ServicesData) buildResponseBodyType(body, att *expr.AttributeExpr, loc *codegen.Location, e *expr.HTTPEndpointExpr, svr bool, view *string, sd *ServiceData) *TypeData {
	if body.Type == expr.Empty {
		return nil
	}
	var (
		name        string
		varname     string
		desc        string
		def         string
		ref         string
		validateDef string
		validateRef string
		viewName    string
		mustInit    bool

		svc     = sd.Service
		httpctx = httpContext(sd.Scope, false, svr)
		pkg     = pkgWithDefault(loc, sd.Service.PkgName)
		svcctx  = serviceContext(pkg, sd.Service.Scope)
	)
	if svr && view != nil && *view != "" {
		viewName = *view
		body = expr.DupAtt(body)
		if rt, ok := body.Type.(*expr.ResultTypeExpr); ok {
			var err error
			rt, err = expr.Project(rt, *view)
			if err != nil {
				panic(err)
			}
			body.Type = rt
			sd.ServerTypeNames[rt.Name()] = false
		}
	}

	name = body.Type.Name()
	ref = sd.Scope.GoTypeRef(body)
	mustInit = att.Type != expr.Empty && needInit(body.Type)

	addMarshalTags(body, make(map[string]struct{}))

	if ut, ok := body.Type.(expr.UserType); ok {
		varname = codegen.Goify(ut.Name(), true)
		def = goTypeDef(sd.Scope, ut.Attribute(), !svr, svr)
		desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body.",
			varname, svc.Name, e.Name())
		if !svr && view == nil {
			validateDef = codegen.ValidationCode(body, ut, httpctx, true, expr.IsAlias(body.Type), false, "body")
			if validateDef != "" {
				target := "&body"
				if expr.IsArray(ut) {
					target = "body"
				}
				validateRef = fmt.Sprintf("err = Validate%s(%s)", varname, target)
			}
		}
	} else if !expr.IsPrimitive(body.Type) && mustInit {
		if svr {
			name = codegen.Goify(e.Name(), true) + "ResponseBody"
			varname = name
			desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body.",
				varname, svc.Name, e.Name())
			def = goTypeDef(sd.Scope, body, !svr, svr)
		} else {
			varname = sd.Scope.GoTypeRef(body)
			desc = body.Description
			def = ""
		}
		validateRef = codegen.ValidationCode(body, nil, httpctx, true, expr.IsAlias(body.Type), false, "body")
	} else {
		httpctx = httpContext(sd.Scope, false, true)
		validateRef = codegen.ValidationCode(body, nil, httpctx, true, expr.IsAlias(body.Type), false, "body")
		varname = sd.Scope.GoTypeRef(body)
		desc = body.Description
	}
	if svr {
		sd.ServerTypeNames[name] = false
		collectUserTypes(body.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, false, false, true, sd); d != nil {
				sd.ServerBodyAttributeTypes = append(sd.ServerBodyAttributeTypes, d)
			}
		})
	}
	init := sds.buildResponseBodyInit(body, att, pkg, e, view, mustInit, validateDef, svr, svcctx, httpctx, sd)
	return &TypeData{
		Name:        name,
		VarName:     varname,
		Description: desc,
		Def:         def,
		Ref:         ref,
		Init:        init,
		ValidateDef: validateDef,
		ValidateRef: validateRef,
		Example:     body.Example(sds.Root.API.ExampleGenerator),
		View:        viewName,
	}
}

func (sds *ServicesData) buildRequestBodyInit(
	body, att *expr.AttributeExpr,
	e *expr.HTTPEndpointExpr,
	pkg, validateDef string,
	svr bool,
	svcctx, httpctx *codegen.AttributeContext,
	sd *ServiceData,
) *InitData {
	if svr || att.Type == expr.Empty || !needInit(body.Type) {
		return nil
	}

	const sourceVar = "p"

	initName := fmt.Sprintf("New%s", codegen.Goify(sd.Scope.GoTypeName(body), true))
	initDesc := fmt.Sprintf("%s builds the HTTP request body from the payload of the %q endpoint of the %q service.",
		initName, e.Name(), sd.Service.Name)
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
		fmt.Println(err.Error())
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
	e *expr.HTTPEndpointExpr,
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
		rtname = codegen.Goify(e.Name(), true) + "ResponseBody"
		rtref = rtname
	}
	initName := fmt.Sprintf("New%s", rtname)
	initDesc := fmt.Sprintf("%s builds the HTTP response body from the result of the %q endpoint of the %q service.",
		initName, e.Name(), sd.Service.Name)
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
		panic(err)
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
