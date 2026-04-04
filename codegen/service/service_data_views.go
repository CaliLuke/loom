package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// typeContext returns a contextual attribute for service types. Service types
// are Go types and uses non-pointers to hold attributes having default values.
func typeContext(scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(false, false, true, "", scope)
}

// projectedTypeContext returns a contextual attribute for a projected type.
// Projected types are Go types that uses pointers for all attributes (even the
// required ones).
func projectedTypeContext(pkg string, ptr bool, scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(ptr, false, true, pkg, scope)
}

// collectViewUnionTypes traverses the attribute to gather all union sum-type
// definitions referenced by view-projected types. It always uses the provided
// location for all nested user types so that unions are generated in the views
// package and refer to view-local types (preventing import cycles).
func collectViewUnionTypes(att *expr.AttributeExpr, scope *codegen.NameScope, loc *codegen.Location, unions map[string]*UnionTypeData, seen map[string]struct{}) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[dt.ID()]; ok {
			return
		}
		seen[dt.ID()] = struct{}{}
		collectViewUnionTypes(dt.Attribute(), scope, loc, unions, seen)
	case *expr.Object:
		for _, nat := range sortedNamedAttributes(*dt) {
			collectViewUnionTypes(nat.Attribute, scope, loc, unions, seen)
		}
	case *expr.Array:
		collectViewUnionTypes(dt.ElemType, scope, loc, unions, seen)
	case *expr.Map:
		collectViewUnionTypes(dt.KeyType, scope, loc, unions, seen)
		collectViewUnionTypes(dt.ElemType, scope, loc, unions, seen)
	case *expr.Union:
		hash := dt.Hash()
		if _, ok := unions[hash]; !ok {
			unions[hash] = buildViewUnionTypeData(dt, scope, loc)
		}
		for _, nat := range dt.Values {
			collectViewUnionTypes(nat.Attribute, scope, loc, unions, seen)
		}
	}
}

// collectProjectedTypes builds a projected type for every user type found when
// recursing through the attributes. The projected types live in the views
// package and support the marshaling and unmarshalling of result types that
// make use of views. We need to build projected types for all user types - not
// just result types - because user types may contain result types and thus may
// need to be marshalled in different ways depending on the view being used.
func collectProjectedTypes(projected, att *expr.AttributeExpr, viewspkg string, scope, viewScope *codegen.NameScope, seen map[string]*ProjectedTypeData) []*ProjectedTypeData {
	collect := func(projected, att *expr.AttributeExpr) []*ProjectedTypeData {
		return collectProjectedTypes(projected, att, viewspkg, scope, viewScope, seen)
	}
	var data []*ProjectedTypeData
	switch pt := projected.Type.(type) {
	case expr.UserType:
		dt := att.Type.(expr.UserType)
		if pd, ok := seen[dt.ID()]; ok {
			if pd != nil {
				projected.Type = pd.Type
			}
			return data
		}
		seen[dt.ID()] = nil
		pt.Rename(pt.Name() + "View")
		types := collect(pt.Attribute(), dt.Attribute())
		pd := buildProjectedType(projected, att, viewspkg, scope, viewScope)
		seen[dt.ID()] = pd
		data = append(data, pd)
		data = append(data, types...)
	case *expr.Array:
		dt := att.Type.(*expr.Array)
		types := collect(pt.ElemType, dt.ElemType)
		data = append(data, types...)
	case *expr.Map:
		dt := att.Type.(*expr.Map)
		types := collect(pt.KeyType, dt.KeyType)
		data = append(data, types...)
		types = collect(pt.ElemType, dt.ElemType)
		data = append(data, types...)
	case *expr.Object:
		dt := att.Type.(*expr.Object)
		for _, n := range *pt {
			types := collect(n.Attribute, dt.Attribute(n.Name))
			data = append(data, types...)
		}
	case *expr.Union:
		dt := att.Type.(*expr.Union)
		for i, n := range pt.Values {
			types := collect(n.Attribute, dt.Values[i].Attribute)
			data = append(data, types...)
		}
	}
	return data
}

