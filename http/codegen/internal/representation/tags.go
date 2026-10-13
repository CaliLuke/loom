package representation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

// FieldWireNames returns the actual JSON and form names emitted for a field.
// A name of "-" excludes that codec; an empty form name is not form-encoded.
// JSON's empty tag name falls back to the emitted Go field name.
func FieldWireNames(attribute *expr.AttributeExpr, name string) (string, string, error) {
	tags, err := emittedFieldTags(attribute, name, false, false)
	if err != nil {
		return "", "", err
	}
	jsonName := emittedJSONFieldName(attribute, name, tags)
	formName, _, _ := strings.Cut(tags.Get("form"), ",")
	return jsonName, formName, nil
}

// AttributeTags computes the struct field tags.
func AttributeTags(att *expr.AttributeExpr, t string, optional, omitZero bool) string {
	var omitEmpty string
	// Always use omitempty for JSON-RPC ID attributes, even when required
	// since it is part of a different top-level field in the transport
	if optional || isJSONRPCID(att) {
		omitEmpty = ",omitempty"
	}
	jsonName, _ := encodingmeta.JSONTag(att.Meta, t, encodingmeta.JSONTagPolicy{
		Optional:      optional || omitZero || isJSONRPCID(att),
		Presence:      omitZero,
		ForceOmission: isJSONRPCID(att),
	})
	_, explicitJSON := att.Meta["struct:tag:json"]
	if omitZero && strings.Split(jsonName, ",")[0] == "-" {
		panic(codegen.NewError(nil, att, fmt.Errorf("JSON field %q uses presence semantics and cannot be omitted with tag '-'", t)))
	}
	if custom := mergedAttributeTags(att, jsonName, omitZero || explicitJSON || hasJSONTagName(att)); custom != "" {
		return custom
	}
	return codegen.StructTag(map[string]string{
		"form": t + omitEmpty,
		"json": jsonName,
		"xml":  t + omitEmpty,
	})
}

// FieldOmission is shared by emitted tags and semantic target plans.
func FieldOmission(parent *expr.MappedAttributeExpr, name string, attribute *expr.AttributeExpr, pointer, useDefault, jsonPresence bool) (bool, bool) {
	optional := objectFieldOptional(parent, name, pointer, useDefault)
	omitZero := !parent.IsRequiredNoDefault(name) && (jsonPresence || codegen.IsExplicitPresenceType(attribute) || expr.AllowsNull(attribute))
	return optional, omitZero
}

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

// isJSONRPCID checks if the attribute is marked as a JSON-RPC ID attribute
func isJSONRPCID(att *expr.AttributeExpr) bool {
	if att.Meta == nil {
		return false
	}
	_, ok := att.Meta["jsonrpc:id"]
	return ok
}

func emittedFieldTags(attribute *expr.AttributeExpr, name string, optional, omitZero bool) (reflect.StructTag, error) {
	tags := AttributeTags(attribute, expr.ElementName(name), optional, omitZero)
	unquoted, err := strconv.Unquote(strings.TrimSpace(tags))
	if err != nil {
		return "", fmt.Errorf("HTTP field %q tags: %w", name, err)
	}
	return reflect.StructTag(unquoted), nil
}

func emittedJSONFieldName(attribute *expr.AttributeExpr, name string, tags reflect.StructTag) string {
	wireName, _, _ := strings.Cut(tags.Get("json"), ",")
	if wireName == "" {
		wireName = codegen.GoifyAtt(attribute, expr.AttributeName(name), true)
	}
	return wireName
}
