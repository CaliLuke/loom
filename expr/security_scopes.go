package expr

import "slices"

// RequiredScopes returns the requirement scopes declared by scheme, in authored
// requirement order, followed by its explicit per-scheme requirements. For
// non-OAuth schemes the explicit names are application roles. The result is
// independently owned. Validation separately rejects unknown OAuth scopes.
func (s *SecurityExpr) RequiredScopes(scheme *SchemeExpr) []string {
	var scopes []string
	for _, required := range s.Scopes {
		for _, declared := range scheme.Scopes {
			if required == declared.Name {
				scopes = append(scopes, required)
				break
			}
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
