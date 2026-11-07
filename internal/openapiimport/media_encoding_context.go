package openapiimport

import (
	"maps"

	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/CaliLuke/loom/internal/openapiref"
)

type mediaSchemaContext struct {
	encoding map[string]httpencoding.Serialization
	active   map[*Schema]bool
}

// mergeMediaEncoding returns detached effective metadata. Explicit outer slots
// override local slots; an empty Encoding Object keeps the source type default.
func mergeMediaEncoding(outer, inner map[string]httpencoding.Serialization) map[string]httpencoding.Serialization {
	result := maps.Clone(inner)
	if result == nil {
		result = make(map[string]httpencoding.Serialization, len(outer))
	}
	for name, occurrence := range outer {
		value := result[name]
		if occurrence.MediaType != "" {
			value.MediaType = occurrence.MediaType
		}
		if occurrence.Style != "" {
			value.Style = occurrence.Style
		}
		if occurrence.Explode != nil {
			value.Explode = occurrence.Explode
		}
		if occurrence.AllowReserved != nil {
			value.AllowReserved = occurrence.AllowReserved
		}
		result[name] = value
	}
	return result
}

// mediaEncodingFor resolves occurrence-first slots through aliases without
// modifying declarations shared by other media occurrences.
func (p *documentPlanner) mediaEncodingFor(schema *Schema) map[string]httpencoding.Serialization {
	var result map[string]httpencoding.Serialization
	seen := make(map[*Schema]bool)
	for schema != nil && !seen[schema] {
		seen[schema] = true
		result = mergeMediaEncoding(result, schema.MediaEncoding)
		if schema.Ref == "" {
			break
		}
		name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		if err != nil {
			break
		}
		schema = p.schemas[name].Schema
	}
	return result
}

// mediaPropertyNames collects the field namespace of one media schema. A root
// union's Encoding Object may name a field present in only one alternative.
func (p *documentPlanner) mediaPropertyNames(schema *Schema, names map[string]bool, seen map[*Schema]bool) {
	schema = p.urlSchema(schema)
	if schema == nil || seen[schema] {
		return
	}
	seen[schema] = true
	for _, branch := range schema.OneOf {
		p.mediaPropertyNames(branch, names, seen)
	}
	properties := make(map[string]*Schema)
	p.formProperties(schema, properties, make(map[*Schema]bool))
	for name := range properties {
		names[name] = true
	}
}
