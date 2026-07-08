package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

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
		ReturnsError:  true,
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
		ReturnsError:  true,
		Code:          executeInitTypeTemplate(resultInitTData),
	}, resref
}

func executeInitTypeTemplate(data viewedResultInitTemplateData) string {
	return renderInitTypeCode(data)
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
		lines = append(lines, "\t\treturn "+data.ReturnVar+", loom.InvalidEnumValueError(\"view\", "+data.ViewExpr+", []any{")
		for _, value := range quotedViews(data.Views) {
			lines = append(lines, "\t\t\t"+value+",")
		}
		lines = append(lines, "\t\t})")
		lines = append(lines, "}")
		lines = append(lines, "return "+data.ReturnVar+", nil")
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
