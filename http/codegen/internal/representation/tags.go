package representation

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func objectFieldOptional(ma *expr.MappedAttributeExpr, name string, ptr, useDefault bool) bool {
	switch {
	case ptr:
		return true
	case useDefault:
		return !ma.IsRequired(name) && !ma.HasDefaultValue(name)
	default:
		return !ma.IsRequired(name)
	}
}

// AttributeTags computes the struct field tags.
func AttributeTags(att *expr.AttributeExpr, t string, optional, omitZero bool) string {
	var omitEmpty string
	// Always use omitempty for JSON-RPC ID attributes, even when required
	// since it is part of a different top-level field in the transport
	if optional || isJSONRPCID(att) {
		omitEmpty = ",omitempty"
	}
	jsonOption := omitEmpty
	if omitZero {
		jsonOption = ",omitzero"
	}
	jsonName := t
	explicitJSON := false
	if att != nil && att.Meta != nil {
		if v := att.Meta["struct:tag:json"]; len(v) > 0 {
			jsonName = strings.Join(v, ",")
			explicitJSON = true
		}
		if v := att.Meta["struct:tag:json:name"]; len(v) > 0 && v[0] != "" {
			if !explicitJSON {
				jsonName = strings.Join(v, ",")
			}
		}
	}
	if omitZero && strings.Split(jsonName, ",")[0] == "-" {
		panic(codegen.NewError(nil, att, fmt.Errorf("JSON field %q uses presence semantics and cannot be omitted with tag '-'", t)))
	}
	jsonName = MergeJSONOmitOption(jsonName, strings.TrimPrefix(jsonOption, ","))
	if custom := mergedAttributeTags(att, jsonName, omitZero || explicitJSON || hasJSONTagName(att)); custom != "" {
		return custom
	}
	return codegen.StructTag(map[string]string{
		"form": t + omitEmpty,
		"json": jsonName,
		"xml":  t + omitEmpty,
	})
}

func mergedAttributeTags(att *expr.AttributeExpr, jsonTag string, includeJSON bool) string {
	if att == nil || att.Meta == nil {
		return ""
	}
	tags := make(map[string]string)
	for key, values := range att.Meta {
		if !strings.HasPrefix(key, "struct:tag:") || key == "struct:tag:json:name" {
			continue
		}
		tags[strings.TrimPrefix(key, "struct:tag:")] = strings.Join(values, ",")
	}
	if len(tags) == 0 {
		return ""
	}
	if includeJSON {
		tags["json"] = jsonTag
	}
	return codegen.StructTag(tags)
}

func hasJSONTagName(att *expr.AttributeExpr) bool {
	if att == nil || att.Meta == nil {
		return false
	}
	return len(att.Meta["struct:tag:json:name"]) > 0
}

// MergeJSONOmitOption replaces omission options while preserving the exact ignore tag.
func MergeJSONOmitOption(tag, option string) string {
	if tag == "-" || option == "" {
		return tag
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

// isJSONRPCID checks if the attribute is marked as a JSON-RPC ID attribute
func isJSONRPCID(att *expr.AttributeExpr) bool {
	if att.Meta == nil {
		return false
	}
	_, ok := att.Meta["jsonrpc:id"]
	return ok
}

// FieldOmission is shared by emitted tags and semantic target plans.
func FieldOmission(parent *expr.MappedAttributeExpr, name string, attribute *expr.AttributeExpr, pointer, useDefault, jsonPresence bool) (bool, bool) {
	optional := objectFieldOptional(parent, name, pointer, useDefault)
	omitZero := !parent.IsRequiredNoDefault(name) && (jsonPresence || codegen.IsExplicitPresenceType(attribute) || expr.AllowsNull(attribute))
	return optional, omitZero
}