// buildProjectedType builds projected type for the given user type.
//
// viewspkg is the name of the views package
func buildProjectedType(projected, att *expr.AttributeExpr, viewspkg string, scope, viewScope *codegen.NameScope) *ProjectedTypeData {
	var (
		projections []*InitData
		typeInits   []*InitData
		views       []*ViewData

		varname = viewScope.GoTypeName(projected)
		pt      = projected.Type.(expr.UserType)
	)
	if _, isrt := pt.(*expr.ResultTypeExpr); isrt {
		typeInits = buildTypeInits(projected, att, viewspkg, scope, viewScope)
		projections = buildProjections(projected, att, viewspkg, scope, viewScope)
		views = buildViews(att.Type.(*expr.ResultTypeExpr), viewScope)
	}
	validations := buildValidations(projected, viewScope)
	removeMeta(projected)
	return &ProjectedTypeData{
		UserTypeData: &UserTypeData{
			Name:        varname,
			Description: fmt.Sprintf("%s is a type that runs validations on a projected type.", varname),
			VarName:     varname,
			Def:         viewScope.GoTypeDef(pt.Attribute(), true, true),
			Ref:         viewScope.GoTypeRef(projected),
			Type:        pt,
		},
		Projections: projections,
		TypeInits:   typeInits,
		Validations: validations,
		ViewsPkg:    viewspkg,
		Views:       views,
	}
}

// buildViews builds the view data for all the views in the given result type.
func buildViews(rt *expr.ResultTypeExpr, viewScope *codegen.NameScope) []*ViewData {
	views := make([]*ViewData, len(rt.Views))
	for i, view := range rt.Views {
		vatt := expr.AsObject(view.Type)
		attrs := make([]string, len(*vatt))
		for j, nat := range *vatt {
			attrs[j] = nat.Name
		}
		views[i] = &ViewData{
			Name:        view.Name,
			Description: view.Description,
			Attributes:  attrs,
			TypeVarName: viewScope.GoTypeName(&expr.AttributeExpr{Type: rt}),
		}
	}
	return views
}

func viewedResultDefaultViewName(att *expr.AttributeExpr, rt *expr.ResultTypeExpr) string {
	var viewName string
	if !rt.HasMultipleViews() {
		viewName = expr.DefaultView
	}
	if v, ok := att.Meta.Last(expr.ViewMetaKey); ok {
		viewName = v
	}
	return viewName
}

// buildViewedResultType builds a viewed result type from the given result type
// and projected type.
func buildViewedResultType(att, projected *expr.AttributeExpr, viewspkg string, scope, viewScope *codegen.NameScope) *ViewedResultTypeData {
	rt := att.Type.(*expr.ResultTypeExpr)
	isarr := expr.IsArray(att.Type)
	viewName := viewedResultDefaultViewName(att, rt)
	views := buildViews(rt, viewScope)
	resvar := scope.GoTypeName(att)
	vresref := viewScope.GoFullTypeRef(att, viewspkg)
	validate := buildViewedResultValidation(projected, views, scope, att, resvar)
	init := buildViewedResultInit(att, views, viewspkg, scope, resvar, vresref, isarr)
	resinit, resref := buildViewedResultResultInit(att, projected, views, viewspkg, scope, viewScope, resvar)
	projT := wrapProjected(projected.Type.(expr.UserType))
	return &ViewedResultTypeData{
		UserTypeData: &UserTypeData{
			Name:        resvar,
			Description: fmt.Sprintf("%s is the viewed result type that is projected based on a view.", resvar),
			VarName:     resvar,
			Def:         viewScope.GoTypeDef(projT.Attribute(), false, true),
			Ref:         resref,
			Type:        projT,
		},
		FullName:     scope.GoFullTypeName(att, viewspkg),
		FullRef:      vresref,
		ResultInit:   resinit,
		Init:         init,
		Views:        views,
		Validate:     validate,
		IsCollection: isarr,
		ViewName:     viewName,
		ViewsPkg:     viewspkg,
	}
}

func buildViewedResultValidation(projected *expr.AttributeExpr, views []*ViewData, scope *codegen.NameScope, att *expr.AttributeExpr, resvar string) *ValidateData {
	validateTData := viewedResultValidateTemplateData{
		Projected: scope.GoTypeName(projected),
		ArgVar:    "result",
		Source:    "result",
		Views:     views,
		IsViewed:  true,
	}
	validate := executeValidateTypeTemplate(validateTData)
	name := "Validate" + resvar
	return &ValidateData{
		Name:        name,
		Description: fmt.Sprintf("%s runs the validations defined on the viewed result type %s.", name, resvar),
		Ref:         scope.GoTypeRef(att),
		Validate:    validate,
	}
}

