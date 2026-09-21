package dsl

import (
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// Authorization declares a stable application-owned access requirement. Input
// must be a named object Type or Empty for decisions based only on context.
// Declare requirements at the top level and apply them with Authorize.
func Authorization(name string, input expr.DataType) *expr.AuthorizationExpr {
	if _, ok := eval.Current().(eval.TopExpr); !ok {
		eval.IncompatibleDSL()
		return nil
	}
	if strings.TrimSpace(name) == "" || input == nil {
		eval.ReportError("authorization requires a name and input type")
		return nil
	}
	for _, a := range expr.Root.Authorizations {
		if a.Name == name {
			eval.ReportError("authorization %q already defined", name)
			return nil
		}
	}
	a := &expr.AuthorizationExpr{Name: name, Input: &expr.AttributeExpr{Type: input}}
	expr.Root.Authorizations = append(expr.Root.Authorizations, a)
	return a
}

// StrictAuthorization requires an explicit authorization classification on
// every method in an API or service. It does not alter credential security.
func StrictAuthorization() {
	switch current := eval.Current().(type) {
	case *expr.APIExpr:
		current.StrictAuthorization = true
	case *expr.ServiceExpr:
		current.StrictAuthorization = true
	default:
		eval.IncompatibleDSL()
	}
}

// Authorize applies a requirement in a Method or AuthorizationCase. Multiple
// requirements are conjunctive. The optional function contains Bind calls.
// An evaluator is mandatory even outside StrictAuthorization scopes.
func Authorize(requirement *expr.AuthorizationExpr, fn ...func()) {
	a := authorizationScope()
	if a == nil {
		return
	}
	if requirement == nil || len(fn) > 1 {
		eval.ReportError("Authorize requires a registered requirement and at most one binding function")
		return
	}
	use := &expr.AuthorizationUseExpr{Requirement: requirement}
	if len(fn) == 1 {
		eval.Execute(fn[0], use)
	}
	a.Requirements = append(a.Requirements, use)
}

// Bind connects a top-level authorization input field to a dot-separated
// payload path. Inside a union case the selector path denotes its active
// branch. An empty path selects the entire payload or selected root branch.
func Bind(input, payload string) {
	use, ok := eval.Current().(*expr.AuthorizationUseExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	use.Bindings = append(use.Bindings, &expr.AuthorizationBindingExpr{Input: input, Payload: payload})
}

// NoAccessCheck classifies an intentional authorization exemption. The reason
// must be nonempty. Authentication requirements remain in force; use
// NoSecurity separately when anonymous calls are intended.
func NoAccessCheck(reason string) {
	a := authorizationScope()
	if a == nil {
		return
	}
	if strings.TrimSpace(reason) == "" || a.Exemption != "" {
		eval.ReportError("NoAccessCheck requires one nonempty reason")
		return
	}
	a.Exemption = reason
}

// AuthorizeBy selects exhaustive AuthorizationCase declarations using a string
// enum or union payload path. Use an empty path for a root union payload.
// This DSL is method-only and cannot be combined with method requirements.
func AuthorizeBy(payload string, fn func()) {
	m, ok := eval.Current().(*expr.MethodExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if m.Authorization != nil {
		eval.ReportError("AuthorizeBy cannot be combined with another method authorization declaration")
		return
	}
	m.Authorization = &expr.MethodAuthorizationExpr{VariantSelection: true, Selector: payload}
	eval.Execute(fn, m.Authorization)
}

// AuthorizationCase classifies one string enum value or union branch name.
// Its function must declare Authorize requirements or NoAccessCheck.
func AuthorizationCase(value string, fn func()) {
	a, ok := eval.Current().(*expr.MethodAuthorizationExpr)
	if !ok || !a.VariantSelection {
		eval.IncompatibleDSL()
		return
	}
	c := &expr.AuthorizationCaseExpr{Value: value, Authorization: &expr.MethodAuthorizationExpr{}}
	eval.Execute(fn, c.Authorization)
	a.Cases = append(a.Cases, c)
}

func authorizationScope() *expr.MethodAuthorizationExpr {
	switch current := eval.Current().(type) {
	case *expr.MethodExpr:
		if current.Authorization == nil {
			current.Authorization = &expr.MethodAuthorizationExpr{}
		}
		return current.Authorization
	case *expr.MethodAuthorizationExpr:
		return current
	default:
		eval.IncompatibleDSL()
		return nil
	}
}
