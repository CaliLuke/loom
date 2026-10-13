package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

type comparisonSchemaResolver struct {
	registry map[string]*openapiimport.Schema
	authored map[string]bool
	expanded map[string]bool
}

func newComparisonSchemaResolver(expected, actual *openapiimport.Document) *comparisonSchemaResolver {
	r := &comparisonSchemaResolver{
		registry: make(map[string]*openapiimport.Schema),
		authored: make(map[string]bool),
		expanded: make(map[string]bool),
	}
	for _, component := range actual.Components.Schemas {
		r.registry[comparisonSchemaReference(component.Name)] = component.Schema
	}
	for _, component := range expected.Components.Schemas {
		r.authored[comparisonSchemaReference(component.Name)] = true
	}
	return r
}

func (r *comparisonSchemaResolver) inlineResponse(expected, actual *openapiimport.Response) error {
	if expected.Schema == nil || expected.Schema.Ref != "" || actual.Schema == nil || actual.Schema.Ref == "" {
		return nil
	}
	schema, err := r.resolve(actual.Schema, make(map[string]bool))
	if err != nil {
		return err
	}
	actual.Schema = schema
	removeDuplicatedResponseSchemaExamples(expected, actual)
	return nil
}

func (r *comparisonSchemaResolver) resolve(occurrence *openapiimport.Schema, active map[string]bool) (*openapiimport.Schema, error) {
	if occurrence.Ref == "" {
		return cloneOpenAPIContractValue(occurrence)
	}
	ref := occurrence.Ref
	target, ok := r.registry[ref]
	if !ok || target == nil {
		return nil, fmt.Errorf("comparison cannot resolve schema reference %q", ref)
	}
	if active[ref] {
		return nil, fmt.Errorf("comparison cannot inline cyclic schema reference %q", ref)
	}
	active[ref] = true
	base, err := r.resolve(target, active)
	delete(active, ref)
	if err != nil {
		return nil, err
	}
	local, err := cloneOpenAPIContractValue(occurrence)
	if err != nil {
		return nil, err
	}
	local.Ref = ""
	// The typed importer has already admitted this schema. Combine only disjoint
	// or identical fields; differing overlapping predicates/annotations need an
	// equivalence proof this comparison deliberately does not attempt. Reflection
	// ensures a new model field is retained or rejected, never silently forgotten.
	destination, overlay := reflect.ValueOf(base).Elem(), reflect.ValueOf(local).Elem()
	for index := 0; index < overlay.NumField(); index++ {
		field, existing := overlay.Field(index), destination.Field(index)
		if comparisonFieldEmpty(field) {
			continue
		}
		name := overlay.Type().Field(index).Name
		if !field.CanInterface() || !existing.CanSet() {
			return nil, fmt.Errorf("comparison cannot normalize schema field %s at %q", name, ref)
		}
		if !comparisonFieldEmpty(existing) && !reflect.DeepEqual(existing.Interface(), field.Interface()) {
			return nil, fmt.Errorf("comparison cannot normalize conflicting %s at %q", name, ref)
		}
		existing.Set(field)
	}
	r.expanded[ref] = true
	return base, nil
}

func (r *comparisonSchemaResolver) prune(document *openapiimport.Document) {
	// First remove only expansion candidates; remaining document roots include
	// authored components and unexpanded generated components. Compute reachable
	// targets against the original immutable registry before final removal.
	original := document.Components.Schemas
	document.Components.Schemas = nil
	for _, component := range original {
		ref := comparisonSchemaReference(component.Name)
		if !r.expanded[ref] || r.authored[ref] {
			document.Components.Schemas = append(document.Components.Schemas, component)
		}
	}
	live := make(map[string]bool)
	var retain func(*openapiimport.Schema)
	retain = func(schema *openapiimport.Schema) {
		if schema.Ref == "" || live[schema.Ref] {
			return
		}
		live[schema.Ref] = true
		if target := r.registry[schema.Ref]; target != nil {
			walkComparisonSchemas(reflect.ValueOf(target), retain)
		}
	}
	walkComparisonSchemas(reflect.ValueOf(document), retain)
	document.Components.Schemas = nil
	for _, component := range original {
		ref := comparisonSchemaReference(component.Name)
		if !r.expanded[ref] || r.authored[ref] || live[ref] {
			document.Components.Schemas = append(document.Components.Schemas, component)
		}
	}
}

// walkComparisonSchemas follows only typed model fields. Interface values are
// literals (examples/defaults/extensions), never schema references. Each graph
// walk has its own visited set; reference reachability is owned by the caller.
func walkComparisonSchemas(value reflect.Value, visit func(*openapiimport.Schema)) {
	seen := make(map[*openapiimport.Schema]bool)
	var walk func(reflect.Value)
	walk = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		if value.Type() == reflect.TypeFor[*openapiimport.Schema]() && !value.IsNil() {
			schema := value.Interface().(*openapiimport.Schema)
			if seen[schema] {
				return
			}
			seen[schema] = true
			visit(schema)
		}
		switch value.Kind() {
		case reflect.Pointer:
			if !value.IsNil() {
				walk(value.Elem())
			}
		case reflect.Struct:
			for index := 0; index < value.NumField(); index++ {
				if value.Type().Field(index).IsExported() {
					walk(value.Field(index))
				}
			}
		case reflect.Slice, reflect.Array:
			for index := 0; index < value.Len(); index++ {
				walk(value.Index(index))
			}
		}
	}
	walk(value)
}

func comparisonFieldEmpty(value reflect.Value) bool {
	return value.IsZero() || (value.Kind() == reflect.Slice || value.Kind() == reflect.Map) && value.Len() == 0
}

func comparisonSchemaReference(name string) string {
	return "#/components/schemas/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
}
