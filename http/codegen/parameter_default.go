package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// renderParameterDefault delegates default materialization to the shared Go
// representation owner. Its caller decides optional wire absence.
func renderParameterDefault(physical *expr.AttributeExpr, ctx *codegen.AttributeContext, value any, target, name string, pointer bool) string {
	plan, err := ctx.DefaultValue(physical, value)
	if err != nil {
		panic(fmt.Errorf("parameter %q default: %w", name, err))
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
		fmt.Fprintf(&code, "if parameterDefaultError != nil {\nreturn fmt.Errorf(\"%%s\", %q)\n}\n", "parameter "+name+": invalid default")
	}
	code.WriteString("}\n")
	return code.String()
}
