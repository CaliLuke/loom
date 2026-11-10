package codegen

import "github.com/CaliLuke/loom/expr"

type (
	// TransformFields supplies mapped field selections for a stable object during
	// conversion analysis. Mutable mapped attributes also implement this contract;
	// callers must keep them stable for the duration of a transform.
	TransformFields interface {
		expr.FieldFactsProvider
		// Find returns the selected child for a logical field name, or nil.
		Find(name string) *expr.AttributeExpr
		// GetDefault returns the selected child's effective default, or nil.
		GetDefault(name string) any
		// IsRequired reports the occurrence's normalized field requirement.
		IsRequired(name string) bool
	}

	// transformFields borrows children from a stable transform-owned graph while
	// keeping scalar selection and layout facts in an immutable immediate snapshot.
	transformFields struct {
		*expr.ObjectFields
		attribute *expr.AttributeExpr
	}
)

// transformMappedAttribute captures immediate fields without cloning children.
// Discovery and emission must see the same selected child occurrences.
func transformMappedAttribute(attribute *expr.AttributeExpr) *transformFields {
	return &transformFields{ObjectFields: expr.NewObjectFields(attribute), attribute: attribute}
}

// Find returns the original child selected by the captured mapped field facts.
func (fields *transformFields) Find(name string) *expr.AttributeExpr {
	return fields.attribute.PhysicalMemberAttribute(fields.FieldFacts(name))
}

// GetDefault resolves the selected original child's inherited default.
func (fields *transformFields) GetDefault(name string) any {
	return fields.Find(name).EffectiveDefault()
}

// IsRequired reads occurrence requiredness, including dangling required names.
func (fields *transformFields) IsRequired(name string) bool {
	return fields.FieldFacts(name).Required
}
