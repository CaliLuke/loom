package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func buildViewedResultInit(att, projected, envelope *expr.AttributeExpr, views []*ViewData, viewspkg string, scope, viewScope *codegen.NameScope, resvar, vresref string, isarr bool, validateName string) *InitData {
	initTData := viewedResultInitTemplateData{
		ToViewed:           true,
		ArgVar:             "res",
		ReturnVar:          "vres",
		Views:              views,
		ReturnTypeRef:      vresref,
		IsCollection:       isarr,
		IsObject:           expr.IsObject(att.Type),
		IsNullable:         expr.IsNullable(att),
		OccurrenceNullable: att.Nullable && !expr.IsNullable(&expr.AttributeExpr{Type: att.Type}),
		ProjectedRef:       viewScope.GoFullTypeRef(projected, viewspkg),
		TargetType:         scope.GoFullTypeName(envelope, viewspkg),
		InitName:           projectionHelperBaseName(scope, att),
		ValidateName:       validateName,
		ViewExpr:           "view",
	}
	name := "NewViewed" + resvar
	description := fmt.Sprintf("%s projects result type %s to viewed result type %s and validates the selected view. Invalid views and result constraints return a fault that wraps the validation error.", name, resvar, resvar)
	if initTData.IsNullable {
		description += " It returns a fault error when res is absent."
	} else if initTData.IsObject {
		description += " It returns a fault error when res is nil."
	}
	return &InitData{
		Name:        name,
		Description: description,
		Args: []*InitArgData{
			{Name: "res", Ref: fullTypeRefForAttribute(scope, att, "")},
			{Name: "view", Ref: "string"},
		},
		ReturnTypeRef: vresref,
		ReturnsError:  true,
		Code:          renderViewedResultInit(initTData),
	}
}

func buildViewedResultResultInit(att, projected *expr.AttributeExpr, views []*ViewData, scope, viewScope *codegen.NameScope, resvar, vresref string) (*InitData, string) {
	resref := fullTypeRefForAttribute(scope, att, "")
	resultInitTData := viewedResultInitTemplateData{
		ToResult:           true,
		IsObject:           expr.IsObject(att.Type),
		OccurrenceNullable: att.Nullable && !expr.IsNullable(&expr.AttributeExpr{Type: att.Type}),
		ArgVar:             "vres",
		ReturnVar:          "res",
		Views:              views,
		ReturnTypeRef:      resref,
		InitName:           projectedResultInitHelperBaseName(scope, viewScope, att, projected),
		ViewExpr:           "vres.View",
	}
	name := "New" + resvar
	return &InitData{
		Name:          name,
		Description:   fmt.Sprintf("%s initializes result type %s from viewed result type %s.", name, resvar, resvar),
		Args:          []*InitArgData{{Name: "vres", Ref: vresref}},
		ReturnTypeRef: resref,
		ReturnsError:  true,
		Code:          renderViewedResultInit(resultInitTData),
	}, resref
}

func renderViewedResultInit(data viewedResultInitTemplateData) string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, "var "+data.ReturnVar+" "+data.ReturnTypeRef)
	if data.ToViewed && (data.IsObject || data.IsNullable) {
		missing := data.ArgVar + " == nil"
		if data.IsNullable {
			missing = "!" + data.ArgVar + ".Present()"
		}
		lines = append(lines,
			"if "+missing+" {",
			"\treturn "+data.ReturnVar+", loom.Fault(\"missing result\")",
			"}",
		)
	}
	lines = append(lines, "switch "+data.ViewExpr+" {")
	for _, view := range data.Views {
		lines = append(lines, "\tcase "+quotedViewCase(view.Name)+":")
		initName := data.InitName
		if view.Name != expr.DefaultView {
			initName += codegen.Goify(view.Name, true)
		}
		if data.ToViewed {
			lines = append(lines, indentViewValidation(indentViewValidation(viewedResultProjectionCall(data, initName))))
			prefix := ""
			if !data.IsCollection {
				prefix = "&"
			}
			lines = append(lines, "\t\t"+data.ReturnVar+" = "+prefix+data.TargetType+"{Projected: p, View: "+fmt.Sprintf("%q", view.Name)+" }")
		} else {
			lines = append(lines, indentViewValidation(indentViewValidation(viewedResultProjectionCall(data, initName))))
		}
	}
	lines = append(lines, "\tdefault:")
	lines = append(lines, "\t\terr := loom.InvalidEnumValueError(\"view\", "+data.ViewExpr+", []any{")
	for _, value := range quotedViews(data.Views) {
		lines = append(lines, "\t\t\t"+value+",")
	}
	lines = append(lines, "\t\t})")
	if data.ToViewed {
		lines = append(lines, "\t\treturn "+data.ReturnVar+", loom.NewServiceError(err, \"fault\", false, false, true)")
	} else {
		lines = append(lines, "\t\treturn "+data.ReturnVar+", err")
	}
	lines = append(lines, "}")
	if data.ToViewed {
		lines = append(lines,
			"if err := "+data.ValidateName+"("+data.ReturnVar+"); err != nil {",
			"\tvar zero "+data.ReturnTypeRef,
			"\treturn zero, loom.NewServiceError(err, \"fault\", false, false, true)",
			"}",
		)
	}
	lines = append(lines, "return "+data.ReturnVar+", nil")
	return strings.Join(lines, "\n")
}

