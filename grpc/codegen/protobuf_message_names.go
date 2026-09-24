package codegen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// protoMessageNames holds the names that the fields, oneofs and oneof
	// fields of a protocol buffer message take. They share one namespace,
	// so newProtoMessageNames makes the oneof names and oneof field names
	// unique in the message. The proto file, the field number checks, the
	// Go conversion code and the validation code all use these names.
	protoMessageNames struct {
		// fields maps the name of each attribute of the message object,
		// without the mapping suffix that follows a colon, to its protocol
		// buffer name. The name of a union attribute is the
		// name of its oneof.
		fields map[string]string
		// branches maps the name of each union attribute, without the
		// mapping suffix, to the protocol
		// buffer names of the fields of its oneof, in branch order.
		branches map[string][]string
		// conflict describes the first two attributes that are not unions
		// and have the same protocol buffer name, if any. Such names come
		// from the design and are not changed.
		conflict *protoNameConflict
	}

	// protoNameConflict describes two attributes of a message that have the
	// same protocol buffer name.
	protoNameConflict struct {
		name, first, second string
	}
)

// newProtoMessageNames returns the protocol buffer names of the fields,
// oneofs and oneof fields of the message with the attributes obj.
//
// An attribute that is not a union keeps its name. Such attributes take their
// names first, then the union attributes take theirs in declaration order, so
// the first oneof or oneof field to use a name keeps it. A union attribute is
// a oneof named after the attribute, followed by "_oneof" as many times as
// needed to differ from the names of its branches and from the names already
// taken. A oneof field is named after its branch unless the name is taken, in
// which case it takes the name of the union attribute and an underscore as a
// prefix, as many times as needed. A prefixed name contains an underscore, so
// it is never a protocol buffer keyword and drops the underscore that
// protoBufify appends to keywords. A name is taken when it, or the Go name
// that protoc-gen-go makes of it, is used already.
func newProtoMessageNames(obj *expr.Object) *protoMessageNames {
	names := &protoMessageNames{
		fields:   make(map[string]string, len(*obj)),
		branches: make(map[string][]string),
	}
	owners := make(map[string]string, len(*obj))
	goNames := make(map[string]struct{}, len(*obj))
	taken := func(name string) bool {
		if _, ok := owners[name]; ok {
			return true
		}
		_, ok := goNames[protoGoName(name)]
		return ok
	}
	claim := func(name, owner string) {
		owners[name] = owner
		goNames[protoGoName(name)] = struct{}{}
	}
	for _, nat := range *obj {
		if expr.IsUnion(nat.Attribute.Type) {
			continue
		}
		name := protoFieldName(nat.Name)
		owner := fmt.Sprintf("attribute %q", nat.Name)
		if other, ok := owners[name]; ok && names.conflict == nil {
			names.conflict = &protoNameConflict{name: name, first: other, second: owner}
		}
		claim(name, owner)
		names.fields[protoNameKey(nat.Name)] = name
	}
	for _, nat := range *obj {
		union := expr.AsUnion(nat.Attribute.Type)
		if union == nil {
			continue
		}
		oneof := protoFieldName(nat.Name)
		prefix := strings.TrimSuffix(oneof, "_") + "_"
		branches := make([]string, len(union.Values))
		for i, branch := range union.Values {
			branches[i] = protoFieldName(branch.Name)
		}
		for taken(oneof) || slices.Contains(branches, oneof) {
			oneof += "_oneof"
		}
		claim(oneof, fmt.Sprintf("oneof of attribute %q", nat.Name))
		names.fields[protoNameKey(nat.Name)] = oneof
		for i, branch := range union.Values {
			for taken(branches[i]) {
				branches[i] = prefix + strings.TrimSuffix(branches[i], "_")
			}
			claim(branches[i], fmt.Sprintf("branch %q of attribute %q", branch.Name, nat.Name))
		}
		names.branches[protoNameKey(nat.Name)] = branches
	}
	return names
}

// protoNameKey returns the key of the attribute name in the maps of
// protoMessageNames: the name without the suffix that follows a colon, which
// protoBufify ignores as well. Callers such as the transforms, which walk the
// mapped attributes of an object, look names up without the suffix.
func protoNameKey(name string) string {
	if i := strings.Index(name, ":"); i > 0 {
		return name[:i]
	}
	return name
}

// protoFieldName returns the protocol buffer name of the message field,
// oneof or oneof field that name describes.
func protoFieldName(name string) string {
	return codegen.SnakeCase(protoBufify(name, false, false))
}

// protoGoName returns the name of the Go struct field that protoc-gen-go
// generates for the protocol buffer field or oneof name, which also suffixes
// the Go type of a oneof field. It follows the GoCamelCase function of
// google.golang.org/protobuf for the names that protoFieldName returns, which
// consist of ASCII letters, digits and underscores: an underscore followed
// by a lowercase letter is removed, and a letter that starts a word is made
// uppercase.
func protoGoName(name string) string {
	b := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c == '_' && i == 0:
			b = append(b, 'X')
		case c == '_' && i+1 < len(name) && isASCIILower(name[i+1]):
		case '0' <= c && c <= '9':
			b = append(b, c)
		default:
			if isASCIILower(c) {
				c -= 'a' - 'A'
			}
			b = append(b, c)
			for ; i+1 < len(name) && isASCIILower(name[i+1]); i++ {
				b = append(b, name[i+1])
			}
		}
	}
	return string(b)
}

// isASCIILower reports whether c is an ASCII lowercase letter.
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// goField returns the name of the Go struct field that protoc-gen-go
// generates for the attribute name of the message, the field of the oneof
// interface for a union attribute.
func (n *protoMessageNames) goField(name string) string {
	return protoGoName(n.field(name))
}

// goBranches returns the names of the Go fields of the oneof fields of the
// union attribute name of the message in branch order. protoc-gen-go also
// names the Go wrapper type of each oneof field after the message and this
// name.
func (n *protoMessageNames) goBranches(name string) []string {
	branches := n.oneofFields(name)
	res := make([]string, len(branches))
	for i, branch := range branches {
		res[i] = protoGoName(branch)
	}
	return res
}

// field returns the protocol buffer name of the attribute name of the
// message, the name of the oneof for a union attribute.
func (n *protoMessageNames) field(name string) string {
	return n.fields[protoNameKey(name)]
}

// oneofFields returns the protocol buffer names of the oneof fields of the
// union attribute name of the message in branch order.
func (n *protoMessageNames) oneofFields(name string) []string {
	return n.branches[protoNameKey(name)]
}
