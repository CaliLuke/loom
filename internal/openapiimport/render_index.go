package openapiimport

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/openapiref"
)

// indexErrors completes declaration planning after operation names are final.
// Component identity comes from the admitted index shared with the renderer.
func (r *renderer) indexErrors() error {
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
			name, err := openapiref.Name(response.response.Schema.Ref, "#/components/schemas/")
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
			name, err := openapiref.Name(response.response.Schema.Ref, "#/components/schemas/")
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
	r.assignErrorCloneNames()
	return nil
}

func (r *renderer) assignErrorCloneNames() {
	typeNames := codegen.NewNameScope()
	used := make(map[string]int, len(r.schemas)+len(r.document.Components.SecuritySchemes))
	for _, schema := range r.document.Components.Schemas {
		typeNames.Unique(schema.GoName)
		uniqueName("Imported"+schema.GoName, used)
	}
	for _, scheme := range r.document.Components.SecuritySchemes {
		uniqueName(scheme.GoName, used)
	}
	for operationIndex := range r.operations {
		for responseIndex := range r.operations[operationIndex].failures {
			response := &r.operations[operationIndex].failures[responseIndex]
			if response.cloneErrorType {
				response.cloneErrorTypeName = typeNames.Unique(codegen.Goify(response.errorName, true))
				response.cloneErrorGoName = uniqueName("Imported"+response.cloneErrorTypeName+"Error", used)
			}
		}
	}
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
	referenced, err := openapiref.Name(named.Schema.Ref, "#/components/schemas/")
	if err != nil {
		return false, fmt.Errorf("render OpenAPI design: schema reference %q %w", named.Schema.Ref, err)
	}
	return r.errorSchemaRendersAsObject(referenced, visited)
}

func (r *renderer) markErrorSchema(schema *Schema) error {
	if schema == nil || schema.Ref == "" {
		return nil
	}
	name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
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