func buildViewedResultInit(att *expr.AttributeExpr, views []*ViewData, viewspkg string, scope *codegen.NameScope, resvar, vresref string, isarr bool) *InitData {
	initTData := viewedResultInitTemplateData{
		ToViewed:      true,
		ArgVar:        "res",
		ReturnVar:     "vres",
		Views:         views,
		ReturnTypeRef: vresref,
		IsCollection:  isarr,
		TargetType:    scope.GoFullTypeName(att, viewspkg),
		InitName:      projectionHelperBaseName(scope, att),
		ViewExpr:      "view",
	}
	name := "NewViewed" + resvar
	return &InitData{
		Name:        name,
		Description: fmt.Sprintf("%s initializes viewed result type %s from result type %s using the given view.", name, resvar, resvar),
		Args: []*InitArgData{
			{Name: "res", Ref: fullTypeRefForAttribute(scope, att, "")},
			{Name: "view", Ref: "string"},
		},
		ReturnTypeRef: vresref,
		Code:          executeInitTypeTemplate(initTData),
	}
}

func buildViewedResultResultInit(att, projected *expr.AttributeExpr, views []*ViewData, viewspkg string, scope, viewScope *codegen.NameScope, resvar string) (*InitData, string) {
	resref := fullTypeRefForAttribute(scope, att, "")
	resultInitTData := viewedResultInitTemplateData{
		ToResult:      true,
		ArgVar:        "vres",
		ReturnVar:     "res",
		Views:         views,
		ReturnTypeRef: resref,
		InitName:      projectedResultInitHelperBaseName(scope, viewScope, att, projected),
		ViewExpr:      "vres.View",
	}
	name := "New" + resvar
	return &InitData{
		Name:          name,
		Description:   fmt.Sprintf("%s initializes result type %s from viewed result type %s.", name, resvar, resvar),
		Args:          []*InitArgData{{Name: "vres", Ref: scope.GoFullTypeRef(att, viewspkg)}},
		ReturnTypeRef: resref,
		Code:          executeInitTypeTemplate(resultInitTData),
	}, resref
}

func executeValidateTypeTemplate(data viewedResultValidateTemplateData) string {
	return renderValidateTypeCode(data)
}

func executeInitTypeTemplate(data viewedResultInitTemplateData) string {
	return renderInitTypeCode(data)
}

func renderValidateTypeCode(data viewedResultValidateTemplateData) string {
	var lines []string
	if data.IsViewed {
		lines = append(lines, "switch "+data.ArgVar+".View {")
		for _, view := range data.Views {
			caseLine := "case " + quotedViewCase(view.Name) + ":"
			lines = append(lines, "\t"+caseLine)
			validateName := "Validate" + data.Projected
			if view.Name != expr.DefaultView {
				validateName += codegen.Goify(view.Name, true)
			}
			lines = append(lines, "\t\terr = "+validateName+"("+data.ArgVar+".Projected)")
		}
		lines = append(lines, "\tdefault:")
		lines = append(lines, "\t\terr = loom.InvalidEnumValueError(\"view\", "+data.Source+".View, []any{ "+strings.Join(quotedViews(data.Views), ", ")+" })")
		lines = append(lines, "}")
		return strings.Join(lines, "\n")
	}

	if data.IsCollection {
		lines = append(lines, "for _, "+data.Source+" := range "+data.ArgVar+" {")
		lines = append(lines, "\tif err2 := "+data.ValidateVar+"("+data.Source+"); err2 != nil {")
		lines = append(lines, "\t\terr = loom.MergeErrors(err, err2)")
		lines = append(lines, "\t}")
		lines = append(lines, "}")
		return strings.Join(lines, "\n")
	}

	if data.Validate != "" {
		lines = append(lines, data.Validate)
	} else if needsValidationFieldSpacer(data.Fields) {
		lines = append(lines, "")
	}
	for _, field := range data.Fields {
		fieldName := codegen.Goify(field.Name, true)
		if field.IsRequired {
			lines = append(lines, "if "+data.Source+"."+fieldName+" == nil {")
			lines = append(lines, "\terr = loom.MergeErrors(err, loom.MissingFieldError("+fmt.Sprintf("%q", field.Name)+", "+fmt.Sprintf("%q", data.Source)+"))")
			lines = append(lines, "}")
		}
		lines = append(lines, "if "+data.Source+"."+fieldName+" != nil {")
		lines = append(lines, "\tif err2 := "+field.ValidateVar+"("+data.Source+"."+fieldName+"); err2 != nil {")
		lines = append(lines, "\t\terr = loom.MergeErrors(err, err2)")
		lines = append(lines, "\t}")
		lines = append(lines, "}")
	}
	return strings.Join(lines, "\n")
}

