package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// ScopeFor requires OAuth2 scopes from one scheme in the enclosing Security
// requirement. The scheme is a name or a scheme reference already listed in
// Security. Each scope must be declared by that scheme. Repeated calls add
// requirements; they do not replace Scope or another scheme's requirements.
func ScopeFor(scheme any, scopes ...string) {
	requireSchemeNames(scheme, scopes, true)
}

// RoleFor requires application role names from one non-OAuth scheme in the
// enclosing Security requirement. The scheme is a name or a scheme reference
// already listed in Security. Roles need no scheme-level inventory and are not
// exchanged on the wire. The application's authentication callback receives
// them in RequiredScopes and validates them against its own role assignments.
func RoleFor(scheme any, roles ...string) {
	requireSchemeNames(scheme, roles, false)
}

func requireSchemeNames(scheme any, names []string, oauth bool) {
	current, ok := eval.Current().(*expr.SecurityExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	var name string
	switch value := scheme.(type) {
	case string:
		name = value
	case *expr.SchemeExpr:
		if value != nil {
			name = value.SchemeName
		}
	default:
		eval.InvalidArgError("security scheme or security scheme name", scheme)
		return
	}
	for _, selected := range current.Schemes {
		if selected.SchemeName != name {
			continue
		}
		if (selected.Kind == expr.OAuth2Kind) != oauth || selected.Kind == expr.NoKind {
			eval.ReportError("ScopeFor requires an OAuth2 scheme; RoleFor requires a non-OAuth scheme")
			return
		}
		if current.SchemeRequirements == nil {
			current.SchemeRequirements = make(map[string][]string)
		}
		current.SchemeRequirements[name] = append(current.SchemeRequirements[name], names...)
		return
	}
	eval.ReportError("security scheme %q is not in this requirement", name)
}
