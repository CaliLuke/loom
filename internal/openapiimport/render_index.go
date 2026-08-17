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
	errorNames := make(map[string]map[string]struct{})
	for operationIndex := range r.operations {
		for responseIndex := range r.operations[operationIndex].failures {
			response := &r.operations[operationIndex].failures[responseIndex]
			if len(response.headers) > 0 {
				continue
			}
			if err := r.markErrorSchema(response.response.Schema); err != nil {
				return err
			}
			if response.response.Schema == nil || response.response.Schema.Ref == "" {
				continue
			}
			name, err := localComponentReferenceName(response.response.Schema.Ref, "#/components/schemas/")
			if err != nil {
				return fmt.Errorf("render OpenAPI design: schema reference %q %w", response.response.Schema.Ref, err)
			}
			if errorNames[name] == nil {
				errorNames[name] = make(map[string]struct{})
			}
			errorNames[name][response.errorName] = struct{}{}
		}
	}
	for operationIndex := range r.operations {
		for responseIndex := range r.operations[operationIndex].failures {
			response := &r.operations[operationIndex].failures[responseIndex]
			if len(response.headers) > 0 || response.response.Schema == nil || response.response.Schema.Ref == "" {
				continue
			}
			name, err := localComponentReferenceName(response.response.Schema.Ref, "#/components/schemas/")
			if err != nil {
				return fmt.Errorf("render OpenAPI design: schema reference %q %w", response.response.Schema.Ref, err)
			}
			object, err := r.errorSchemaRendersAsObject(name, make(map[string]struct{}))
			if err != nil {
				return err
			}
			response.cloneErrorType = len(errorNames[name]) > 1 && object
		}
	}
	return nil
}

func (r *renderer) errorSchemaRendersAsObject(name string, visited map[string]struct{}) (bool, error) {
	if _, seen := visited[name]; seen {
		return false, fmt.Errorf("render OpenAPI design: component schema %q contains a reference cycle", name)
	}
	visited[name] = struct{}{}
	named, ok := r.schemas[name]
	if !ok {
		return false, fmt.Errorf("render OpenAPI design: component schema %q does not resolve", name)
	}
	if named.Schema == nil {
		return false, fmt.Errorf("render OpenAPI design: component schema %q is nil", name)
	}
	if named.Schema.Ref == "" {
		_, object, err := r.schemaExpression(named.Schema, "#/components/schemas/"+name)
		return object, err
	}
	referenced, err := localComponentReferenceName(named.Schema.Ref, "#/components/schemas/")
	if err != nil {
		return false, fmt.Errorf("render OpenAPI design: schema reference %q %w", named.Schema.Ref, err)
	}
	return r.errorSchemaRendersAsObject(referenced, visited)
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