func renderInitTypeCode(data viewedResultInitTemplateData) string {
	var lines []string
	switch {
	case data.ToResult || data.ToViewed:
		lines = append(lines, "")
		lines = append(lines, "var "+data.ReturnVar+" "+data.ReturnTypeRef)
		lines = append(lines, "switch "+data.ViewExpr+" {")
		for _, view := range data.Views {
			lines = append(lines, "\tcase "+quotedViewCase(view.Name)+":")
			initName := data.InitName
			if view.Name != expr.DefaultView {
				initName += codegen.Goify(view.Name, true)
			}
			if data.ToViewed {
				lines = append(lines, "\t\tp := "+initName+"("+data.ArgVar+")")
				prefix := ""
				if !data.IsCollection {
					prefix = "&"
				}
				lines = append(lines, "\t\t"+data.ReturnVar+" = "+prefix+data.TargetType+"{Projected: p, View: "+fmt.Sprintf("%q", view.Name)+" }")
			} else {
				lines = append(lines, "\t\t"+data.ReturnVar+" = "+initName+"("+data.ArgVar+".Projected)")
			}
		}
		lines = append(lines, "\tdefault:")
		lines = append(lines, "\t\tpanic(loom.InvalidEnumValueError(\"view\", "+data.ViewExpr+", []any{")
		for _, value := range quotedViews(data.Views) {
			lines = append(lines, "\t\t\t"+value+",")
		}
		lines = append(lines, "\t\t}))")
		lines = append(lines, "}")
		lines = append(lines, "return "+data.ReturnVar)
	case data.IsCollection:
		lines = append(lines, data.ReturnVar+" := make("+data.TargetType+", len("+data.ArgVar+"))")
		lines = append(lines, "for i, n := range "+data.ArgVar+" {")
		lines = append(lines, "\t"+data.ReturnVar+"[i] = "+data.InitName+"(n)")
		lines = append(lines, "}")
		lines = append(lines, "return "+data.ReturnVar)
	default:
		if data.Code != "" {
			lines = append(lines, data.Code)
		}
		for _, field := range data.Fields {
			lines = append(lines, "if "+data.Source+"."+field.VarName+" != nil {")
			lines = append(lines, "\t"+data.Target+"."+field.VarName+" = "+field.FieldInit+"("+data.Source+"."+field.VarName+")")
			lines = append(lines, "}")
		}
		lines = append(lines, "return "+data.ReturnVar)
	}
	return strings.Join(lines, "\n")
}

func quotedViewCase(viewName string) string {
	if viewName == expr.DefaultView {
		return `"default", ""`
	}
	return fmt.Sprintf("%q", viewName)
}

func quotedViews(views []*ViewData) []string {
	quoted := make([]string, 0, len(views))
	for _, view := range views {
		quoted = append(quoted, fmt.Sprintf("%q", view.Name))
	}
	return quoted
}

func needsValidationFieldSpacer(fields []validateFieldTemplateData) bool {
	if len(fields) == 0 {
		return false
	}
	for _, field := range fields {
		if field.IsRequired {
			return false
		}
	}
	return true
}

func fullTypeRefForAttribute(scope *codegen.NameScope, att *expr.AttributeExpr, defaultPkg string) string {
	pkg := defaultPkg
	if loc := codegen.UserTypeLocation(att.Type); loc != nil {
		pkg = loc.PackageName()
	}
	return scope.GoFullTypeRef(att, pkg)
}

