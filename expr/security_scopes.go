package expr

// RequiredScopes returns the requirement scopes declared by scheme, in authored
// requirement order. The result is independently owned. Requirement validation
// separately rejects scopes absent from every scheme in the requirement.
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
	return scopes
}
