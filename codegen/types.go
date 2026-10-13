package codegen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

// GoNativeTypeName returns the Go built-in type corresponding to the given
// primitive type. GoNativeType panics if t is not a primitive type.
func GoNativeTypeName(t expr.DataType) string {
	switch t.Kind() {
	case expr.BooleanKind:
		return "bool"
	case expr.IntKind:
		return "int"
	case expr.Int32Kind:
		return "int32"
	case expr.Int64Kind:
		return "int64"
	case expr.UIntKind:
		return "uint"
	case expr.UInt32Kind:
		return "uint32"
	case expr.UInt64Kind:
		return "uint64"
	case expr.Float32Kind:
		return "float32"
	case expr.Float64Kind:
		return "float64"
	case expr.StringKind:
		return "string"
	case expr.BytesKind:
		return "[]byte"
	case expr.AnyKind:
		return "any"
	default:
		panic(fmt.Sprintf("cannot compute native Go type for %T", t)) // bug
	}
}

// AttributeTags computes the struct field tags from its metadata if any.
func AttributeTags(_, att *expr.AttributeExpr) string {
	tags := make(map[string]string)
	for key, val := range att.Meta {
		if strings.HasPrefix(key, "struct:tag:") && key != "struct:tag:json:name" {
			tags[key[11:]] = strings.Join(val, ",")
		}
	}
	return StructTag(tags)
}

// AttributeTagsWithName computes tags for the canonical service layout, with
// materialized defaults and ordinary pointer selection. Full JSON tags override
// automatic omission; name-only tags omit absence without discarding empties.
// Declaration emitters with another layout pass their actual carrier internally.
func AttributeTagsWithName(parent *expr.AttributeExpr, fieldName string, att *expr.AttributeExpr) string {
	if att == nil {
		return ""
	}
	policy := encodingmeta.JSONFieldPolicy{Storage: att.JSONFieldStorage(false, IsExplicitPresenceType(att) || expr.AllowsNull(att))}
	if parent != nil && fieldName != "" {
		policy = expr.ServiceFieldLayout(parent, fieldName, att, false, true).JSON
	}
	return attributeTagsForLayout(fieldName, att, policy)
}

func attributeTagsForLayout(fieldName string, att *expr.AttributeExpr, policy encodingmeta.JSONFieldPolicy) string {
	tags := attributeTagValues(att)
	field := encodingmeta.ResolveJSONField(att.Meta, expr.AttributeName(fieldName), policy)
	if field.Present {
		tags["json"] = field.Tag
	}
	return StructTag(tags)
}

// JSONFieldName returns the effective JSON member name for an object field.
// A complete json struct tag takes precedence over struct:tag:json:name; the
// authored field name is used when neither metadata key is present.
func JSONFieldName(fieldName string, att *expr.AttributeExpr) string {
	if att == nil {
		return fieldName
	}
	return expr.JSONFieldName(fieldName, att)
}

// StructTag renders tags as a Go struct field tag preceded by a space, or
// returns "" when tags is empty. Keys are emitted in sorted order and must be
// valid struct tag keys; design validation rejects any other struct:tag key.
// Each value is quoted with strconv.Quote so reflect.StructTag.Get returns it
// unchanged whatever it contains. The tag is a raw string literal unless it
// contains a character a raw string cannot hold, such as a backtick, in which
// case it is an interpreted string literal.
func StructTag(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	elems := make([]string, 0, len(keys))
	for _, key := range keys {
		elems = append(elems, key+":"+strconv.Quote(tags[key]))
	}
	tag := strings.Join(elems, " ")
	if strconv.CanBackquote(tag) {
		return " `" + tag + "`"
	}
	return " " + strconv.Quote(tag)
}

func attributeTagValues(att *expr.AttributeExpr) map[string]string {
	tags := make(map[string]string)
	for key, values := range att.Meta {
		if strings.HasPrefix(key, "struct:tag:") && key != "struct:tag:json:name" {
			tags[key[11:]] = strings.Join(values, ",")
		}
	}
	return tags
}
