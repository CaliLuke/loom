package expr

import "github.com/CaliLuke/loom/eval"

// ObjectOpenness is the authored policy for object members that the object does
// not declare. It is read from the attribute's own metadata and does not follow
// user type references; each consumer queries the attribute it already owns.
type ObjectOpenness int

const (
	// ObjectOpennessUnset leaves the object open. API-level closed-object
	// OpenAPI emission may still document it as closed.
	ObjectOpennessUnset ObjectOpenness = iota
	// ObjectOpen explicitly admits undeclared members, including under API-level
	// closed-object emission.
	ObjectOpen
	// ObjectClosed rejects undeclared members.
	ObjectClosed
	// ObjectTyped admits undeclared members whose values satisfy the object's
	// AdditionalAttributes, and lets Required name members it does not
	// declare.
	ObjectTyped
)

// objectOpennessMeta is the authored DSL spelling of the policy. Only this
// owner, the DSL documentation and the OpenAPI importer renderer may name it.
const objectOpennessMeta = "openapi:additionalProperties"

// AdmitsUndeclared reports whether the policy accepts undeclared members.
func (o ObjectOpenness) AdmitsUndeclared() bool {
	return o != ObjectClosed
}

// ObjectOpenness returns the authored openness policy of the attribute.
func (a *AttributeExpr) ObjectOpenness() ObjectOpenness {
	if a == nil {
		return ObjectOpennessUnset
	}
	if a.AdditionalAttributes != nil {
		return ObjectTyped
	}
	value, ok := a.Meta.Last(objectOpennessMeta)
	switch {
	case !ok:
		return ObjectOpennessUnset
	case value == "false":
		return ObjectClosed
	default:
		return ObjectOpen
	}
}

// effectiveObjectOpenness returns the policy governing the attribute's value
// through the chain of named types it references: closed when any link is
// closed, as the conjunction of the chain, otherwise the nearest authored
// policy. It does not depend on validation having visited every alias.
func (a *AttributeExpr) effectiveObjectOpenness() ObjectOpenness {
	effective := ObjectOpennessUnset
	seen := make(map[*AttributeExpr]bool)
	for a != nil && !seen[a] {
		seen[a] = true
		switch policy := a.ObjectOpenness(); {
		case policy == ObjectClosed:
			return ObjectClosed
		case effective == ObjectOpennessUnset:
			effective = policy
		}
		named, ok := a.Type.(UserType)
		if !ok {
			break
		}
		a = named.Attribute()
	}
	return effective
}

// effectiveAdditionalAttributes returns the typed policy governing the
// attribute's value through the chain of named types it references, or nil
// when the effective policy is not typed.
func (a *AttributeExpr) effectiveAdditionalAttributes() *AttributeExpr {
	if a.effectiveObjectOpenness() != ObjectTyped {
		return nil
	}
	seen := make(map[*AttributeExpr]bool)
	for a != nil && !seen[a] {
		seen[a] = true
		if a.AdditionalAttributes != nil {
			return a.AdditionalAttributes
		}
		named, ok := a.Type.(UserType)
		if !ok {
			break
		}
		a = named.Attribute()
	}
	return nil
}

// undeclaredMemberPolicies returns the typed policies of the bases that do
// not declare the member name: each constrains that member's value.
func undeclaredMemberPolicies(bases []*AttributeExpr, name string) []*AttributeExpr {
	var policies []*AttributeExpr
	for _, base := range bases {
		if object := AsObject(base.Type); object != nil && object.Attribute(name) != nil {
			continue
		}
		if additional := base.effectiveAdditionalAttributes(); additional != nil && !admitsAnyValue(additional) {
			policies = append(policies, additional)
		}
	}
	return policies
}

// CloseObject sets the attribute's openness policy to ObjectClosed.
func (a *AttributeExpr) CloseObject() {
	if a.Meta == nil {
		a.Meta = make(MetaExpr)
	}
	a.Meta[objectOpennessMeta] = []string{"false"}
}

// IsObjectOpennessMeta reports whether key stores the authored openness
// policy. A named-type occurrence's policy selects the representation of the
// value, so representations compare it as shape rather than as an annotation.
func IsObjectOpennessMeta(key string) bool {
	return key == objectOpennessMeta
}

// CopyObjectOpenness gives target the authored policy of source, as when a
// transport body is derived from a payload or result user type, or when a
// representation applies a named-type occurrence's policy to its own copy of
// the type. It does nothing when source authors no policy. A typed policy
// that target already carries, such as a transport body's renamed copy, is
// kept.
func CopyObjectOpenness(target *AttributeExpr, source *AttributeExpr) {
	if source.AdditionalAttributes != nil && target.AdditionalAttributes == nil {
		target.AdditionalAttributes = DupAtt(source.AdditionalAttributes)
	}
	values, ok := source.Meta[objectOpennessMeta]
	if !ok {
		return
	}
	if target.Meta == nil {
		target.Meta = make(MetaExpr)
	}
	target.Meta[objectOpennessMeta] = append([]string(nil), values...)
}

// reopensClosedObject reports whether the attribute authors any policy other
// than closed, or removes the policy, over an inherited closed object. A key
// without a value counts, because composition closes the object before
// validation could report it.
func (a *AttributeExpr) reopensClosedObject() bool {
	if a.removedMeta[objectOpennessMeta] || a.AdditionalAttributes != nil {
		return true
	}
	values, ok := a.Meta[objectOpennessMeta]
	return ok && (len(values) == 0 || values[len(values)-1] != "false")
}

// validateObjectOpenness checks this attribute's authored policy. Named
// declaration metadata is checked separately by root declaration validation.
func (a *AttributeExpr) validateObjectOpenness(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if a.AdditionalAttributes != nil {
		switch {
		case AsObject(a.Type) == nil:
			verr.Add(parent, `%sAdditionalAttributes applies only to objects`, ctx)
		case a.Meta[objectOpennessMeta] != nil || a.removedMeta[objectOpennessMeta]:
			verr.Add(parent, `%sAdditionalAttributes cannot be combined with %s`, ctx, objectOpennessMeta)
		}
		return verr
	}
	values, ok := a.Meta[objectOpennessMeta]
	switch {
	case !ok:
	case len(values) == 0:
		verr.Add(parent, `%s%s must be "true" or "false", got no value`, ctx, objectOpennessMeta)
	case values[len(values)-1] != "true" && values[len(values)-1] != "false":
		verr.Add(parent, `%s%s must be "true" or "false", got %q`, ctx, objectOpennessMeta, values[len(values)-1])
	case AsObject(a.Type) == nil:
		verr.Add(parent, `%s%s applies only to objects`, ctx, objectOpennessMeta)
	case a.Type != nil && a.TypedObjectDeclaration() != nil:
		// A typed declaration fixes the physical layout of every occurrence.
		verr.Add(parent, `%s%s cannot override the AdditionalAttributes of %q`, ctx, objectOpennessMeta, a.Type.Name())
	case values[len(values)-1] == "true":
		if named, ok := a.Type.(UserType); ok && named.Attribute().effectiveObjectOpenness() == ObjectClosed {
			verr.Add(parent, `%scannot reopen closed type %q with %s "true"`, ctx, named.Name(), objectOpennessMeta)
		}
	}
	return verr
}
