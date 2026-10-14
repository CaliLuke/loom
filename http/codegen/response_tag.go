package codegen

import (
	"fmt"
	"strconv"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// ResponseTagData lowers one semantic selector for its actual result carrier.
// Server and client renderers consume these expressions without guessing names,
// pointer storage, presence wrappers or nominal string types.
type ResponseTagData struct {
	// Field is the Go field identifier in the service or projected result.
	Field string
	// Match is the complete condition, including an optional if initializer.
	Match string
	// Assignment reconstructs the selected tag in the decoded result.
	Assignment string
}

func responseTagData(tag *expr.HTTPResponseTag, result *expr.AttributeExpr, viewed bool, context *codegen.AttributeContext) *ResponseTagData {
	if tag == nil {
		return nil
	}
	key, attribute := result.FindAttribute(tag.Key)
	if attribute == nil {
		panic(fmt.Sprintf("response Tag %q is missing from its physical result representation", tag.Key))
	}
	name := context.Scope.Field(attribute, expr.AttributeName(key), true)
	server, client := "res."+name, "res."+name
	if viewed {
		server, client = "res.Projected."+name, "p."+name
	}
	literal := strconv.Quote(tag.Value)
	data := &ResponseTagData{Field: name}
	switch context.FieldPresence(expr.NewMappedAttributeExpr(result), expr.AttributeName(key), attribute) {
	case codegen.NullablePresence, codegen.OptionalPresence:
		data.Match = "tag, ok := " + server + ".Value(); ok && tag == " + literal
		data.Assignment = client + ".SetValue(" + literal + ")"
	default:
		if context.IsPrimitivePointer(key, result) {
			data.Match = server + " != nil && *" + server + " == " + literal
			declaration := "tmp := " + literal
			if attribute.Type != expr.String {
				declaration = "var tmp " + context.Scope.Ref(attribute, context.Pkg(attribute)) + " = " + literal
			}
			data.Assignment = declaration + "\n" + client + " = &tmp"
		} else {
			data.Match = server + " == " + literal
			data.Assignment = client + " = " + literal
		}
	}
	return data
}
