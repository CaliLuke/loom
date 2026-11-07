package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// responseNativeSource identifies the native payload inside the server's
// response carrier. A viewed result's wrapper is not its native value.
func responseNativeSource(viewed bool) string {
	if viewed {
		return "res.Projected"
	}
	return "res"
}

// responseBodyAbsence captures parent and selected-member suppliedness separately
// in the physical conversion context. A replacement body may bypass only the
// selected member; native prerequisites and metadata retain their own contract.
func responseBodyAbsence(attribute, body *expr.AttributeExpr, context *codegen.AttributeContext, source string, viewed, server bool, data *ServiceData) (parent, member string) {
	if !server {
		return "", ""
	}
	field := selectBodyField(attribute, body)
	if field.Name == "" {
		return "", ""
	}
	if viewed {
		context = viewContext(data.Service.ViewsPkg, data.Service.ViewScope)
		source = responseNativeSource(true)
	}
	nullable := expr.IsNullable(attribute)
	if nullable {
		attribute = representation.ConcreteAttribute(attribute)
		source = "responseValue"
	}
	member = sourceMemberAbsence(attribute, field, context, source)
	if !nullable {
		parent = sourceValueAbsence(attribute, source, strings.HasPrefix(context.Scope.Ref(attribute, context.Pkg(attribute)), "*"))
	}
	return parent, member
}

// selectedPreparation keeps parent suppliedness separate from the selected
// member. A formatter can replace that member, but never the native metadata.
func (plan *ResponseEncodePlan) selectedPreparation(onError func(string) string, bodyCondition string) string {
	var code strings.Builder
	for _, body := range plan.bodies {
		conditions := []string{body.sourceParentAbsence, body.sourceAbsence}
		if bodyCondition != "" && conditions[1] != "" {
			conditions[1] = "(" + bodyCondition + ") && (" + conditions[1] + ")"
		}
		for _, condition := range conditions {
			if condition == "" {
				continue
			}
			if plan.UseViewedBodySwitch {
				view := fmt.Sprintf("res.View == %q", body.View)
				if body.View == "default" {
					view += ` || res.View == ""`
				}
				condition = "(" + view + ") && (" + condition + ")"
			}
			fmt.Fprintf(&code, "if %s {\n%s\n}\n", condition, onError(`fmt.Errorf("selected response value is absent")`))
		}
	}
	return code.String()
}
