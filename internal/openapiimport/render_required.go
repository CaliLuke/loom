package openapiimport

import "fmt"

func (r *renderer) schemaRequired(schema *Schema, path string) error {
	if len(schema.Required) == 0 {
		return nil
	}
	typed := r.objectAdmitsTypedMembers(schema)
	for _, name := range schema.Required {
		// A typed policy, authored or inherited, supplies the value of an
		// undeclared required member.
		if !typed && !r.schemaDefinesProperty(schema, name, make(map[*Schema]bool)) {
			return fmt.Errorf("render OpenAPI design: %s required property %q is not defined", path, name)
		}
	}
	r.quotedCall("Required", schema.Required)
	return nil
}

func (r *renderer) schemaDefinesProperty(schema *Schema, name string, seen map[*Schema]bool) bool {
	if schema == nil || seen[schema] {
		return false
	}
	seen[schema] = true
	for _, property := range schema.Properties {
		if property.Name == name {
			return true
		}
	}
	if schema.Ref != "" {
		base, err := r.resolveSchemaReference(schema, "required")
		if err == nil && r.schemaDefinesProperty(base, name, seen) {
			return true
		}
	}
	for _, base := range schema.Bases {
		if r.schemaDefinesProperty(base, name, seen) {
			return true
		}
	}
	return false
}
