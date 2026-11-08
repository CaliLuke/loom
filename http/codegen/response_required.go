package codegen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// responseRequiredValidation applies captured branch membership to actual
// native storage. Presence and pointer decisions remain owned by the shared
// attribute context and service layout; empty supplied collections are valid.
func responseRequiredValidation(response *ResponseData, carrier *expr.AttributeExpr, context *codegen.AttributeContext, value, view string) string {
	render := func(required []string) string {
		var out strings.Builder
		for _, name := range required {
			key, attribute := carrier.FindAttribute(name)
			if attribute == nil {
				panic("captured response requirement has no native member")
			}
			member := value + "." + context.Scope.Field(attribute, key, true)
			presence := context.FieldPresence(expr.NewObjectFields(carrier), key, attribute)
			var absent string
			switch {
			case presence == codegen.OptionalPresence || presence == codegen.NullablePresence:
				absent = "!" + member + ".Present()"
			case expr.ServiceFieldLayout(carrier, key, attribute, context.Pointer, context.UseDefault).Pointer,
				expr.IsArray(attribute.Type), expr.IsMap(attribute.Type), attribute.Type.Kind() == expr.BytesKind, expr.IsAny(attribute.Type):
				absent = member + " == nil"
			}
			if absent != "" {
				fmt.Fprintf(&out, "if %s {\nreturn loom.MissingFieldError(%q, \"response\")\n}\n", absent, name)
			}
		}
		return out.String()
	}
	if view == "" {
		return render(response.Required[""])
	}
	keys := make([]string, 0, len(response.Required))
	for key := range response.Required {
		if key != "" {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	var out strings.Builder
	fmt.Fprintf(&out, "switch %s {\n", view)
	for _, key := range keys {
		fmt.Fprintf(&out, "case %q", key)
		if key == "default" {
			out.WriteString(", \"\"")
		}
		out.WriteString(":\n")
		out.WriteString(render(response.Required[key]))
	}
	out.WriteString("default:\nreturn fmt.Errorf(\"unknown response view\")\n}\n")
	return out.String()
}
