package codegen

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

var digits = regexp.MustCompile("[0-9]+")

var (
	// reserved protocol buffer keywords and package names
	reservedProtoBuf = map[string]struct{}{
		// types
		"bool":     {},
		"bytes":    {},
		"double":   {},
		"fixed32":  {},
		"fixed64":  {},
		"float":    {},
		"int32":    {},
		"int64":    {},
		"sfixed32": {},
		"sfixed64": {},
		"sint32":   {},
		"sint64":   {},
		"string":   {},
		"uint32":   {},
		"uint64":   {},

		// reserved
		"enum":     {},
		"import":   {},
		"map":      {},
		"message":  {},
		"oneof":    {},
		"option":   {},
		"package":  {},
		"public":   {},
		"repeated": {},
		"reserved": {},
		"returns":  {},
		"rpc":      {},
		"service":  {},
		"syntax":   {},
	}
)

// protoBufify makes a valid protocol buffer identifier out of any string. It
// returns the result of protoBufIdentifier with an underscore appended to
// protocol buffer keywords, which cannot name fields.
func protoBufify(str string, firstUpper, acronym bool) string {
	return fixReservedProtoBuf(protoBufIdentifier(str, firstUpper, acronym))
}

// protoBufIdentifier makes a valid protocol buffer identifier out of any
// string. Protocol buffer identifiers consist of ASCII letters, digits and
// underscores and start with a letter. protoBufIdentifier treats any other
// character, including any non-ASCII rune, as a word separator and removes
// it. It produces a "CamelCase" version of the string that protoc-gen-go
// uses unchanged as the Go name of the element. Following Goify,
// identifiers that would start with a digit receive the Val or val prefix,
// and a string without any ASCII letter or digit produces Val or val.
//
// If firstUpper is true the first character of the identifier is uppercase
// otherwise it's lowercase.
func protoBufIdentifier(str string, firstUpper, acronym bool) string {
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

	// Protocol buffer identifiers are ASCII only. Turn any other rune into
	// a word separator so that CamelCase never keeps it.
	str = strings.Map(asciiIdentifierRune, str)

	// The CamelCase implementation of protoc-gen-go considers digits as words
	// but our CamelCase implementation considers them as lower case characters,
	// compensate by adding an underscore after any series of digits.
	// See https://github.com/golang/protobuf/blob/d04d7b157bb510b1e0c10132224b616ac0e26b17/protoc-gen-go/generator/generator.go#L2648-L2685
	str = string(digits.ReplaceAllFunc([]byte(str), func(match []byte) []byte {
		res := make([]byte, len(match)+1) // need to allocate new slice
		copy(res, match)
		res[len(res)-1] = '_'
		return res
	}))

	str = codegen.CamelCase(str, firstUpper, acronym)
	if str == "" {
		// All characters are invalid. Produce a default value.
		if firstUpper {
			return "Val"
		}
		return "val"
	}
	if '0' <= str[0] && str[0] <= '9' {
		if firstUpper {
			str = "Val" + str
		} else {
			str = "val" + str
		}
	}

	return str
}

// protoMetaName returns the protocol buffer message name that the
// struct:name:proto metadata in meta selects and whether meta sets one.
func protoMetaName(meta expr.MetaExpr) (string, bool) {
	names := meta["struct:name:proto"]
	if len(names) == 0 {
		return "", false
	}
	return protoMetaMessageName(names[0]), true
}

// protoMetaMessageName returns the protocol buffer message name for the
// struct:name:proto metadata value name. ASCII values are returned unchanged
// because the design validates them as protocol buffer identifiers. protoc
// accepts only ASCII identifiers, so in any other value the non-ASCII runes
// are word separators as in protoBufIdentifier.
func protoMetaMessageName(name string) string {
	for i := 0; i < len(name); i++ {
		if name[i] >= utf8.RuneSelf {
			return protoBufIdentifier(name, true, true)
		}
	}
	return name
}

// protoBufifyAtt honors any struct:field:name meta set on the attribute and
// and calls protoBufify with the tag value if present or the given name
// otherwise.
func protoBufifyAtt(att *expr.AttributeExpr, name string, upper bool) string {
	if tname, ok := att.Meta["struct:field:name"]; ok {
		if len(tname) > 0 {
			name = tname[0]
		}
	}
	return protoBufify(name, upper, false)
}

// asciiIdentifierRune returns r when it is an ASCII rune and an underscore
// otherwise. It maps every non-ASCII rune, including the replacement rune of
// invalid UTF-8, to a word separator for protoBufify.
func asciiIdentifierRune(r rune) rune {
	if r > unicode.MaxASCII {
		return '_'
	}
	return r
}

// fixReservedProtoBuf appends an underscore on to protocol buffer reserved
// keywords.
func fixReservedProtoBuf(w string) string {
	if _, ok := reservedProtoBuf[codegen.CamelCase(w, false, false)]; ok {
		w += "_"
	}
	return w
}
