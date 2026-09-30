// Package enumvalue projects resolved authored values onto their declared host shapes.
package enumvalue

import "github.com/CaliLuke/loom/expr"

// Normalize resolves a DSL value through the shared semantic owner and returns
// its detached declared host shape. It applies declared primitive precision,
// Bytes text conversion, finalized object names, typed map keys and retained
// union selection without invoking a custom codec. Nil collections use their
// declared empty shape and Any retains its raw builtin host representation.
//
// Normalize first uses full semantic resolution. When only root or named
// ancestry predicates exclude a structurally valid value, it projects the
// bounded declared shape without establishing semantic admission. Values that
// neither traversal can project and malformed declarations use
// CanonicalizeExample's structural fallback for compatibility with callers
// that separately validate or omit them. The fallback does not establish
// semantic acceptance and may retain borrowed opaque values.
func Normalize(attribute *expr.AttributeExpr, value any) any {
	if value == nil || attribute == nil || attribute.Type == nil {
		return value
	}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		return expr.CanonicalizeExample(attribute, value)
	}
	source := context.SupplyValue(expr.ValueInput{Raw: value, Origin: "enum value normalization"})
	result := context.Resolve(occurrence, source, expr.ValueRoleEnum)
	declared, ok := result.DeclaredJSONValue()
	if !ok {
		declared, ok = context.ResolveDeclaredShape(occurrence, source).DeclaredJSONValue()
	}
	if !ok {
		return expr.CanonicalizeExample(attribute, value)
	}
	return declared
}
