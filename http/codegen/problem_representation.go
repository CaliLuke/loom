package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// ProblemSourcePreparation derives the public document once from the resolved
// native occurrence. It preserves nullable presence before physical projection.
func (r *ResponseData) ProblemSourcePreparation(errorVariable string) string {
	if !r.EncodePlan.NeedsProblemSource {
		return ""
	}
	var status string
	if r.JSONRPCCode != nil {
		status = fmt.Sprint(*r.JSONRPCCode)
	} else {
		status = responseWireStatus(r)
	}
	value := fmt.Sprintf("loomhttp.NewProblemResponse(ctx, %s, %s, %s, %s)", errorVariable, status, r.ProblemTypeOverride, r.ProblemTitleOverride)
	if !r.ErrorPayload {
		return "problem := " + value
	}
	return "var problem loom.Nullable[loomhttp.ProblemResponse]\nif res.IsNull() {\nproblem.SetNull()\n} else if _, ok := res.Value(); ok {\nproblem.SetValue(*" + value + ")\n}"
}

// problemRepresentation describes the runtime public document, independently of
// ServiceError and of the prepared body projection. Its private scope preserves
// the fixed name of the external runtime type.
func problemRepresentation() (*expr.AttributeExpr, *codegen.AttributeContext) {
	shape := expr.DupAtt(expr.ErrorResult.AttributeExpr)
	attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
		TypeName:      "ProblemResponse",
		AttributeExpr: shape,
	}}
	return attribute, serviceContext("loomhttp", codegen.NewNameScope())
}

// responseBodySource resolves the physical input before the prepared body is
// projected. Built-in server errors use the public runtime document; other
// responses retain their declared service representation.
func responseBodySource(att *expr.AttributeExpr, loc *codegen.Location, server bool, data *ServiceData) (*expr.AttributeExpr, *codegen.AttributeContext, string) {
	if server && expr.IsDefaultErrorResult(att.Type) {
		problem, context := problemRepresentation()
		problem.Nullable = expr.IsNullable(att)
		return problem, context, "problem"
	}
	return att, serviceContext(data.Service.LocationPackageName(loc), data.Service.Scope), "res"
}

func (b *errorBuilder) buildProblemClientResult(errorResponse *transportir.ResponseStatus, clientBody *TypeData, args []*InitArgData) codegen.TransformPlan {
	problem, targetContext := problemRepresentation()
	locals := codegen.NewNameScope()
	locals.Unique("v")
	for _, arg := range args {
		locals.Unique(arg.VarName)
	}
	variable := locals.Unique("problem")
	body := responseStatusBody(errorResponse)
	var code strings.Builder
	var errorAware bool
	var fields []*InitArgData
	origin, selected := body.Meta["origin:attribute"]
	nullable := expr.IsNullable(errorResponse.Error.Attribute)
	wholeNullable := nullable && clientBody != nil && !selected && expr.IsNullable(clientBody.attribute)
	problem.Nullable = wholeNullable
	if clientBody != nil && !selected {
		plan, err := codegen.GoTransform(clientBody.attribute, problem, "body", variable, clientBody.context.Dup(), targetContext, "unmarshal", true)
		if err != nil {
			panic(codegen.NewError(b.sds.Ctx, body, fmt.Errorf("project HTTP problem response: %w", err)))
		}
		b.sd.ClientTransformHelpers = codegen.AppendHelpers(b.sd.ClientTransformHelpers, plan.Helpers)
		errorAware = plan.ErrorAware
		code.WriteString(plan.Code)
		code.WriteByte('\n')
	} else {
		name := targetContext.Scope.Name(problem, targetContext.DefaultPkg, &codegen.AttributeContext{UseDefault: true})
		fmt.Fprintf(&code, "%s := &%s{}\n", variable, name)
		if selected {
			_, member := problem.FindAttribute(origin[0])
			fields = append(fields, &InitArgData{attribute: clientBody.attribute, context: clientBody.context, fieldAttribute: member, AttributeData: &AttributeData{
				VarName:      "body",
				Type:         body.Type,
				FieldName:    codegen.GoifyAtt(member, origin[0], true),
				FieldType:    member.Type,
				FieldPointer: targetContext.IsPrimitivePointer(origin[0], problem),
			}})
		}
	}
	fieldPlan := problemFieldPlan(problem, targetContext, body, fields, args, variable, wholeNullable)
	b.sd.ClientTransformHelpers = codegen.AppendHelpers(b.sd.ClientTransformHelpers, fieldPlan.Helpers)
	errorAware = errorAware || fieldPlan.ErrorAware
	code.WriteString(fieldPlan.Code)
	switch {
	case nullable:
		errorAware = true
		code.WriteString(nullableProblemResult(variable, wholeNullable, problemCodeConveyed(body, args), errorResponse.Error.Name))
	case errorAware:
		attribute := errorResponse.Error.Attribute
		pkg := b.svc.LocationPackageName(codegen.UserTypeLocation(attribute.Type))
		ref := b.svc.Scope.GoFullTypeRef(attribute, pkg)
		fmt.Fprintf(&code, "var v %s\nif *transformErr == nil {\nv = loomhttp.ProblemErrorFromBody(*%s)\n}\n", ref, variable)
	default:
		fmt.Fprintf(&code, "v := loomhttp.ProblemErrorFromBody(*%s)", variable)
	}
	return codegen.TransformPlan{Code: code.String(), ErrorAware: errorAware}
}

