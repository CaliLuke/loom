package codegen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
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

// AttributeTagsWithName computes the struct field tags from its metadata,
// interpreting the "struct:tag:json:name" key when present.
//
// The "struct:tag:json" meta key always takes precedence and is treated as a
// complete tag override value. When only "struct:tag:json:name" is set, Loom
// computes the json tag and appends ",omitempty" when the field is not
// required by its parent object. When no explicit JSON metadata is present,
// Loom emits a default json tag that preserves the DSL field name.
func AttributeTagsWithName(parent *expr.AttributeExpr, fieldName string, att *expr.AttributeExpr) string {
	if att == nil {
		return ""
	}
	tags, jsonName := attributeTagValues(att)
	normalizeAttributeJSONTag(tags, jsonName, parent, fieldName, att)
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

func attributeTagValues(att *expr.AttributeExpr) (map[string]string, string) {
	tags := make(map[string]string)
	var jsonName string
	keys := make([]string, 0, len(att.Meta))
	for k := range att.Meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		val := att.Meta[key]
		if !strings.HasPrefix(key, "struct:tag:") {
			continue
		}
		if key == "struct:tag:json:name" {
			if jsonName == "" && len(val) > 0 {
				jsonName = strings.Join(val, ",")
			}
			continue
		}
		name := key[11:]
		value := strings.Join(val, ",")
		tags[name] = value
		if name == "json" {
			jsonName = ""
		}
	}
	return tags, jsonName
}

func normalizeAttributeJSONTag(tags map[string]string, jsonName string, parent *expr.AttributeExpr, fieldName string, att *expr.AttributeExpr) {
	jsonTag, hasJSONTag := tags["json"]
	if !hasJSONTag {
		if jsonName == "" {
			jsonName = expr.AttributeName(fieldName)
		}
		if jsonName == "" {
			return
		}
		if parent != nil && fieldName != "" && !parent.IsRequired(fieldName) {
			jsonName = appendJSONOmitOption(jsonName, att)
		}
		tags["json"] = jsonName
		return
	}
	if parent != nil && fieldName != "" && !parent.IsRequired(fieldName) &&
		(IsExplicitPresenceType(att) || expr.AllowsNull(att)) {
		tags["json"] = appendJSONOmitOption(jsonTag, att)
	}
}

func appendJSONOmitOption(tag string, att *expr.AttributeExpr) string {
	option := "omitempty"
	if IsExplicitPresenceType(att) || expr.AllowsNull(att) {
		option = "omitzero"
	}
	parts := strings.Split(tag, ",")
	options := make([]string, 0, len(parts))
	seen := false
	for _, part := range parts[1:] {
		if part == "omitempty" || part == "omitzero" {
			if !seen {
				options = append(options, option)
				seen = true
			}
			continue
		}
		options = append(options, part)
	}
	if !seen {
		options = append(options, option)
	}
	return strings.Join(append([]string{parts[0]}, options...), ",")
}
