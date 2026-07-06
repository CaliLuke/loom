package securityreq

import "github.com/CaliLuke/loom/expr"

// Effective returns the effective security requirements after expanding
// session-auth transport schemes and removing duplicates.
func Effective(requirements []*expr.SecurityExpr, sessionAuths []*expr.SessionAuthExpr) []*expr.SecurityExpr {
	merged := make([]*expr.SecurityExpr, 0, len(requirements)+len(sessionAuths))
	merged = append(merged, requirements...)
	for _, sessionAuth := range sessionAuths {
		for _, transport := range sessionAuth.Transports {
			if transport == nil || transport.Scheme == nil {
				continue
			}
			req := &expr.SecurityExpr{
				Schemes: []*expr.SchemeExpr{expr.DupScheme(transport.Scheme)},
			}
			if !containsRequirement(merged, req) {
				merged = append(merged, req)
			}
		}
	}
	return merged
}

// OpenAPI projects security requirements to OpenAPI security requirement
// objects.
func OpenAPI(requirements []*expr.SecurityExpr) []map[string][]string {
	if len(requirements) == 0 {
		return nil
	}
	result := make([]map[string][]string, len(requirements))
	for i, requirement := range requirements {
		schemes := make(map[string][]string, len(requirement.Schemes))
		for _, scheme := range requirement.Schemes {
			scopes := make([]string, 0)
			if scheme.Kind == expr.OAuth2Kind {
				if len(requirement.Scopes) > 0 {
					scopes = requirement.Scopes
				}
			}
			schemes[scheme.Hash()] = scopes
		}
		result[i] = schemes
	}
	return result
}

func containsRequirement(requirements []*expr.SecurityExpr, candidate *expr.SecurityExpr) bool {
	for _, requirement := range requirements {
		if len(requirement.Scopes) != len(candidate.Scopes) || len(requirement.Schemes) != len(candidate.Schemes) {
			continue
		}
		matched := true
		for i, scope := range requirement.Scopes {
			if candidate.Scopes[i] != scope {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		for i, scheme := range requirement.Schemes {
			other := candidate.Schemes[i]
			if scheme.Kind != other.Kind || scheme.SchemeName != other.SchemeName || scheme.In != other.In || scheme.Name != other.Name {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
