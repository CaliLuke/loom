package openapiimport

import (
	"fmt"
	"go/token"
)

func (r *renderer) index() error {
	for _, schema := range r.document.Components.Schemas {
		if schema.Name == "" || schema.GoName == "" {
			return fmt.Errorf("render OpenAPI design: component schema names must not be empty")
		}
		identifier := "Imported" + schema.GoName
		if !token.IsIdentifier(identifier) || token.Lookup(identifier).IsKeyword() {
			return fmt.Errorf("render OpenAPI design: component schema %q has invalid Go name %q", schema.Name, schema.GoName)
		}
		if _, exists := r.schemas[schema.Name]; exists {
			return fmt.Errorf("render OpenAPI design: component schema %q is defined more than once", schema.Name)
		}
		r.schemas[schema.Name] = schema
	}
	for _, operation := range r.operations {
		for _, response := range operation.failures {
			if len(response.headers) > 0 {
				continue
			}
			if err := r.markErrorSchema(response.response.Schema); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *renderer) markErrorSchema(schema *Schema) error {
	if schema == nil || schema.Ref == "" {
		return nil
	}
	name, err := localComponentReferenceName(schema.Ref, "#/components/schemas/")
	if err != nil {
		return fmt.Errorf("render OpenAPI design: schema reference %q %w", schema.Ref, err)
	}
	if _, marked := r.errorSchemas[name]; marked {
		return nil
	}
	named, ok := r.schemas[name]
	if !ok {
		return fmt.Errorf("render OpenAPI design: schema reference %q does not resolve", schema.Ref)
	}
	r.errorSchemas[name] = struct{}{}
	return r.markErrorSchema(named.Schema)
}
