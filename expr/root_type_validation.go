package expr

import "github.com/CaliLuke/loom/eval"

// validateTypeDeclarations checks declaration-owned properties even when no
// method references the type. Runtime admission stays occurrence-scoped.
func (r *RootExpr) validateTypeDeclarations() *eval.ValidationErrors {
	var verr eval.ValidationErrors
	for _, ut := range r.Types {
		verr.Merge(ut.Attribute().validateMetadata("", ut))
		verr.Merge(ut.Attribute().validateExamples("", ut))
		if union := AsUnion(ut); union != nil && hasInlineUnionCycle(union) {
			verr.Add(ut, "recursive OneOf branch cycle in %q is not supported; place the recursive reference inside an object field", union.Name())
		}
	}
	for _, rt := range r.ResultTypes {
		verr.Merge(rt.Attribute().validateMetadata("", rt))
		verr.Merge(rt.Attribute().validateExamples("", rt))
	}
	return &verr
}

// validateMetadata checks only metadata owned by this attribute. Declaration
// validation and occurrence validation share this entry point, without walking
// the referenced declaration or claiming runtime validation of its children.
func (a *AttributeExpr) validateMetadata(ctx string, parent eval.Expression) *eval.ValidationErrors {
	var verr eval.ValidationErrors
	verr.Merge(a.validatePresenceMetadata(ctx, parent))
	verr.Merge(a.validateStructTagKeys(ctx, parent))
	verr.Merge(validateStructMeta(ctx, a.Meta, parent))
	verr.Merge(a.validateObjectOpenness(ctx, parent))
	verr.Merge(a.validateUnionClosure(ctx, parent))
	if result, ok := parent.(*ResultTypeExpr); ok && result.Attribute() == a {
		// A result declaration selects its own view; a reference selects a
		// view on its referenced result type instead.
		if err := result.validateExplicitViewMeta(); err != nil {
			verr.Add(parent, "%s%s", ctx, err.Error())
		}
	} else {
		verr.Merge(a.validateViewReference(ctx, parent))
	}
	return &verr
}
