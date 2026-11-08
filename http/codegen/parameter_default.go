package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// renderParameterDefault delegates default materialization to the shared Go
// representation owner. Its caller decides optional wire absence and supplies
// an enclosing function whose only result is an error: a default that cannot
// be materialized at runtime returns a server fault, never a client decode
// error. A default that cannot be rendered is a generation error attributed
// to physical.
func renderParameterDefault(physical *expr.AttributeExpr, ctx *codegen.AttributeContext, value any, target, name string, pointer bool) string {
	plan, err := ctx.DefaultValue(physical, value)
	if err != nil {
		panic(codegen.NewError(nil, physical, fmt.Errorf("parameter %q default: %w", name, err)))
	}
	var code strings.Builder
	code.WriteString("{\n")
	if plan.ErrorAware {
		code.WriteString("var parameterDefaultError error\ntransformErr := &parameterDefaultError\n")
	}
	if pointer {
		fmt.Fprintf(&code, "var parameterDefault %s = %s\n%s = &parameterDefault\n", ctx.Scope.Ref(physical, ctx.Pkg(physical)), plan.Expression, target)
	} else {
		fmt.Fprintf(&code, "%s = %s\n", target, plan.Expression)
	}
	if plan.ErrorAware {
		fmt.Fprintf(&code, "if parameterDefaultError != nil {\nreturn %s\n}\n", parameterDefaultFault(name))
	}
	code.WriteString("}\n")
	return code.String()
}

// parameterDefaultFault returns the server fault reported when an authored
// default of the parameter name cannot be materialized.
func parameterDefaultFault(name string) string {
	return fmt.Sprintf("loom.Fault(\"%%s\", %q)", "parameter "+name+": invalid default")
}

// renderElementDefault materializes the authored default of a parameter
// element in its native layout through renderParameterDefault.
func renderElementDefault(element *Element) string {
	return renderParameterDefault(element.attribute, element.context, element.DefaultValue, element.VarName, element.HTTPName, element.Pointer)
}
