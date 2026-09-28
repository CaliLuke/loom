package dsl

import (
	"slices"

	"github.com/CaliLuke/loom/expr"
)

// copyAttributeKeepingType copies occurrence metadata without traversing the
// referenced types, whose definitions may still be incomplete. The copy owns
// its metadata entries, validation record, required names and documentation.
// Types, example values and defaults retain their existing shared ownership.
func copyAttributeKeepingType(attribute *expr.AttributeExpr) *expr.AttributeExpr {
	cloned := *attribute
	cloned.Bases = slices.Clone(attribute.Bases)
	cloned.References = slices.Clone(attribute.References)
	if attribute.Meta != nil {
		cloned.Meta = make(expr.MetaExpr, len(attribute.Meta))
		for key, values := range attribute.Meta {
			cloned.Meta[key] = slices.Clone(values)
		}
	}
	if attribute.Validation != nil {
		cloned.Validation = attribute.Validation.Dup()
	}
	if attribute.Docs != nil {
		docs := *attribute.Docs
		cloned.Docs = &docs
	}
	return &cloned
}
