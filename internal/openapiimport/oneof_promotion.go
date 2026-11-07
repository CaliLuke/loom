package openapiimport

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/openapiref"
)

// prepareUnionFields establishes concrete names throughout a branch graph,
// including declarations reached through references. The same graph visitation
// handles recursive named records without relying on declaration order.
func (a *analyzer) prepareUnionFields(document *Document, schema *Schema, name string, used map[string]struct{}) {
	if schema == nil {
		return
	}
	if a.unionPrepared == nil {
		a.unionPrepared = make(map[*Schema]bool)
	}
	if a.unionPrepared[schema] {
		return
	}
	a.unionPrepared[schema] = true
	if schema.Ref != "" {
		component, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		if err == nil {
			a.prepareUnionFields(document, a.schemas[component], component, used)
		}
		return
	}
	for index := range schema.Properties {
		property := &schema.Properties[index]
		childName := name + codegen.Goify(property.Name, true)
		a.prepareUnionFields(document, property.Schema, childName, used)
		property.Schema = a.prepareSchemaObjects(document, property.Schema, childName, "#/components/schemas/"+escapeJSONPointer(name)+"/properties/"+escapeJSONPointer(property.Name), objectUnionBranch, used)
	}
	for _, branch := range schema.OneOf {
		a.prepareUnionFields(document, branch, name+"Variant", used)
	}
	for _, base := range schema.Bases {
		a.prepareUnionFields(document, base, name, used)
	}
	a.prepareUnionFields(document, schema.Items, name+"Item", used)
	if schema.AdditionalProperties != nil {
		a.prepareUnionFields(document, schema.AdditionalProperties.Schema, name+"AdditionalProperty", used)
	}
}