// wrapProjected builds a viewed result type by wrapping the given projected
// in a result type with "projected" and "view" attributes.
func wrapProjected(projected expr.UserType) expr.UserType {
	rt := projected.(*expr.ResultTypeExpr)
	pratt := &expr.NamedAttributeExpr{
		Name:      "projected",
		Attribute: &expr.AttributeExpr{Type: rt, Description: "Type to project"},
	}
	prview := &expr.NamedAttributeExpr{
		Name:      "view",
		Attribute: &expr.AttributeExpr{Type: expr.String, Description: "View to render"},
	}
	return &expr.ResultTypeExpr{
		UserTypeExpr: &expr.UserTypeExpr{
			AttributeExpr: &expr.AttributeExpr{
				Type:       &expr.Object{pratt, prview},
				Validation: &expr.ValidationExpr{Required: []string{"projected", "view"}},
			},
			TypeName: rt.TypeName,
		},
		Identifier: rt.Identifier,
		Views:      rt.Views,
	}
}

// buildTypeInits builds the data to generate the constructor code to
// initialize a result type from a projected type.
func buildTypeInits(projected, att *expr.AttributeExpr, viewspkg string, scope, viewScope *codegen.NameScope) []*InitData {
	prt := projected.Type.(*expr.ResultTypeExpr)
	pobj := expr.AsObject(projected.Type)
	parr := expr.AsArray(projected.Type)
	if parr != nil {
		pobj = expr.AsObject(parr.ElemType.Type)
	}

	init := make([]*InitData, 0, len(prt.Views))
	for _, view := range prt.Views {
		var typ expr.DataType
		obj := &expr.Object{}
		walkViewAttrs(pobj, view, func(name string, att, _ *expr.AttributeExpr) {
			obj.Set(name, att)
		})
		typ = obj
		if parr != nil {
			typ = &expr.Array{ElemType: &expr.AttributeExpr{
				Type: &expr.ResultTypeExpr{
					UserTypeExpr: &expr.UserTypeExpr{
						AttributeExpr: &expr.AttributeExpr{Type: obj},
						TypeName:      scope.GoTypeName(parr.ElemType),
					},
				},
			}}
		}
		src := &expr.AttributeExpr{
			Type: &expr.ResultTypeExpr{
				UserTypeExpr: &expr.UserTypeExpr{
					AttributeExpr: &expr.AttributeExpr{Type: typ},
					TypeName:      scope.GoTypeName(projected),
				},
				Views:      prt.Views,
				Identifier: prt.Identifier,
			},
		}

		srcCtx := projectedTypeContext(viewspkg, true, viewScope)
		tgtCtx := typeContext(scope)
		resvar := scope.GoTypeName(att)
		name := projectedResultInitHelperBaseName(scope, viewScope, att, projected)
		if view.Name != expr.DefaultView {
			name += codegen.Goify(view.Name, true)
		}
		code, helpers := buildConstructorCode(src, att, "vres", "res", srcCtx, tgtCtx, view.Name)

		pkg := ""
		if loc := codegen.UserTypeLocation(att.Type); loc != nil {
			pkg = loc.PackageName()
		}
		init = append(init, &InitData{
			Name:          name,
			Description:   fmt.Sprintf("%s converts projected type %s to service type %s.", name, viewScope.GoTypeName(projected), resvar),
			Args:          []*InitArgData{{Name: "vres", Ref: viewScope.GoFullTypeRef(projected, viewspkg)}},
			ReturnTypeRef: scope.GoFullTypeRef(att, pkg),
			Code:          code,
			Helpers:       helpers,
		})
	}
	return init
}

