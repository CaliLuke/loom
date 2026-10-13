package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func buildViewedResultValidation(projected *expr.AttributeExpr, views []*ViewData, scope *codegen.NameScope, ref, resvar string) *ValidateData {
	validateTData := viewedResultValidateTemplateData{
		Projected: scope.GoValueTypeName(projected),
		ArgVar:    "result",
		Source:    "result",
		Views:     views,
		IsViewed:  true,
		Root:      newViewValidationCarrier(projected, projectedTypeContext("", true, scope), expr.DefaultView),
	}
	validate := executeValidateTypeTemplate(validateTData)
	name := "Validate" + resvar
	return &ValidateData{
		Name:        name,
		Description: fmt.Sprintf("%s runs the validations defined on the viewed result type %s.", name, resvar),
		Ref:         ref,
		Validate:    validate,
	}
}

func executeValidateTypeTemplate(data viewedResultValidateTemplateData) string {
	return renderValidateTypeCode(data)
}

func renderValidateTypeCode(data viewedResultValidateTemplateData) string {
	var lines []string
	function := codegen.ValidationFunction{Argument: data.ArgVar}
	if data.IsViewed {
		return renderViewedResultValidation(data)
	}

	input := data.ArgVar
	if data.RootNullable {
		input = "actual"
	}
	if data.IsCollection {
		index := "_"
		if data.Element.nilable || data.Element.nullable {
			index = "i"
		}
		lines = append(lines, "for "+index+", "+data.Source+" := range "+input+" {")
		if data.Element.nilable || data.Element.nullable {
			lines = append(lines, "\tif "+data.Element.missing(data.Source)+" {")
			lines = append(lines, "\t\terr = loom.MergeErrors(err, loom.InvalidNullElementError("+function.Path("")+", i))")
			lines = append(lines, "\t\tcontinue", "\t}")
		}
		lines = append(lines, indentViewValidation(data.Element.validation(data.Source, function.Path("[*]"))))
		lines = append(lines, "}")
	} else {
		if data.Validate != "" {
			lines = append(lines, data.Validate)
		} else if needsValidationFieldSpacer(data.Fields) {
			lines = append(lines, "")
		}
		for _, field := range data.Fields {
			source := data.Source + "." + codegen.Goify(field.Name, true)
			if field.IsRequired {
				lines = append(lines, "if "+field.Carrier.missing(source)+" {")
				lines = append(lines, "\terr = loom.MergeErrors(err, loom.MissingFieldError("+fmt.Sprintf("%q", field.Name)+", "+function.Path("")+"))")
				lines = append(lines, "}")
			}
			validation := field.Carrier.validation(source, function.Path("."+field.Name))
			if field.Carrier.nullable {
				lines = append(lines, validation)
			} else {
				lines = append(lines, "if "+source+" != nil {", indentViewValidation(validation), "}")
			}
		}
	}
	body := strings.Join(lines, "\n")
	if data.RootNullable && strings.TrimSpace(body) != "" {
		return "if actual, ok := " + data.ArgVar + ".Value(); ok {\n" + indentViewValidation(body) + "\n}"
	}
	return body
}

func renderViewedResultValidation(data viewedResultValidateTemplateData) string {
	lines := make([]string, 0, 2*len(data.Views)+4)
	function := codegen.ValidationFunction{Argument: data.ArgVar}
	if data.Root.nullable {
		lines = append(lines, "if !"+data.ArgVar+".Projected.Present() {", "\treturn loom.Fault(\"missing result\")", "}")
	}
	lines = append(lines, "switch "+data.ArgVar+".View {")
	for _, view := range data.Views {
		caseLine := "case " + quotedViewCase(view.Name) + ":"
		lines = append(lines, "\t"+caseLine)
		validateName := "Validate" + data.Projected
		if view.Name != expr.DefaultView {
			validateName += codegen.Goify(view.Name, true)
		}
		if data.Root.nullable && !data.Root.intrinsic {
			carrier := data.Root
			carrier.function.Name = validateName
			lines = append(lines, indentViewValidation(indentViewValidation(carrier.validation(data.ArgVar+".Projected", function.Path("")))))
		} else {
			lines = append(lines, "\t\terr = "+(codegen.ValidationFunction{Name: validateName}).Call(data.ArgVar+".Projected", function.Path("")))
		}
	}
	lines = append(lines, "\tdefault:")
	lines = append(lines, "\t\terr = loom.InvalidEnumValueError(\"view\", "+data.Source+".View, []any{ "+strings.Join(quotedViews(data.Views), ", ")+" })")
	lines = append(lines, "}")
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
	tname := scope.GoValueTypeName(projected)
	ctx := projectedTypeContext("", !expr.IsPrimitive(projected.Type), scope)
	function := codegen.NamedValidationFunction(projected, ctx, "result")
	var validations []*ValidateData
	if rt, isrt := ut.(*expr.ResultTypeExpr); isrt {
		arr := expr.AsArray(projected.Type)
		for _, view := range rt.Views {
			data := viewedResultValidateTemplateData{
				Projected:    tname,
				ArgVar:       "result",
				Source:       "result",
				IsCollection: arr != nil,
				RootNullable: expr.IsNullable(projected),
			}
			var vn string
			name := function.Name
			if view.Name != expr.DefaultView {
				vn = codegen.Goify(view.Name, true)
				name += vn
			}

			if arr != nil {
				data.Source = "item"
				data.Element = newViewValidationCarrier(arr.ElemType, ctx, view.Name)
			} else {
				if data.RootNullable {
					data.Source = "actual"
				}
				data.Validate, data.Fields = buildViewObjectValidation(projected, rt, view, scope, data.Source)
			}

			validations = append(validations, &ValidateData{
				Name:        name,
				Description: fmt.Sprintf("%s runs the validations defined on %s using the %q view.", name, tname, view.Name),
				Ref:         function.Ref,
				Validate:    renderValidateTypeCode(data),
			})
		}
	} else {
		validations = append(validations, &ValidateData{
			Name:        function.Name,
			Description: fmt.Sprintf("%s runs the validations defined on %s.", function.Name, tname),
			Ref:         function.Ref,
			Validate:    codegen.ValidationCode(ut.Attribute(), ut, function.Context(ctx), true, expr.IsAlias(ut), true, "result"),
		})
	}
	return validations
}

func buildViewObjectValidation(projected *expr.AttributeExpr, rt *expr.ResultTypeExpr, view *expr.ViewExpr, scope *codegen.NameScope, source string) (string, []validateFieldTemplateData) {
	required := view.AttributeExpr
	ctx := projectedTypeContext("", !expr.IsPrimitive(projected.Type), scope)
	var fields []validateFieldTemplateData
	o := &expr.Object{}
	walkViewAttrs(expr.AsObject(projected.Type), view, func(name string, attr, vatt *expr.AttributeExpr) {
		if _, ok := attr.Type.(*expr.ResultTypeExpr); ok {
			vw := ""
			if v, ok := vatt.Meta.Last(expr.ViewMetaKey); ok && v != expr.DefaultView {
				vw = v
			}
			fields = append(fields, validateFieldTemplateData{
				Name:       name,
				Carrier:    newViewValidationCarrier(attr, ctx, vw),
				IsRequired: required.IsRequired(name),
			})
		} else {
			o.Set(name, attr)
		}
	})
	validate := codegen.ValidationCode(&expr.AttributeExpr{Type: o, Validation: view.Validation}, rt, (codegen.ValidationFunction{Argument: source}).Context(ctx), true, false, true, source)
	return validate, fields
}
