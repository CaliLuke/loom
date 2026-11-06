package openapiimport

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/openapiref"
)

type objectFieldBinding struct {
	property  NamedProperty
	name      string
	inherited bool
	emit      bool
}

// allocateObjectFields assigns distinct Go field names. members are Go names
// the object generates for other members, which declared fields must avoid.
func allocateObjectFields(fields []objectFieldBinding, errorType bool, members []string) {
	scope := codegen.NewNameScope()
	reserved := make(map[string]bool)
	generated := make(map[string]bool, len(members))
	for _, name := range members {
		scope.Unique(name)
		reserved[name] = true
		generated[name] = true
	}
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
		if errorType && method || generated[original] {
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
	members, err := r.generatedMemberNames(schema, positions, path)
	if err != nil {
		return nil, err
	}
	allocateObjectFields(fields, errorType, members)
	return fields, nil
}

// generatedMemberNames returns the Go field names a typed object generates
// besides its declared fields: the carrier of undeclared members and a field
// for each required member that no composed field declares. Declared fields
// can be renamed around them, but these names are fixed, so two of them that
// coincide are reported.
func (r *renderer) generatedMemberNames(schema *Schema, declared map[string]int, path string) ([]string, error) {
	if !r.objectAdmitsTypedMembers(schema) {
		return nil, nil
	}
	names := []string{expr.AdditionalAttributesFieldName}
	owners := map[string]string{expr.AdditionalAttributesFieldName: "the additional properties carrier"}
	for _, name := range schema.Required {
		if _, exists := declared[name]; exists {
			continue
		}
		goName := codegen.Goify(name, true)
		if owner, taken := owners[goName]; taken {
			return nil, fmt.Errorf("render OpenAPI design: %s required undeclared property %q generates Go field %q, which %s also generates", path, name, goName, owner)
		}
		owners[goName] = fmt.Sprintf("required undeclared property %q", name)
		names = append(names, goName)
	}
	return names, nil
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
