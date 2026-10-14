package codegen

import (
	"fmt"
	"strconv"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// stringRepresentationCrossing separates copying a physical value from invoking
// authored semantic codecs. A copy of identical custom storage needs no codec.
func stringRepresentationCrossing(source, target *expr.AttributeExpr, ta *TransformAttrs) bool {
	if !representation.CustomString(source) && !representation.CustomString(target) {
		return false
	}
	return ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source)) != ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target))
}

func transformStringExpression(source, target *expr.AttributeExpr, value, path string, ta *TransformAttrs) string {
	ref := ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target))
	locals := representationLocals(ref + " " + value)
	result := locals.Unique("convertedValue")
	text := locals.Unique("semanticText")
	failure := locals.Unique("conversionError")
	code := fmt.Sprintf("func() %s {\nvar %s %s\nif *transformErr != nil {\nreturn %s\n}\n", ref, result, ref, result)
	if representation.CustomString(source) {
		code += fmt.Sprintf("%s, %s := loom.StringValue(%s, %s)\nif %s != nil {\n*transformErr = %s\nreturn %s\n}\n", text, failure, value, strconv.Quote(path), failure, failure, result)
	} else {
		code += fmt.Sprintf("%s := string(%s)\n", text, value)
	}
	if representation.CustomString(target) {
		declaration := ":="
		if representation.CustomString(source) {
			declaration = "="
		}
		code += fmt.Sprintf("%s, %s %s loom.ParseStringValue[%s](%s, %s)\nif %s != nil {\n*transformErr = %s\n}\nreturn %s\n}()", result, failure, declaration, ref, text, strconv.Quote(path), failure, failure, result)
	} else {
		code += fmt.Sprintf("return %s(%s)\n}()", ref, text)
	}
	return code
}
