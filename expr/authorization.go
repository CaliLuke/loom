package expr

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

type (
	// AuthorizationExpr declares an application-owned decision with a stable
	// name and a concrete named object input, or Empty for a context-only check.
	AuthorizationExpr struct {
		// Name identifies the requirement independently of service method names.
		Name string
		// Input defines the typed values passed to the application evaluator.
		Input *AttributeExpr
	}

	// MethodAuthorizationExpr describes either conjunctive requirements, an
	// explicit exemption, or exhaustive cases selected from a payload field.
	MethodAuthorizationExpr struct {
		// Requirements must all succeed before the operation can execute.
		Requirements []*AuthorizationUseExpr
		// Exemption explains why no additional authorization check is required.
		Exemption string
		// VariantSelection distinguishes a selector (including a root selector) from direct requirements.
		VariantSelection bool
		// Selector is a dot-separated payload path to a string enum or union.
		Selector string
		// Cases classify every declared selector value.
		Cases []*AuthorizationCaseExpr
	}

	// AuthorizationCaseExpr classifies one enum value or union branch name.
	AuthorizationCaseExpr struct {
		// Value is the authored enum value or union branch name.
		Value string
		// Authorization contains this case's requirements or exemption.
		Authorization *MethodAuthorizationExpr
	}

	// AuthorizationUseExpr binds a named requirement to a method payload.
	AuthorizationUseExpr struct {
		// Requirement is the registered application decision.
		Requirement *AuthorizationExpr
		// Bindings connect named input fields to typed payload paths.
		Bindings []*AuthorizationBindingExpr
	}

	// AuthorizationBindingExpr connects an input field to a payload path.
	AuthorizationBindingExpr struct {
		// Input is a top-level field of the requirement input.
		Input string
		// Payload is a dot-separated path, with union paths resolved in their case.
		Payload string
	}

	// AuthorizationPathStep describes one validated access along a payload path.
	AuthorizationPathStep struct {
		// Name is an object field or a selected union branch name.
		Name string
		// Parent owns the field and its requiredness.
		Parent *AttributeExpr
		// Attribute is the selected field or branch type.
		Attribute *AttributeExpr
		// Union indicates selection of the active union branch.
		Union bool
	}
)

// EvalName returns the requirement's diagnostic identity.
func (a *AuthorizationExpr) EvalName() string {
	return "authorization " + a.Name
}

// EvalName returns the classification's diagnostic identity.
func (*MethodAuthorizationExpr) EvalName() string {
	return "method authorization"
}

// EvalName returns the binding scope's diagnostic identity.
func (a *AuthorizationUseExpr) EvalName() string {
	return a.Requirement.EvalName() + " binding"
}

// AuthorizationPath resolves a payload path without reflection. Within a union
// case, traversing selector selects branch before continuing into its fields.
// Empty paths select the entire payload; this is useful for a root union.
func AuthorizationPath(payload *AttributeExpr, path, selector, branch string) ([]*AuthorizationPathStep, error) {
	var steps []*AuthorizationPathStep
	current := payload
	parts := strings.Split(path, ".")
	if path == "" {
		parts = nil
	}
	for i := 0; i <= len(parts); i++ {
		if current == nil || current.Type == nil || AllowsNull(current) {
			return nil, fmt.Errorf("authorization path %q has an unsupported nullable or missing type", path)
		}
		if union := AsUnion(current.Type); union != nil && branch != "" && strings.Join(parts[:i], ".") == selector {
			var selected *NamedAttributeExpr
			for _, value := range union.Values {
				if value.Name == branch {
					selected = value
					break
				}
			}
			if selected == nil {
				return nil, fmt.Errorf("authorization union branch %q does not exist", branch)
			}
			steps = append(steps, &AuthorizationPathStep{Name: selected.Name, Parent: current, Attribute: selected.Attribute, Union: true})
			current = selected.Attribute
		}
		if i == len(parts) {
			break
		}
		field := current.Find(parts[i])
		if field == nil || !IsObject(current.Type) {
			return nil, fmt.Errorf("authorization payload path %q: field %q does not exist", path, parts[i])
		}
		steps = append(steps, &AuthorizationPathStep{Name: parts[i], Parent: current, Attribute: field})
		current = field
	}
	return steps, nil
}

// AuthorizationValues returns the finite authored values of a selector.
func AuthorizationValues(att *AttributeExpr) ([]string, error) {
	if union := AsUnion(att.Type); union != nil {
		values := make([]string, len(union.Values))
		for i, branch := range union.Values {
			values[i] = branch.Name
		}
		return values, nil
	}
	underlying := att.Type
	for {
		ut, ok := underlying.(UserType)
		if !ok {
			break
		}
		underlying = ut.Attribute().Type
	}
	if underlying == String && att.Validation != nil && len(att.Validation.Values) > 0 {
		values := make([]string, len(att.Validation.Values))
		for i, value := range att.Validation.Values {
			str, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("authorization selector must be a string enum or union")
			}
			values[i] = str
		}
		return values, nil
	}
	if ut, ok := att.Type.(UserType); ok {
		return AuthorizationValues(ut.Attribute())
	}
	return nil, fmt.Errorf("authorization selector must be a string enum or union")
}

func (m *MethodExpr) validateAuthorization() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	a := m.Authorization
	if a == nil {
		if m.Service.StrictAuthorization || Root.API.StrictAuthorization {
			verr.Add(m, "missing authorization classification: use Authorize, AuthorizeBy, or NoAccessCheck")
		}
		return verr
	}
	m.validateAuthorizationClassification(a, "", "", verr)
	return verr
}

