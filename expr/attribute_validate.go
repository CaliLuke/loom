package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
)


// Validate tests whether the attribute required fields exist.  Since attributes
// are unaware of their context, additional context information can be provided
// to be used in error messages.  The parent definition context is automatically
// added to error messages.
func (a *AttributeExpr) Validate(ctx string, parent eval.Expression) *eval.ValidationErrors {
	if validated[a] {
		return nil
	}
	validated[a] = true
	verr := new(eval.ValidationErrors)
	if err := a.resolveTypeRef(); err != nil {
		verr.Add(parent, "%s", err.Error())
		return verr
	}
	if a.Type == nil {
		verr.Add(parent, "attribute type is nil")
		return verr
	}
	if ctx != "" {
		ctx += " - "
	}
	verr.Merge(a.validateEnumDefault(ctx, parent))
	if v := a.Validation; v != nil {
		verr.Merge(v.Validate(ctx, parent))
	}
	verr.Merge(a.validateExamples(ctx, parent))
	verr.Merge(a.validateChildTypes(ctx, parent))
	verr.Merge(a.validateViewReference(ctx, parent))

	return verr
}

// Prepare resolves any deferred named type references before validation.
func (a *AttributeExpr) Prepare() {
	a.prepareTypeRefs(make(map[*AttributeExpr]struct{}))
}

func (a *AttributeExpr) validatePkgPath(pkgPath string, t DataType) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if ar := AsArray(t); ar != nil {
		verr.Merge(a.validatePkgPath(pkgPath, ar.ElemType.Type))
	}
	if mp := AsMap(t); mp != nil {
		verr.Merge(a.validatePkgPath(pkgPath, mp.KeyType.Type))
		verr.Merge(a.validatePkgPath(pkgPath, mp.ElemType.Type))
	}
	if u := AsUnion(t); u != nil {
		for _, nat := range u.Values {
			if nat == nil || nat.Attribute == nil {
				continue
			}
			verr.Merge(a.validatePkgPath(pkgPath, nat.Attribute.Type))
		}
	}
	if ut, ok := t.(UserType); pkgPath != "" && ok {
		// This check ensures we error if a sub-type has a different custom package type set
		// or if two user types have different custom packages but share a sub-type (field that's a user type)
		if ut.Attribute().Meta != nil &&
			ut.Attribute().Meta["struct:pkg:path"] != nil &&
			ut.Attribute().Meta["struct:pkg:path"][0] != pkgPath {
			verr.Add(a, "type \"%s\" has conflicting packages %s and %s", ut.Name(), ut.Attribute().Meta["struct:pkg:path"][0], pkgPath)
		}
		ut.Attribute().AddMeta("struct:pkg:path", pkgPath)
	}
	if len(verr.Errors) > 0 {
		return verr
	}
	return nil
}

func (a *AttributeExpr) validateChildTypes(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if o := AsObject(a.Type); o != nil {
		verr.Merge(a.validateObjectChildren(ctx, parent, o))
		return verr
	}
	if ar := AsArray(a.Type); ar != nil {
		verr.Merge(ar.ElemType.Validate(ctx, a))
		return verr
	}
	if u := AsUnion(a.Type); u != nil {
		for _, ut := range u.Values {
			verr.Merge(ut.Attribute.Validate(ctx, parent))
		}
	}
	return verr
}

func (a *AttributeExpr) validateObjectChildren(ctx string, parent eval.Expression, obj *Object) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, n := range a.AllRequired() {
		if a.Find(n) == nil {
			verr.Add(parent, `%srequired field %q does not exist in type %s`, ctx, n, a.Type.Name())
		}
	}
	pkgPath := attributePkgPath(a.Type)
	for _, nat := range *obj {
		verr.Merge(a.validatePkgPath(pkgPath, nat.Attribute.Type))
		fieldCtx := fmt.Sprintf("field %s", nat.Name)
		verr.Merge(nat.Attribute.Validate(fieldCtx, parent))
	}
	return verr
}

func attributePkgPath(dt DataType) string {
	ut, ok := dt.(UserType)
	if !ok {
		return ""
	}
	meta, ok := ut.Attribute().Meta["struct:pkg:path"]
	if !ok || len(meta) == 0 {
		return ""
	}
	return meta[0]
}

func (a *AttributeExpr) validateViewReference(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	view, ok := a.Meta.Last(ViewMetaKey)
	if !ok {
		return verr
	}
	rt, ok := a.Type.(*ResultTypeExpr)
	if !ok {
		verr.Add(parent, "%s uses view %q but %q is not a result type", ctx, view, a.Type.Name())
		return verr
	}
	if view == DefaultView || resultTypeHasView(rt, view) {
		return verr
	}
	verr.Add(parent, "%s: type %q does not define view %q", ctx, a.Type.Name(), view)
	return verr
}

func resultTypeHasView(rt *ResultTypeExpr, view string) bool {
	for _, candidate := range rt.Views {
		if candidate.Name == view {
			return true
		}
	}
	return false
}
