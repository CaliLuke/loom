package ir

import "github.com/CaliLuke/loom/expr"

// applyEncodingSourceType preserves the source type used by named content
// encoding without widening or narrowing the normalized JSON value contract.
// A missing source type must remain absent through ref/allOf inspection. When
// normalization produced a concrete singleton, oneOf retains its assertion in
// a child instead of deleting it. Concrete source types on normalized unions
// are redundant assertions, applied before the occurrence's null wrapper.
func (a *Analyzer) applyEncodingSourceType(schema *Schema, attribute *expr.AttributeExpr) *Schema {
	source, found := attribute.Meta.Last("http:encoding:sourceType")
	if !found {
		return schema
	}
	if source != "absent" {
		if schema.Type == "" || schema.Type == source {
			schema.Type = source
			if expr.AllowsNull(attribute) && !expr.IsNullable(attribute) {
				a.applyNullableSchema(schema)
			}
			return schema
		}
		wrapped := takeSchemaAnnotations(schema)
		wrapped.Type = source
		wrapped.AllOf = []*Schema{schema}
		setAcquisitionEntry(a, a.constructions, wrapped, schemaConstruction{slots: []schemaSlot{{location: schemaAllOf}}})
		if expr.AllowsNull(attribute) && !expr.IsNullable(attribute) {
			a.applyNullableSchema(wrapped)
		}
		return wrapped
	}
	if schema.Type == "" && schema.Ref == "" && len(schema.AllOf) == 0 {
		return schema
	}
	wrapped := takeSchemaAnnotations(schema)
	wrapped.OneOf = []*Schema{schema}
	setAcquisitionEntry(a, a.constructions, wrapped, schemaConstruction{slots: []schemaSlot{{location: schemaOneOf}}})
	return wrapped
}

// Source representation belongs to the occurrence that authored it. A marker
// on an alias must wrap its reference without changing the named declaration.
func hasEncodingSourceOverlay(attribute *expr.AttributeExpr, named expr.UserType) bool {
	source, found := attribute.Meta.Last("http:encoding:sourceType")
	declared, declaredFound := named.Attribute().Meta.Last("http:encoding:sourceType")
	return found && (!declaredFound || source != declared)
}
