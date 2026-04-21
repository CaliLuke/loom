package service

import (
	"fmt"

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