// buildProjections builds the data to generate the constructor code to
// project a result type to a projected type based on a view.
func buildProjections(projected, att *expr.AttributeExpr, viewspkg string, scope, viewScope *codegen.NameScope) []*InitData {
	rt := att.Type.(*expr.ResultTypeExpr)
	projections := make([]*InitData, 0, len(rt.Views))
	for _, view := range rt.Views {
		var typ expr.DataType
		obj := &expr.Object{}
		pobj := expr.AsObject(projected.Type)
		parr := expr.AsArray(projected.Type)
		if parr != nil {
			pobj = expr.AsObject(parr.ElemType.Type)
		}
		walkViewAttrs(pobj, view, func(name string, att, _ *expr.AttributeExpr) {
			obj.Set(name, att)
		})
		typ = obj
		if parr != nil {
			typ = &expr.Array{ElemType: &expr.AttributeExpr{
				Type: &expr.ResultTypeExpr{
					UserTypeExpr: &expr.UserTypeExpr{
						AttributeExpr: &expr.AttributeExpr{Type: obj},
						TypeName:      parr.ElemType.Type.Name(),
					},
				},
			}}
		}
		tgt := &expr.AttributeExpr{
			Type: &expr.ResultTypeExpr{
				UserTypeExpr: &expr.UserTypeExpr{
					AttributeExpr: &expr.AttributeExpr{Type: typ},
					TypeName:      projected.Type.Name(),
				},
				Views:      rt.Views,
				Identifier: rt.Identifier,
			},
		}

		srcCtx := typeContext(scope)
		tgtCtx := projectedTypeContext(viewspkg, true, viewScope)
		tname := scope.GoTypeName(projected)
		name := projectionHelperBaseName(scope, att)
		if view.Name != expr.DefaultView {
			name += codegen.Goify(view.Name, true)
		}
		code, helpers := buildConstructorCode(att, tgt, "res", "vres", srcCtx, tgtCtx, view.Name)

		pkg := ""
		if loc := codegen.UserTypeLocation(att.Type); loc != nil {
			pkg = loc.PackageName()
		}
		projections = append(projections, &InitData{
			Name:          name,
			Description:   fmt.Sprintf("%s projects result type %s to projected type %s using the %q view.", name, scope.GoTypeName(att), tname, view.Name),
			Args:          []*InitArgData{{Name: "res", Ref: scope.GoFullTypeRef(att, pkg)}},
			ReturnTypeRef: viewScope.GoFullTypeRef(projected, viewspkg),
			Code:          code,
			Helpers:       helpers,
		})
	}
	return projections
}

func projectedResultInitHelperBaseName(scope, viewScope *codegen.NameScope, att, projected *expr.AttributeExpr) string {
	return "New" + scope.GoTypeName(att) + "From" + viewScope.GoTypeName(projected)
}

func projectionHelperBaseName(scope *codegen.NameScope, att *expr.AttributeExpr) string {
	return "Project" + scope.GoTypeName(att)
}

// buildValidations builds the data required to generate validations for the
// projected types.
func buildValidations(projected *expr.AttributeExpr, scope *codegen.NameScope) []*ValidateData {
	ut := projected.Type.(expr.UserType)
	tname := scope.GoTypeName(projected)
	var validations []*ValidateData
	if rt, isrt := ut.(*expr.ResultTypeExpr); isrt {
		arr := expr.AsArray(projected.Type)
		for _, view := range rt.Views {
			data := viewedResultValidateTemplateData{
				Projected:    tname,
				ArgVar:       "result",
				Source:       "result",
				IsCollection: arr != nil,
			}
			var vn string
			name := "Validate" + tname
			if view.Name != expr.DefaultView {
				vn = codegen.Goify(view.Name, true)
				name += vn
			}

			if arr != nil {
				data.Source = "item"
				data.ValidateVar = "Validate" + scope.GoTypeName(arr.ElemType) + vn
			} else {
				var fields []validateFieldTemplateData
				o := &expr.Object{}
				walkViewAttrs(expr.AsObject(projected.Type), view, func(name string, attr, vatt *expr.AttributeExpr) {
					if rt, ok := attr.Type.(*expr.ResultTypeExpr); ok {
						vw := ""
						if v, ok := vatt.Meta.Last(expr.ViewMetaKey); ok && v != expr.DefaultView {
							vw = v
						}
						fields = append(fields, validateFieldTemplateData{
							Name:        name,
							ValidateVar: "Validate" + scope.GoTypeName(attr) + codegen.Goify(vw, true),
							IsRequired:  rt.Attribute().IsRequired(name),
						})
					} else {
						o.Set(name, attr)
					}
				})
				ctx := projectedTypeContext("", !expr.IsPrimitive(projected.Type), scope)
				data.Validate = codegen.ValidationCode(&expr.AttributeExpr{Type: o, Validation: rt.Validation}, rt, ctx, true, false, true, "result")
				data.Fields = fields
			}

			validations = append(validations, &ValidateData{
				Name:        name,
				Description: fmt.Sprintf("%s runs the validations defined on %s using the %q view.", name, tname, view.Name),
				Ref:         scope.GoTypeRef(projected),
				Validate:    renderValidateTypeCode(data),
			})
		}
	} else {
		name := "Validate" + tname
		ctx := projectedTypeContext("", !expr.IsPrimitive(projected.Type), scope)
		validations = append(validations, &ValidateData{
			Name:        name,
			Description: fmt.Sprintf("%s runs the validations defined on %s.", name, tname),
			Ref:         scope.GoTypeRef(projected),
			Validate:    codegen.ValidationCode(ut.Attribute(), ut, ctx, true, expr.IsAlias(ut), true, "result"),
		})
	}
	return validations
}

