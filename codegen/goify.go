package codegen

import (
	"go/doc"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/loom/expr"
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
	// Optimize trivial case
	if str == "" {
		return ""
	}

	// Remove optional suffix that defines corresponding transport specific
	// name.
	idx := strings.Index(str, ":")
	if idx > 0 {
		str = str[:idx]
	}

	str = CamelCase(str, firstUpper, true)
	if str == "" {
		// All characters are invalid. Produce a default value.
		if firstUpper {
			return "Val"
		}
		return "val"
	}
	first, size := utf8.DecodeRuneInString(str)
	switch {
	case unicode.IsDigit(first):
		if firstUpper {
			str = "Val" + str
		} else {
			str = "val" + str
		}
	case firstUpper && !unicode.IsUpper(first):
		str = exportIdentifier(first, str[size:])
	}
	return fixReservedGo(str)
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

// exportIdentifier returns an exported identifier made of first followed by
// rest. Go only exports identifiers that start with an upper case letter
// (Unicode class Lu), so a title case first letter is converted to upper
// case and a letter without an upper case form, such as one from a caseless
// script, receives the Val prefix.
func exportIdentifier(first rune, rest string) string {
	if upper := unicode.ToUpper(first); unicode.IsUpper(upper) {
		return string(upper) + rest
	}
	return "Val" + string(first) + rest
}

// fixReservedGo appends an underscore on to Go reserved keywords.
func fixReservedGo(w string) string {
	if doc.IsPredeclared(w) || token.IsKeyword(w) || isPackage[w] {
		w += "_"
	}
	return w
}

var (
	isPackage = map[string]bool{
		// stdlib and Loom packages used by generated code
		"errors": true,
		"fmt":    true,
		"http":   true,
		"json":   true,
		"os":     true,
		"url":    true,
		"time":   true,
	}
)