// viewedResultProjectionCall adapts the occurrence carrier to a canonical named
// helper. The envelope keeps presence; the named helper receives its declared
// concrete argument. Intrinsic nullable helpers already own that representation.
func viewedResultProjectionCall(data viewedResultInitTemplateData, name string) string {
	source, target, ref := data.ArgVar, "p", data.ProjectedRef
	assign := ":="
	if data.ToResult {
		source, target, ref = data.ArgVar+".Projected", data.ReturnVar, data.ReturnTypeRef
		assign = "="
	}
	if !data.OccurrenceNullable {
		return target + " " + assign + " " + name + "(" + source + ")"
	}
	argument, value := "actual", "converted"
	if data.IsObject {
		argument, value = "&actual", "*converted"
	}
	code := ""
	if data.ToViewed {
		code = "var " + target + " " + ref + "\n"
	}
	return code + "if " + source + ".IsNull() {\n\t" + target + ".SetNull()\n} else if actual, ok := " + source + ".Value(); ok {\n\tconverted := " + name + "(" + argument + ")\n\t" + target + ".SetValue(" + value + ")\n}"
}

// buildTypeInits builds the data to generate the constructor code to
// initialize a result type from a projected type.
func buildTypeInits(projected, att *expr.AttributeExpr, viewspkg string, scope, viewScope *codegen.NameScope) []*InitData {
	rt := att.Type.(*expr.ResultTypeExpr)
	inits := make([]*InitData, 0, len(rt.Views))
	for _, view := range rt.Views {
		name := projectedResultInitHelperBaseName(scope, viewScope, att, projected)
		if view.Name != expr.DefaultView {
			name += codegen.Goify(view.Name, true)
		}
		code, helpers := buildProjectionConversion(att, projected, view.Name, name, "vres", "res", projectedTypeContext(viewspkg, true, viewScope), typeContext(scope), false)
		pkg := ""
		if loc := codegen.UserTypeLocation(att.Type); loc != nil {
			pkg = scope.PackageName(loc)
		}
		inits = append(inits, &InitData{
			Name:          name,
			Description:   fmt.Sprintf("%s converts projected type %s to service type %s.", name, viewScope.GoTypeName(projected), scope.GoTypeName(att)),
			Args:          []*InitArgData{{Name: "vres", Ref: viewScope.GoFullTypeRef(projected, viewspkg)}},
			ReturnTypeRef: scope.GoFullTypeRef(att, pkg),
			Code:          code,
			Helpers:       helpers,
		})
	}
	return inits
}

// buildProjections builds selected-view conversions into the shared carriers.
func buildProjections(projected, att *expr.AttributeExpr, viewspkg string, scope, viewScope *codegen.NameScope) []*InitData {
	rt := att.Type.(*expr.ResultTypeExpr)
	projections := make([]*InitData, 0, len(rt.Views))
	for _, view := range rt.Views {
		name := projectionHelperBaseName(scope, att)
		if view.Name != expr.DefaultView {
			name += codegen.Goify(view.Name, true)
		}
		code, helpers := buildProjectionConversion(att, projected, view.Name, name, "res", "vres", typeContext(scope), projectedTypeContext(viewspkg, true, viewScope), true)
		pkg := ""
		if loc := codegen.UserTypeLocation(att.Type); loc != nil {
			pkg = scope.PackageName(loc)
		}
		projections = append(projections, &InitData{
			Name:          name,
			Description:   fmt.Sprintf("%s projects result type %s to projected type %s using the %q view.", name, scope.GoTypeName(att), scope.GoTypeName(projected), view.Name),
			Args:          []*InitArgData{{Name: "res", Ref: scope.GoFullTypeRef(att, pkg)}},
			ReturnTypeRef: viewScope.GoFullTypeRef(projected, viewspkg),
			Code:          code,
			Helpers:       helpers,
		})
	}
	return projections
}

func projectedResultInitHelperBaseName(scope, viewScope *codegen.NameScope, att, projected *expr.AttributeExpr) string {
	return "New" + scope.GoValueTypeName(att) + "From" + viewScope.GoValueTypeName(projected)
}

func projectionHelperBaseName(scope *codegen.NameScope, att *expr.AttributeExpr) string {
	return "Project" + scope.GoValueTypeName(att)
}
