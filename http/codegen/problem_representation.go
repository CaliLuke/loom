package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

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
		return problem, context, "problem"
	}
	return att, serviceContext(data.Service.LocationPackageName(loc), data.Service.Scope), "res"
}

func (b *errorBuilder) buildProblemClientResult(errorResponse *transportir.ResponseStatus, args []*InitArgData) string {
	problem, targetContext := problemRepresentation()
	locals := codegen.NewNameScope()
	locals.Unique("v")
	for _, arg := range args {
		locals.Unique(arg.VarName)
	}
	variable := locals.Unique("problem")
	body := responseStatusBody(errorResponse)
	var code strings.Builder
	var fields []*InitArgData
	origin, selected := body.Meta["origin:attribute"]
	if body.Type != expr.Empty && !selected {
		plan, err := codegen.GoTransform(body, problem, "body", variable, b.httpclictx, targetContext, "unmarshal", true)
		if err != nil {
			panic(codegen.NewError(b.sds.Ctx, body, fmt.Errorf("project HTTP problem response: %w", err)))
		}
		b.sd.ClientTransformHelpers = codegen.AppendHelpers(b.sd.ClientTransformHelpers, plan.Helpers)
		code.WriteString(plan.Code)
		code.WriteByte('\n')
	} else {
		name := targetContext.Scope.Name(problem, targetContext.DefaultPkg, false, true)
		fmt.Fprintf(&code, "%s := &%s{}\n", variable, name)
		if selected {
			_, member := problem.FindAttribute(origin[0])
			fields = append(fields, &InitArgData{AttributeData: &AttributeData{
				VarName:      "body",
				Type:         body.Type,
				FieldName:    codegen.GoifyAtt(member, origin[0], true),
				FieldType:    member.Type,
				FieldPointer: targetContext.IsPrimitivePointer(origin[0], problem),
			}})
		}
	}
	for _, arg := range args {
		if arg.FieldName != "" {
			fields = append(fields, arg)
		}
	}
	code.WriteString(initFieldAssignments(fields, variable, targetContext.DefaultPkg, targetContext.Scope.Scope()).Code)
	fmt.Fprintf(&code, "v := loomhttp.ProblemErrorFromBody(*%s)", variable)
	return code.String()
}
