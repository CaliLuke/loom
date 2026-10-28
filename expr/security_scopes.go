package expr

import "slices"

// RequiredScopes returns the requirement scopes declared by scheme, in authored
// requirement order, followed by its explicit per-scheme requirements. For
// non-OAuth schemes the explicit names are application roles. The result is
// independently owned. Validation separately rejects unknown OAuth scopes.
func (s *SecurityExpr) RequiredScopes(scheme *SchemeExpr) []string {
	var scopes []string
	inventory := scheme.ScopeNames()
	for _, required := range s.Scopes {
		if slices.Contains(inventory, required) {
			scopes = append(scopes, required)
		}
	}
	for _, name := range s.SchemeRequirements[scheme.SchemeName] {
		if !slices.Contains(scopes, name) {
			scopes = append(scopes, name)
		}
	}
	return scopes
}

func dupSchemeRequirements(source map[string][]string) map[string][]string {
	if source == nil {
		return nil
	}
	result := make(map[string][]string, len(source))
	for scheme, names := range source {
		result[scheme] = slices.Clone(names)
	}
	return result
}

// FlowScopes returns a flow's effective scope inventory. A nil flow inventory
// inherits the scheme defaults; a nonnil empty inventory remains empty. The
// returned scope declarations are independently owned. Repeated names retain
// their first position and last description, matching OpenAPI map semantics.
func (s *SchemeExpr) FlowScopes(flow *FlowExpr) []*ScopeExpr {
	inventory := flow.Scopes
	if inventory == nil {
		inventory = s.Scopes
	}
	if inventory == nil {
		return nil
	}
	result := make([]*ScopeExpr, 0, len(inventory))
	positions := make(map[string]int, len(inventory))
	for _, scope := range inventory {
		if position, exists := positions[scope.Name]; exists {
			result[position].Description = scope.Description
			continue
		}
		positions[scope.Name] = len(result)
		copy := *scope
		result = append(result, &copy)
	}
	return result
}

// ScopeNames returns the names available to requirement validation and auth
// callbacks, in first-declaration order. For OAuth2 it collects names from the
// effective flow inventories without combining their descriptions or changing
// any flow. Flow selection and granted permissions remain application-owned.
func (s *SchemeExpr) ScopeNames() []string {
	inventories := [][]*ScopeExpr{s.Scopes}
	if s.Kind == OAuth2Kind && len(s.Flows) > 0 {
		inventories = make([][]*ScopeExpr, len(s.Flows))
		for i, flow := range s.Flows {
			inventories[i] = s.FlowScopes(flow)
		}
	}
	var names []string
	for _, inventory := range inventories {
		for _, scope := range inventory {
			if !slices.Contains(names, scope.Name) {
				names = append(names, scope.Name)
			}
		}
	}
	return names
}

func dupScopeDeclarations(source []*ScopeExpr) []*ScopeExpr {
	if source == nil {
		return nil
	}
	result := make([]*ScopeExpr, len(source))
	for i, scope := range source {
		value := *scope
		result[i] = &value
	}
	return result
}
