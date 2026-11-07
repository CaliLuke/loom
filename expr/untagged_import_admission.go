package expr

import "github.com/CaliLuke/loom/eval"

// ValidateUntaggedUnion checks a resolved untagged union's supported branch
// grammar and codec contracts without using global DSL validation state. It
// accepts recursive objects but rejects cycles that consume no object field.
// Importers and other schema adapters can use it on detached expression graphs;
// callers retain responsibility for complete attribute validation and names.
func ValidateUntaggedUnion(union *Union) *eval.ValidationErrors {
	errors := new(eval.ValidationErrors)
	context := &AttributeExpr{Type: union}
	if union == nil || !union.Untagged {
		errors.Add(context, "expected an untagged OneOf")
		return errors
	}
	if hasInlineUnionCycle(union) {
		errors.Add(context, "untagged OneOf has an unconsumed branch cycle")
		return errors
	}
	errors.Merge(validateUntaggedBranches(union, "", context))
	return errors
}
