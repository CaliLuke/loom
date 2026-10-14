package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen/service"
)

// writeServerStreamBodySelection selects the body of the already validated
// result res. emit uses the selected concrete body before transport assignment
// erases its type; mapped SSE data can therefore access the body's fields.
func writeServerStreamBodySelection(b *strings.Builder, bodies []*TypeData, viewed *service.ViewedResultTypeData, emit func(*TypeData)) {
	if viewed == nil {
		emit(bodies[0])
		return
	}
	if viewed.ViewName != "" {
		emit(viewedServerBody(bodies, viewed.ViewName))
		return
	}
	b.WriteString("\tswitch res.View {\n")
	for _, view := range viewed.Views {
		if view.Name == "default" {
			fmt.Fprintf(b, "\tcase %q, \"\":\n", view.Name)
		} else {
			fmt.Fprintf(b, "\tcase %q:\n", view.Name)
		}
		emit(viewedServerBody(bodies, view.Name))
	}
	b.WriteString("\t}\n")
}

func writeServerBodyInitCall(b *strings.Builder, body *TypeData, prefix string) {
	if body.Init.ServerErrorAware {
		prefix = strings.Replace(prefix, "body :=", "body, conversionError :=", 1)
	}
	fmt.Fprintf(b, "%s%s(", prefix, body.Init.Name)
	for _, arg := range body.Init.ServerArgs {
		fmt.Fprintf(b, "%s, ", arg.Ref)
	}
	b.WriteString(")\n")
	if body.Init.ServerErrorAware {
		b.WriteString("if conversionError != nil {\nreturn conversionError\n}\n")
	}
}