// problemCodeConveyed is a projection fact, not an inference from a decoded zero.
func problemCodeConveyed(body *expr.AttributeExpr, args []*InitArgData) bool {
	if origin := body.Meta["origin:attribute"]; len(origin) > 0 {
		if origin[0] == "code" {
			return true
		}
	} else if object := expr.AsObject(body.Type); object != nil {
		for _, field := range *object {
			if expr.AttributeName(field.Name) == "code" && expr.JSONFieldName(expr.ElementName(field.Name), field.Attribute) != "-" {
				return true
			}
		}
	}
	for _, arg := range args {
		if arg.FieldName == "Code" {
			return true
		}
	}
	return false
}

// problemFieldPlan assigns decoded metadata only to concrete document storage.
func problemFieldPlan(problem *expr.AttributeExpr, context *codegen.AttributeContext, body *expr.AttributeExpr, fields, args []*InitArgData, variable string, nullable bool) codegen.TransformPlan {
	origin := body.Meta["origin:attribute"]
	for _, arg := range args {
		if arg.FieldName != "" {
			field := *arg
			fields = append(fields, &field)
		}
	}
	for _, arg := range fields {
		_, arg.fieldAttribute = problem.FindAttribute(arg.Name)
		if arg.fieldAttribute == nil && len(origin) > 0 && arg.VarName == "body" {
			_, arg.fieldAttribute = problem.FindAttribute(origin[0])
		}
	}
	if !nullable || len(fields) == 0 {
		return initFieldAssignments(fields, variable, context)
	}
	plan := initFieldAssignments(fields, "problemValue", context)
	plan.Code = fmt.Sprintf("if *transformErr == nil {\nif problemValue, problemHasValue := %s.Value(); problemHasValue {\n%s\n%s.SetValue(problemValue)\n} else if responseMetadataPresent {\n*transformErr = fmt.Errorf(\"null response body cannot carry headers or cookies\")\n}\n}\n", variable, plan.Code, variable)
	plan.ErrorAware = true
	return plan
}

func nullableProblemResult(variable string, wholeNullable, conveyed bool, name string) string {
	var code strings.Builder
	code.WriteString("var v loom.Nullable[loom.ServiceError]\nif *transformErr == nil {\n")
	if wholeNullable {
		fmt.Fprintf(&code, "if %s.IsNull() {\nv.SetNull()\n} else if value, ok := %s.Value(); ok {\n", variable, variable)
		code.WriteString("native := loomhttp.ProblemErrorFromBody(value)\n")
	} else {
		fmt.Fprintf(&code, "native := loomhttp.ProblemErrorFromBody(*%s)\n", variable)
	}
	if !conveyed {
		fmt.Fprintf(&code, "native.Name = %q\n", name)
	}
	code.WriteString("v.SetValue(*native)\n")
	if wholeNullable {
		code.WriteString("} else {\n*transformErr = loom.ErrAbsentNullable\n}\n")
	}
	code.WriteString("}\n")
	return code.String()
}
