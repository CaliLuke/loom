package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

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

func executeValidateTypeTemplate(data viewedResultValidateTemplateData) string {
	return renderValidateTypeCode(data)
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
				required := rt.Attribute()
				var fields []validateFieldTemplateData
				o := &expr.Object{}
				walkViewAttrs(expr.AsObject(projected.Type), view, func(name string, attr, vatt *expr.AttributeExpr) {
					if _, ok := attr.Type.(*expr.ResultTypeExpr); ok {
						vw := ""
						if v, ok := vatt.Meta.Last(expr.ViewMetaKey); ok && v != expr.DefaultView {
							vw = v
						}
						fields = append(fields, validateFieldTemplateData{
							Name:        name,
							ValidateVar: "Validate" + scope.GoTypeName(attr) + codegen.Goify(vw, true),
							IsRequired:  required.IsRequired(name),
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