func (m *MethodExpr) validateAuthorizationClassification(a *MethodAuthorizationExpr, selector, branch string, verr *eval.ValidationErrors) {
	if a.Exemption != "" {
		if len(a.Requirements) > 0 || a.VariantSelection {
			verr.Add(m, "authorization exemption conflicts with requirements or cases")
		}
		return
	}
	if a.VariantSelection {
		if len(a.Requirements) > 0 || branch != "" {
			verr.Add(m, "authorization cases cannot be nested or combined with method requirements")
		}
		m.validateAuthorizationCases(a, verr)
		return
	}
	if len(a.Cases) > 0 {
		verr.Add(m, "authorization cases must appear directly in AuthorizeBy")
	}
	if len(a.Requirements) == 0 {
		verr.Add(m, "missing authorization classification")
	}
	for _, use := range a.Requirements {
		m.validateAuthorizationUse(use, selector, branch, verr)
	}
}

func (m *MethodExpr) validateAuthorizationCases(a *MethodAuthorizationExpr, verr *eval.ValidationErrors) {
	steps, err := AuthorizationPath(m.Payload, a.Selector, "", "")
	if err != nil {
		verr.Add(m, "%s", err)
		return
	}
	att := m.Payload
	if len(steps) > 0 {
		att = steps[len(steps)-1].Attribute
	}
	values, err := AuthorizationValues(att)
	if err != nil {
		verr.Add(m, "%s", err)
		return
	}
	seen := make(map[string]bool)
	for _, c := range a.Cases {
		if seen[c.Value] || !slices.Contains(values, c.Value) {
			verr.Add(m, "duplicate or unknown authorization case %q", c.Value)
		}
		seen[c.Value] = true
		m.validateAuthorizationClassification(c.Authorization, a.Selector, c.Value, verr)
	}
	for _, value := range values {
		if !seen[value] {
			verr.Add(m, "missing authorization case %q", value)
		}
	}
}

func (m *MethodExpr) validateAuthorizationUse(use *AuthorizationUseExpr, selector, branch string, verr *eval.ValidationErrors) {
	a := use.Requirement
	if a == nil || !slices.Contains(Root.Authorizations, a) || a.Input == nil {
		verr.Add(m, "unregistered authorization requirement")
		return
	}
	if a.Input.Type != Empty {
		if _, ok := a.Input.Type.(UserType); !ok || !IsObject(a.Input.Type) {
			verr.Add(m, "authorization %q input must be a named object or Empty", a.Name)
			return
		}
	}
	seen := make(map[string]bool)
	for _, binding := range use.Bindings {
		input := a.Input.Find(binding.Input)
		if input == nil || seen[binding.Input] {
			verr.Add(m, "unknown or duplicate authorization input binding %q", binding.Input)
			continue
		}
		seen[binding.Input] = true
		steps, err := AuthorizationPath(m.Payload, binding.Payload, selector, branch)
		if err != nil {
			verr.Add(m, "%s", err)
			continue
		}
		source := m.Payload
		if len(steps) > 0 {
			source = steps[len(steps)-1].Attribute
		}
		if AllowsNull(input) || IsAny(input.Type) || !authorizationAttributesEqual(input, source) {
			verr.Add(m, "authorization binding %q has incompatible or unsupported input type", binding.Input)
		}
	}
	if obj := AsObject(a.Input.Type); obj != nil {
		for _, field := range *obj {
			if !seen[field.Name] {
				verr.Add(m, "authorization input %q is missing a binding", field.Name)
			}
		}
	}
}

// Bindings assign values, so structural DSL equality alone is insufficient:
// nested named identities, field presence and Go metadata must agree as well.
func authorizationAttributesEqual(left, right *AttributeExpr) bool {
	if AllowsNull(left) != AllowsNull(right) {
		return false
	}
	for _, meta := range []MetaExpr{left.Meta, right.Meta} {
		for key := range meta {
			if strings.HasPrefix(key, "struct:") && !slices.Equal(left.Meta[key], right.Meta[key]) {
				return false
			}
		}
	}
	l, lu := left.Type.(UserType)
	r, ru := right.Type.(UserType)
	if lu || ru {
		return lu && ru && l.ID() == r.ID()
	}
	if left.Type.Kind() != right.Type.Kind() {
		return false
	}
	switch l := left.Type.(type) {
	case *Array:
		r := right.Type.(*Array)
		return authorizationAttributesEqual(l.ElemType, r.ElemType)
	case *Map:
		r := right.Type.(*Map)
		return authorizationAttributesEqual(l.KeyType, r.KeyType) && authorizationAttributesEqual(l.ElemType, r.ElemType)
	case *Object:
		r := right.Type.(*Object)
		if len(*l) != len(*r) {
			return false
		}
		for i, field := range *l {
			other := (*r)[i]
			if field.Name != other.Name || left.IsPrimitivePointer(field.Name, true) != right.IsPrimitivePointer(other.Name, true) || left.IsRequired(field.Name) != right.IsRequired(other.Name) || !authorizationAttributesEqual(field.Attribute, other.Attribute) {
				return false
			}
		}
		return true
	case *Union:
		// Anonymous sums have separately generated identities. Reuse a named
		// union when the complete sum is an authorization input.
		return left.Type == right.Type
	default:
		return Equal(left.Type, right.Type)
	}
}

func (a *AuthorizationExpr) validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if a.Input == nil || a.Input.Type == nil {
		verr.Add(a, "authorization input is missing")
		return verr
	}
	if a.Input.Type != Empty {
		if _, ok := a.Input.Type.(UserType); !ok || !IsObject(a.Input.Type) || AllowsNull(a.Input) {
			verr.Add(a, "authorization input must be a non-null named object or Empty")
		}
	}
	return verr
}
