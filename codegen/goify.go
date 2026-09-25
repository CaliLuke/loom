package codegen

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
)

// Goify makes a valid Go identifier out of any string. It does that by removing
// any non letter and non digit character and by making sure the first character
// is a letter or "_". Goify produces a "CamelCase" version of the string, if
// firstUpper is true the first character of the identifier is uppercase
// otherwise it's lowercase. Identifiers that would start with a digit receive
// the Val or val prefix. When firstUpper is true the result is always an
// exported Go identifier: a first letter without an upper case form, such as
// one from a caseless script, also receives the Val prefix.
func Goify(str string, firstUpper bool) string {
	return naming.Goify(str, firstUpper)
}

// GoifyAtt honors any struct:field:name meta set on the attribute and calls
// Goify with the tag value if present or the given name otherwise.
func GoifyAtt(att *expr.AttributeExpr, name string, upper bool) string {
	if tname, ok := att.Meta["struct:field:name"]; ok {
		if len(tname) > 0 {
			name = tname[0]
		}
	}
	return Goify(name, upper)
}

// UnionValTypeName returns the Go type name of the interface and method used to
// type the union.
func UnionValTypeName(unionName string) string {
	return Goify(unionName+"Val", false)
}
