package openapiimport

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/openapiref"
)

type objectFieldBinding struct {
	property  NamedProperty
	name      string
	inherited bool
	emit      bool
}

func allocateObjectFields(fields []objectFieldBinding, errorType bool) {
	scope := codegen.NewNameScope()
	reserved := make(map[string]bool)
	for _, field := range fields {
		for _, name := range []string{codegen.Goify(field.property.Name, true), field.name} {
			if !reserved[name] {
				scope.Unique(name)
				reserved[name] = true
			}
		}
	}
	claimed := make(map[string]bool)
	for index := range fields {
		field := &fields[index]
		original := field.name
		_, method := generatedErrorMethodNames[original]
		if errorType && method {
			field.name = scope.Unique(original + "Field")
		} else if claimed[original] {
			field.name = scope.Unique(original)
		}
		claimed[field.name] = true
		if field.inherited && field.name != original {
			field.emit = true
		}
	}
}

// objectFields keeps wire keys separate from the effective object's Go fields.
// Base declarations retain their own allocation; derived-only overrides never
// mutate a referenced schema shared by other objects.
func (r *renderer) objectFields(schema *Schema, path string, errorType bool, active map[string]bool) ([]objectFieldBinding, error) {
	var fields []objectFieldBinding
	positions := make(map[string]int)
	bases := schema.Bases
	if schema.Ref != "" {
		bases = append([]*Schema{schema}, bases...)
	}
	for index, base := range bases {
		name, err := openapiref.Name(base.Ref, "#/components/schemas/")
		if err != nil {
			return nil, fmt.Errorf("render OpenAPI design: %s/bases/%d: %w", path, index, err)
		}
		if active[name] {
			return nil, fmt.Errorf("render OpenAPI design: %s: cyclic object base %q", path, name)
		}
		named, ok := r.schemas[name]
		if !ok || named.Schema == nil {
			return nil, fmt.Errorf("render OpenAPI design: %s: object base %q does not resolve", path, name)
		}
		active[name] = true
		_, baseError := r.errorSchemas[name]
		inherited, err := r.objectFields(named.Schema, path+"/bases/"+name, baseError, active)
		delete(active, name)
		if err != nil {
			return nil, err
		}
		for _, field := range inherited {
			field.inherited, field.emit = true, false
			if position, exists := positions[field.property.Name]; exists {
				fields[position] = field
			} else {
				positions[field.property.Name] = len(fields)
				fields = append(fields, field)
			}
		}
	}
	for _, property := range schema.Properties {
		field := objectFieldBinding{property: property, name: codegen.Goify(property.Name, true), emit: true}
		if position, exists := positions[property.Name]; exists {
			field.name = fields[position].name
			fields[position] = field
		} else {
			positions[property.Name] = len(fields)
			fields = append(fields, field)
		}
	}
	allocateObjectFields(fields, errorType)
	return fields, nil
}

// objectFieldAttributes lowers the shared identity plan. An inherited override
// contributes metadata only, preserving its base type, constraints and existing
// schema-reference initialization edges.
func (r *renderer) objectFieldAttributes(schema *Schema, path string, errorType bool) error {
	fields, err := r.objectFields(schema, path, errorType, make(map[string]bool))
	if err != nil {
		return err
	}
	for _, field := range fields {
		if !field.emit {
			continue
		}
		property := field.property
		if field.inherited {
			r.open("Attribute(%q, func()", property.Name)
			r.line("Meta(%q, %q)", "struct:field:name", field.name)
			r.close()
			continue
		}
		var metadata []renderedMetadata
		if field.name != codegen.Goify(property.Name, true) {
			metadata = append(metadata, renderedMetadata{name: "struct:field:name", value: field.name})
		}
		if err := r.attribute(property.Name, property.Schema, nil, path+"/properties/"+escapeJSONPointer(property.Name), metadata...); err != nil {
			return err
		}
	}
	return nil
}
