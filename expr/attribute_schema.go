package expr

import "github.com/CaliLuke/loom/eval"

func (a *AttributeExpr) validateSchema(ctx string, parent eval.Expression) *eval.ValidationErrors {
	validator := attributeValidator{schemaOnly: true, seen: make(map[*AttributeExpr]bool)}
	return validator.validate(a, ctx, parent)
}

// finalizeSchema preserves authored documentation without runtime body wrappers.
// An explicit occurrence name still claims a component, including inline schemas.
func (a *AttributeExpr) finalizeSchema() {
	if name, ok := a.Meta.Last("openapi:typename"); ok && name != "" {
		a.AddMeta("openapi:typename:canonical", "true")
		if _, named := a.Type.(UserType); !named && a.Type != Empty {
			a.Type = &UserTypeExpr{TypeName: name, AttributeExpr: DupAtt(a)}
		}
	}
	a.Finalize()
}
