package openapiimport

import (
	"slices"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func (a *analyzer) schemaReference(proxy *base.SchemaProxy, path string) *Schema {
	ref := proxy.GetReference()
	fail := func(message string) *Schema {
		return &Schema{Ref: ref, referenceError: &Diagnostic{Code: "schema-reference-siblings", Path: path, Message: message}}
	}
	if !proxy.IsTransformedRefWithSiblings() {
		return &Schema{Ref: ref, declaredNull: sourceAdmitsNull(proxy, make(map[*base.Schema]bool))}
	}
	if strings.HasPrefix(a.version, "3.0.") {
		return fail("schema reference siblings require OpenAPI 3.1 or 3.2")
	}
	sibling := proxy.Schema()
	semantic, err := proxy.BuildTransformedRefSemanticSchema(sibling)
	if err != nil || sibling == nil || semantic == nil || len(semantic.AllOf) != 2 {
		return fail("schema reference siblings could not be resolved")
	}
	target := referenceSchemaShape(semantic.AllOf[1], make(map[*base.Schema]bool))
	if target == nil {
		return fail("referenced schema has no supported concrete shape")
	}
	if orderedmap.Len(sibling.Properties) > 0 {
		return a.schemaObjectReference(sibling, target, ref, path)
	}
	if sibling.Items != nil || sibling.AdditionalProperties != nil ||
		len(sibling.AllOf) > 0 || len(sibling.AnyOf) > 0 || len(sibling.OneOf) > 0 {
		return fail("structural reference intersections require general schema composition support")
	}
	return a.schemaReferenceOverlay(sibling, target, ref, path)
}

// schemaReferenceOverlay normalizes occurrence assertions with the referenced carrier.
func (a *analyzer) schemaReferenceOverlay(sibling, target *base.Schema, ref, path string) *Schema {
	fail := func(message string) *Schema {
		return &Schema{Ref: ref, referenceError: &Diagnostic{Code: "schema-reference-siblings", Path: path, Message: message}}
	}
	targetTypes := a.referenceCarrierTypes(target, ref)
	if len(sibling.Type) > 0 && !sameReferenceTypes(sibling.Type, target.Type) && !sameReferenceTypes(sibling.Type, targetTypes) {
		return fail("reference sibling type must preserve the referenced concrete type and nullability")
	}
	if sibling.Format != "" && sibling.Format != target.Format && !slices.Contains(target.Type, "string") {
		return fail("reference siblings cannot change the referenced Go carrier format")
	}
	// Type and format provide the carrier context needed to normalize enums,
	// defaults and lengths. They do not replace or mutate the referenced schema.
	source := *sibling
	source.Type = targetTypes
	// A reference to an admitted union shares the same count-before-close
	// occurrence contract as an adjacent oneOf. Keep its closure local.
	unionClosed := len(target.OneOf) > 0 && source.UnevaluatedProperties != nil && source.UnevaluatedProperties.IsB() && !source.UnevaluatedProperties.B
	if unionClosed {
		source.UnevaluatedProperties = nil
	}
	inheritedFormat := source.Format == ""
	if inheritedFormat {
		source.Format = target.Format
	}
	schema := a.schemaSource(&source, path)
	if slices.Contains(targetTypes, "null") && !schema.Nullable {
		return fail("reference sibling membership cannot remove inherited nullability")
	}
	schema.Ref = ref
	schema.UnionClosed = unionClosed
	schema.referenceSiblings = true
	schema.inheritedFormat = inheritedFormat
	return schema
}

func referenceSchemaShape(proxy *base.SchemaProxy, seen map[*base.Schema]bool) *base.Schema {
	if proxy == nil {
		return nil
	}
	source := proxy.Schema()
	if source == nil || seen[source] {
		return nil
	}
	seen[source] = true
	if len(source.Type) > 0 || len(source.OneOf) > 0 && len(source.AnyOf) == 0 && len(source.AllOf) == 0 {
		return source
	}
	if len(source.AllOf) == 1 {
		return referenceSchemaShape(source.AllOf[0], seen)
	}
	return nil
}

func sameReferenceTypes(left, right []string) bool {
	left, right = slices.Clone(left), slices.Clone(right)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

// referenceCarrierTypes applies the same membership normalization as declarations
// before supplying context to an occurrence. A nullable spelling alone does not
// authorize null when enum or const has already excluded it from the base.
func (a *analyzer) referenceCarrierTypes(target *base.Schema, path string) []string {
	if valueType, nullable := nullableUnionType(target.Type); nullable {
		domain := &Schema{Type: valueType, Format: target.Format, Nullable: true}
		a.schemaValueDomain(domain, target, path)
		if !domain.Nullable {
			return []string{valueType}
		}
	}
	return slices.Clone(target.Type)
}
