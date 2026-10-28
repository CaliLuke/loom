package openapiimport

import (
	"slices"
	"sort"

	"github.com/CaliLuke/libopenapi/datamodel/high/base"
	"github.com/CaliLuke/libopenapi/orderedmap"
)

// schemaObjectReference uses object composition for physical fields while
// retaining the reference as a separate contract contributor.
func (a *analyzer) schemaObjectReference(local, target *base.Schema, ref, path string) *Schema {
	fail := func(message string) *Schema {
		return &Schema{Ref: ref, referenceError: &Diagnostic{Code: "schema-reference-siblings", Path: path, Message: message}}
	}
	if !slices.Equal(target.Type, []string{"object"}) || len(local.Type) > 0 && !slices.Equal(local.Type, target.Type) {
		return fail("object reference properties require a nonnullable object target")
	}
	if local.Items != nil || len(local.AllOf) > 0 || len(local.AnyOf) > 0 || len(local.OneOf) > 0 {
		return fail("object reference properties cannot be combined with other structural applicators")
	}
	if message := referenceObjectAdmission(local, target); message != "" {
		return fail(message)
	}
	schema := a.schemaPropertyRefinement(local, target, path)
	schema.Bases = []*Schema{{Ref: ref}}
	schema.referenceSiblings = true
	return schema
}

// schemaPropertyRefinement supplies missing carrier context only inside an
// established intersection. Constraints still belong to their authored layer.
func (a *analyzer) schemaPropertyRefinement(local, target *base.Schema, path string) *Schema {
	fail := func(message string) *Schema {
		return &Schema{referenceError: &Diagnostic{Code: "schema-reference-siblings", Path: path, Message: message}}
	}
	if message := referenceObjectAdmission(local, target); message != "" {
		return fail(message)
	}
	source := *local
	types := a.referenceCarrierTypes(target, path)
	if len(local.Type) > 0 && !sameReferenceTypes(local.Type, types) && !sameReferenceTypes(local.Type, target.Type) {
		return fail("property refinement cannot change the inherited type or nullability")
	}
	source.Type = types
	if local.Format != "" && local.Format != target.Format && !slices.Contains(types, "string") {
		return fail("property refinement cannot change the inherited Go carrier format")
	}
	if source.Format == "" {
		source.Format = target.Format
	}
	if source.Items == nil {
		source.Items = target.Items
	}
	source.Properties = nil
	schema := a.schemaSource(&source, path)
	if slices.Contains(types, "null") && !schema.Nullable {
		return fail("property membership cannot remove inherited nullability")
	}
	for name, property := range local.Properties.FromOldest() {
		propertyPath := path + "/properties/" + escapeJSONPointer(name)
		refined := a.schemaInheritedProperty(property, target.Properties.GetOrZero(name), propertyPath)
		schema.Properties = append(schema.Properties, NamedProperty{Name: name, Schema: refined})
	}
	sort.Slice(schema.Properties, func(i, j int) bool { return schema.Properties[i].Name < schema.Properties[j].Name })
	return schema
}

func (a *analyzer) schemaInheritedProperty(property, inherited *base.SchemaProxy, path string) *Schema {
	if inherited == nil || property == nil || property.IsReference() {
		return a.schema(property, path)
	}
	a.schemaUnknownKeywordDiagnostics(property, path)
	shape := referenceSchemaShape(inherited, make(map[*base.Schema]bool))
	if shape == nil || property.Schema() == nil {
		return a.schema(property, path)
	}
	if inherited.IsReference() && orderedmap.Len(property.Schema().Properties) > 0 {
		return a.schemaObjectReference(property.Schema(), shape, inherited.GetReference(), path)
	}
	refined := a.schemaPropertyRefinement(property.Schema(), shape, path)
	if inherited.IsReference() {
		refined.Ref = inherited.GetReference()
		refined.referenceSiblings = true
	}
	return refined
}

func referenceObjectAdmission(local, target *base.Schema) string {
	closed := target.AdditionalProperties != nil && target.AdditionalProperties.IsB() && !target.AdditionalProperties.B
	if orderedmap.Len(local.Properties) > 0 && orderedmap.Len(target.Properties) == 0 && !closed {
		return "structured properties cannot refine a free-form map"
	}
	if closed {
		for name := range local.Properties.FromOldest() {
			if target.Properties.GetOrZero(name) == nil {
				return "a reference to a closed object cannot add properties"
			}
		}
	}
	if local.AdditionalProperties == nil {
		return ""
	}
	if local.AdditionalProperties.IsA() {
		return "typed additionalProperties intersections require general schema composition support"
	}
	if !local.AdditionalProperties.B {
		for name := range target.Properties.FromOldest() {
			if local.Properties.GetOrZero(name) == nil {
				return "a closed sibling must declare every inherited property"
			}
		}
	}
	return ""
}
