package openapi

import (
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// ToString returns the string representation of the given type.
func ToString(val any) string {
	switch actual := val.(type) {
	case string:
		return actual
	case int:
		return strconv.Itoa(actual)
	case float64:
		return strconv.FormatFloat(actual, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(actual)
	default:
		panic("unexpected key type")
	}
}

// ToStringMap converts map[any]any to a map[string]any
// when possible.
func ToStringMap(val any) any {
	switch actual := val.(type) {
	case map[any]any:
		m := make(map[string]any)
		for k, v := range actual {
			m[ToString(k)] = ToStringMap(v)
		}
		return m
	case []any:
		mapSlice := make([]any, len(actual))
		for i, e := range actual {
			mapSlice[i] = ToStringMap(e)
		}
		return mapSlice
	default:
		return actual
	}
}

func applySchemaOpenAPIMetadata(s *Schema, meta expr.MetaExpr) {
	if s == nil || meta == nil {
		return
	}
	if value, ok := meta.Last("openapi:readOnly"); ok {
		s.ReadOnly = metaBoolValue(value)
	}
	if value, ok := meta.Last("openapi:writeOnly"); ok {
		s.WriteOnly = metaBoolValue(value)
	}
	if value, ok := meta.Last("openapi:deprecated"); ok {
		s.Deprecated = metaBoolValue(value)
	}
	if value, ok := meta.Last("openapi:contentEncoding"); ok {
		s.ContentEncoding = value
	}
	if value, ok := meta.Last("openapi:contentMediaType"); ok {
		s.ContentMediaType = value
	}
	if value, ok := meta.Last("openapi:discriminator:defaultMapping"); ok && s.Discriminator != nil {
		s.Discriminator.DefaultMapping = value
	}
	if value, ok := meta.Last("openapi:discriminator:optional"); ok && metaBoolValue(value) && s.Discriminator != nil {
		s.Discriminator.Optional = true
		s.Required = slices.DeleteFunc(s.Required, func(name string) bool {
			return name == s.Discriminator.PropertyName
		})
	}
	for key, assign := range map[string]func(*XML, string){
		"openapi:xml:name":      func(xml *XML, value string) { xml.Name = value },
		"openapi:xml:namespace": func(xml *XML, value string) { xml.Namespace = value },
		"openapi:xml:prefix":    func(xml *XML, value string) { xml.Prefix = value },
		"openapi:xml:nodeType":  func(xml *XML, value string) { xml.NodeType = value },
	} {
		if value, ok := meta.Last(key); ok {
			if s.XML == nil {
				s.XML = new(XML)
			}
			assign(s.XML, value)
		}
	}
}

func metaBoolValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "true":
		return true
	case "false":
		return false
	default:
		return true
	}
}

// MustGenerate returns true if the meta indicates that a OpenAPI specification should be
// generated, false otherwise.
func MustGenerate(meta expr.MetaExpr) bool {
	m, ok := meta.Last("openapi:generate")
	if ok && m == "false" {
		return false
	}
	return true
}

// AdditionalPropertiesFromExpr extracts the OpenAPI additionalProperties.
func AdditionalPropertiesFromExpr(meta expr.MetaExpr) any {
	m, ok := meta.Last("openapi:additionalProperties")
	if ok && m == "false" {
		return false
	}
	return nil
}

// ClosedObjectModeFromExpr reports whether OpenAPI closed object mode is
// enabled via metadata.
func ClosedObjectModeFromExpr(meta expr.MetaExpr) bool {
	m, ok := meta.Last("openapi:closed-objects")
	return ok && m == "true"
}
