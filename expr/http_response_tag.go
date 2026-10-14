package expr

import "github.com/CaliLuke/loom/eval"

// HTTPResponseTag binds an authored response selector to its semantic result field.
// Generated representations use Key to locate their own physical field; Attribute
// retains the validated declaration rather than an inferred Go field name.
type HTTPResponseTag struct {
	// Key is the result object's complete attribute key, including any wire suffix.
	Key string
	// Attribute is the declared String-kind result field.
	Attribute *AttributeExpr
	// Value is the authored string value selecting this response.
	Value string
}

func (r *HTTPResponseExpr) validateTag(result *AttributeExpr, errors *eval.ValidationErrors) {
	r.TagBinding = nil
	if r.Tag[0] == "" {
		return
	}
	key, attribute := result.FindAttribute(r.Tag[0])
	if attribute == nil || unalias(attribute.Type) != String {
		errors.Add(r, "Tag %q must identify an existing string result field", r.Tag[0])
		return
	}
	r.TagBinding = &HTTPResponseTag{Key: key, Attribute: attribute, Value: r.Tag[1]}
}
