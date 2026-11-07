package openapiimport

import "github.com/CaliLuke/loom/internal/openapiref"

func (p *documentPlanner) formProperties(schema *Schema, properties map[string]*Schema, seen map[*Schema]bool) {
	schema = p.urlSchema(schema)
	if schema == nil || seen[schema] {
		return
	}
	seen[schema] = true
	for _, base := range schema.Bases {
		p.formProperties(base, properties, seen)
	}
	for _, property := range schema.Properties {
		properties[property.Name] = property.Schema
	}
}

func (p *documentPlanner) urlSchema(schema *Schema) *Schema {
	seen := make(map[string]bool)
	for schema != nil && schema.Ref != "" {
		name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		if err != nil || seen[name] {
			return nil
		}
		seen[name] = true
		schema = p.schemas[name].Schema
	}
	return schema
}