// buildConstructorCode builds the transformation code to create a projected
// type from a service type and vice versa.
//
// source and target contains the projected/service contextual attributes
//
// sourceVar and targetVar contains the variable name that holds the source and
// target data structures in the transformation code.
//
// view is used to generate the constructor function name.
func buildConstructorCode(src, tgt *expr.AttributeExpr, sourceVar, targetVar string, sourceCtx, targetCtx *codegen.AttributeContext, view string) (string, []*codegen.TransformFunctionData) {
	var helpers []*codegen.TransformFunctionData
	rt := src.Type.(*expr.ResultTypeExpr)
	arr := expr.AsArray(tgt.Type)

	data := viewedResultInitTemplateData{
		ArgVar:       sourceVar,
		ReturnVar:    targetVar,
		IsCollection: arr != nil,
		TargetType:   targetCtx.Scope.Name(tgt, targetCtx.Pkg(tgt), targetCtx.Pointer, targetCtx.UseDefault),
	}

	if arr != nil {
		init := "new" + targetCtx.Scope.Name(arr.ElemType, "", targetCtx.Pointer, targetCtx.UseDefault)
		if view != "" && view != expr.DefaultView {
			init += codegen.Goify(view, true)
		}
		data.InitName = init
		return renderInitTypeCode(data), helpers
	}

	targetRTs := &expr.Object{}
	tatt := expr.DupAtt(tgt)
	tobj := expr.AsObject(tatt.Type)
	for _, nat := range *tobj {
		if _, ok := nat.Attribute.Type.(*expr.ResultTypeExpr); ok {
			targetRTs.Set(nat.Name, nat.Attribute)
			tobj.Delete(nat.Name)
		}
	}
	data.Source = sourceVar
	data.Target = targetVar

	code, helpers, err := codegen.GoTransform(src, tatt, sourceVar, targetVar, sourceCtx, targetCtx, "transform", true)
	if err != nil {
		panic(err)
	}
	data.Code = code

	if view != "" {
		data.InitName = targetCtx.Scope.Name(src, "", targetCtx.Pointer, targetCtx.UseDefault)
	}
	fields := make([]initFieldTemplateData, 0, len(*targetRTs))
	for _, nat := range *targetRTs {
		finit := "new" + targetCtx.Scope.Name(nat.Attribute, "", targetCtx.Pointer, targetCtx.UseDefault)
		if view != "" {
			v := ""
			if vatt := rt.View(view).Find(nat.Name); vatt != nil {
				if attv, ok := vatt.Meta.Last(expr.ViewMetaKey); ok && attv != expr.DefaultView {
					v = attv
				}
			}
			finit += codegen.Goify(v, true)
		}
		fields = append(fields, initFieldTemplateData{
			VarName:   codegen.Goify(nat.Name, true),
			FieldInit: finit,
		})
	}
	data.Fields = fields
	return renderInitTypeCode(data), helpers
}

// walkViewAttrs iterates through the attributes in att that are found in the
// given view and executes the walker function.
func walkViewAttrs(obj *expr.Object, view *expr.ViewExpr, walker func(name string, attr, vatt *expr.AttributeExpr)) {
	for _, nat := range *expr.AsObject(view.Type) {
		if attr := obj.Attribute(nat.Name); attr != nil {
			walker(nat.Name, attr, nat.Attribute)
		}
	}
}

// removeMeta removes the meta attributes from the given attribute. This is
// needed to make sure that any field name overriding is removed when
// generating protobuf types (as protogen itself won't honor these overrides).
func removeMeta(att *expr.AttributeExpr) {
	_ = codegen.Walk(att, func(a *expr.AttributeExpr) error {
		delete(a.Meta, "struct:pkg:path")
		return nil
	})
}
