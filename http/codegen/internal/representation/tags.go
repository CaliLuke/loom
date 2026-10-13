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
	tags, err := emittedFieldTags(FieldTags(attribute, expr.ElementName(name), false, encodingmeta.JSONFieldPolicy{}), name)
	if err != nil {
		return "", "", err
	}
	jsonName := emittedJSONFieldName(attribute, name, tags)
	formName, _, _ := strings.Cut(tags.Get("form"), ",")
	return jsonName, formName, nil
}

// FieldTags resolves all codec tags from the actual JSON field policy.
// Non-JSON codecs retain their existing empty-value omission contract.
func FieldTags(att *expr.AttributeExpr, name string, optional bool, policy encodingmeta.JSONFieldPolicy) FieldEncoding {
	var omitEmpty string
	if optional || isJSONRPCID(att) {
		omitEmpty = ",omitempty"
	}
	policy.Optional = policy.Optional || isJSONRPCID(att)
	policy.ForceOmission = isJSONRPCID(att)
	field := encodingmeta.ResolveJSONField(att.Meta, name, policy)
	_, explicitJSON := att.Meta["struct:tag:json"]
	if policy.Presence && strings.Split(field.Tag, ",")[0] == "-" {
		panic(codegen.NewError(nil, att, fmt.Errorf("JSON field %q uses presence semantics and cannot be omitted with tag '-'", name)))
	}
	includeJSON := policy.Presence || explicitJSON || hasJSONTagName(att)
	if custom := mergedAttributeTags(att, field.Tag, includeJSON); custom != "" {
		if !includeJSON {
			field = encodingmeta.JSONField{}
		}
		return FieldEncoding{Tags: custom, JSON: field}
	}
	return FieldEncoding{Tags: codegen.StructTag(map[string]string{
		"form": name + omitEmpty,
		"json": field.Tag,
		"xml":  name + omitEmpty,
	}), JSON: field}
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

func emittedFieldTags(encoding FieldEncoding, name string) (reflect.StructTag, error) {
	unquoted, err := strconv.Unquote(strings.TrimSpace(encoding.Tags))
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
