package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// renderStyledPathInitCode keeps the ordinary segment conversion for unchanged
// wildcards and adds fallible codecs for the newly admitted grammars.
func renderStyledPathInitCode(args []*InitArgData, path string) string {
	var code, format strings.Builder
	values := make([]string, len(args))
	last := 0
	for i, match := range expr.HTTPWildcardRegex.FindAllStringIndex(path, -1) {
		arg := args[i]
		format.WriteString(pathFormatLiteral(path[last:match[0]]))
		format.WriteString("/%v")
		last = match[1]
		if !arg.parameter.Styled {
			escape := "loomhttp.EscapePathSegment"
			if strings.HasPrefix(path[match[0]:], "/{*") {
				escape = "loomhttp.EscapePathRemainder"
			}
			values[i] = renderPathValue(&code, arg, escape)
			continue
		}
		value := arg.parameterValueVar
		segment := arg.parameterSegmentVar
		if expr.AsUnion(arg.Type) != nil {
			code.WriteString(renderTextUnionPath(arg, segment))
			values[i] = segment
			continue
		}
		// Constructor arguments are native values, including optional primitives
		// whose presence was resolved by the request builder before this call.
		code.WriteString(renderParameterValue(arg.parameter.attribute, arg.parameter.attribute, arg.parameter.context, arg.VarName, value, false, `"", `))
		fmt.Fprintf(&code, "%s, parameterError := loomhttp.EncodePathParam(%q, %s, %s)\nif parameterError != nil {\nreturn \"\", parameterError\n}\n", segment, arg.parameter.HTTPName, parameterCodecLiteral(arg.parameter.Codec), value)
		values[i] = segment
	}
	format.WriteString(pathFormatLiteral(path[last:]))
	fmt.Fprintf(&code, "return fmt.Sprintf(%q, %s), nil\n", format.String(), strings.Join(values, ", "))
	return code.String()
}
