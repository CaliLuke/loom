package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
)

// unionClosureMeta owns the authored spelling of occurrence closure.
const unionClosureMeta = "openapi:unevaluatedProperties"

// CaptureUnionClosure retains a constructor attribute's closure on a detached
// raw union. Constructors discard their temporary attribute, so they must check
// its policy before returning the type. Named references are finalized later.
func (a *AttributeExpr) CaptureUnionClosure() (*Union, error) {
	union, ok := a.Type.(*Union)
	if !ok {
		return nil, fmt.Errorf("union constructor closure requires a raw union")
	}
	if errors := a.validateUnionClosure("", a); len(errors.Errors) > 0 {
		return nil, errors
	}
	copy := *union
	copy.closed = a.UnionClosed()
	return &copy, nil
}

// UnionClosed reports whether this union occurrence rejects members that its
// selected alternative does not evaluate. The policy follows named aliases;
// adding it to a reference never changes the referenced declaration.
func (a *AttributeExpr) UnionClosed() bool {
	seen := make(map[*AttributeExpr]bool)
	for a != nil && !seen[a] {
		seen[a] = true
		if value, present := a.Meta.Last(unionClosureMeta); present && value == "false" {
			return true
		}
		named, ok := a.Type.(UserType)
		if !ok {
			union, ok := a.Type.(*Union)
			return ok && union.Closed()
		}
		a = named.Attribute()
	}
	return false
}

func (a *AttributeExpr) validateUnionClosure(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	values, present := a.Meta[unionClosureMeta]
	if !present {
		return verr
	}
	if AsUnion(a.Type) == nil {
		verr.Add(parent, "%s%s requires a union", ctx, unionClosureMeta)
	}
	valid := len(values) > 0
	for _, value := range values {
		valid = valid && value == "false"
	}
	if !valid {
		verr.Add(parent, "%s%s supports only false", ctx, unionClosureMeta)
	}
	return verr
}
