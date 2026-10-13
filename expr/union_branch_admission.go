package expr

import "github.com/CaliLuke/loom/eval"

// validateUnionBranchIdentities admits logical names and effective tags within
// one union after preparation has resolved its named branches. These are public
// branch identities, distinct from generated Go names and protobuf field tags.
func validateUnionBranchIdentities(union *Union, ctx string, parent eval.Expression) *eval.ValidationErrors {
	errors := new(eval.ValidationErrors)
	names := make(map[string]string, len(union.Values))
	tags := make(map[string]string, len(union.Values))
	for _, branch := range union.Values {
		name := AttributeName(branch.Name)
		if previous, exists := names[name]; exists {
			errors.Add(parent, "%sOneOf branch %q repeats logical name %q already used by branch %q", ctx, branch.Name, name, previous)
		} else {
			names[name] = branch.Name
		}
		tag := UnionVariantTag(branch)
		if previous, exists := tags[tag]; exists {
			errors.Add(parent, "%sOneOf branch %q repeats discriminator tag %q already used by branch %q", ctx, branch.Name, tag, previous)
		} else {
			tags[tag] = branch.Name
		}
	}
	return errors
}
