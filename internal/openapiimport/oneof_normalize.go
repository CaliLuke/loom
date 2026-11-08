package openapiimport

import (
	"fmt"
	"slices"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

type nullMembership uint8

const (
	nullExcluded nullMembership = iota
	nullOccurrence
	nullDeclared
)

// schemaUntaggedOneOf normalizes each branch before counting null membership.
// It never flattens nested unions: exact-one is not associative.
func (a *analyzer) schemaUntaggedOneOf(schema *Schema, source *base.Schema, path string) bool {
	if len(source.OneOf) == 0 || len(source.AnyOf) > 0 || len(source.AllOf) > 0 {
		return false
	}
	branches := make([]*Schema, 0, len(source.OneOf))
	bare := 0
	for index, proxy := range source.OneOf {
		if isNullOnlySchema(proxy) {
			bare++
			continue
		}
		if proxy == nil {
			return false
		}
		branch := a.schema(proxy, fmt.Sprintf("%s/oneOf/%d", path, index))
		branches = append(branches, branch)
	}
	*schema = Schema{OneOf: branches}
	fail := func(code, at, message string) {
		schema.compositionError = &Diagnostic{Code: code, Path: at, Message: message}
	}
	if len(branches) == 0 {
		fail("schema-type", path, "null-only oneOf requires a null-only type (#1010)")
		return true
	}
	matches, selected := 0, 0
	for index, branch := range branches {
		if normalizedNullKind(branch) != nullExcluded {
			matches++
			selected = index
		}
	}
	if bare > 0 && matches == 1 {
		if normalizedNullKind(branches[selected]) == nullDeclared {
			fail("schema-type", path+"/oneOf", "oneOf cannot remove declared null membership; null-only types require #1010")
			return true
		}
		removeOccurrenceNull(branches[selected])
	}
	nullable := bare == 1 && matches == 0
	if len(branches) == 1 && source.Discriminator == nil {
		a.copyNormalizedSchema(schema, branches[0])
	}
	schema.Nullable = schema.Nullable || nullable
	if source.UnevaluatedProperties != nil && source.UnevaluatedProperties.IsB() && !source.UnevaluatedProperties.B {
		if len(schema.OneOf) == 0 {
			// Keep the occurrence closure separate from the selected branch's own
			// openness when singleton normalization would otherwise erase it.
			*schema = Schema{OneOf: branches, Nullable: nullable}
		}
		schema.UnionClosed = true
	}
	a.unionSiblings = append(a.unionSiblings, oneOfSiblingCheck{schema: schema, source: source, path: path, branches: branches})
	return true
}

func normalizedNullKind(schema *Schema) nullMembership {
	if schema == nil {
		return nullExcluded
	}
	if schema.Ref != "" {
		if schema.declaredNull {
			return nullDeclared
		}
		if schema.Nullable {
			return nullOccurrence
		}
		return nullExcluded
	}
	if schema.Unconstrained {
		return nullDeclared
	}
	if schema.Nullable {
		return nullOccurrence
	}
	count, kind := 0, nullExcluded
	for _, branch := range schema.OneOf {
		if candidate := normalizedNullKind(branch); candidate != nullExcluded {
			count++
			kind = candidate
		}
	}
	if count == 1 {
		return kind
	}
	return nullExcluded
}

func removeOccurrenceNull(schema *Schema) {
	schema.Nullable = false
	schema.TypeUnion = slices.DeleteFunc(slices.Clone(schema.TypeUnion), func(value string) bool {
		return value == "null"
	})
	if len(schema.TypeUnion) == 1 {
		schema.TypeUnion = nil
	}
	for _, branch := range schema.OneOf {
		if normalizedNullKind(branch) == nullOccurrence {
			removeOccurrenceNull(branch)
		}
	}
}

// sourceAdmitsNull classifies a reference without expanding it into the normalized
// occurrence graph. References retain declaration ownership even when their
// target spells nullability through a nested composition.
func sourceAdmitsNull(proxy *base.SchemaProxy, active map[*base.Schema]bool) bool {
	if proxy == nil {
		return false
	}
	source := proxy.Schema()
	if source == nil || active[source] {
		return false
	}
	active[source] = true
	defer delete(active, source)
	if len(source.Type) > 0 && !slices.Contains(source.Type, "null") && (source.Nullable == nil || !*source.Nullable) {
		return false
	}
	if hasSchemaEnum(source) {
		found := false
		for _, value := range source.Enum {
			if value != nil && value.Tag == "!!null" {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	if source.Const != nil && source.Const.Tag != "!!null" {
		return false
	}
	if len(source.OneOf) > 0 {
		count := 0
		for _, branch := range source.OneOf {
			if sourceAdmitsNull(branch, active) {
				count++
			}
		}
		return count == 1
	}
	if len(source.AnyOf) > 0 {
		for _, branch := range source.AnyOf {
			if sourceAdmitsNull(branch, active) {
				return true
			}
		}
		return false
	}
	for _, branch := range source.AllOf {
		if !sourceAdmitsNull(branch, active) {
			return false
		}
	}
	return true
}

// copyNormalizedSchema transfers pending source checks to the surviving value.
// Deferred discriminator hints share their carrier; sibling diagnostics retain
// the original source path while updating the schema that planning will visit.
func (a *analyzer) copyNormalizedSchema(target, source *Schema) {
	*target = *source
	for i := range a.unionSiblings {
		if a.unionSiblings[i].schema == source {
			a.unionSiblings[i].schema = target
		}
	}
}
