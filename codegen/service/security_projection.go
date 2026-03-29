package service

import "github.com/CaliLuke/loom/expr"

// BuildRequirementsData projects expression security requirements into
// generator-facing requirement and scheme data for the given method.
func BuildRequirementsData(requirements []*expr.SecurityExpr, method *expr.MethodExpr) (RequirementsData, SchemesData) {
	reqs := make(RequirementsData, 0, len(requirements))
	var schemes SchemesData
	for _, req := range requirements {
		rs := make(SchemesData, 0, len(req.Schemes))
		for _, scheme := range req.Schemes {
			projected := BuildSchemeData(scheme, method)
			rs = rs.Append(projected)
			schemes = schemes.Append(projected)
		}
		if len(rs) == 0 {
			continue
		}
		reqs = append(reqs, &RequirementData{Schemes: rs, Scopes: req.Scopes})
	}
	return reqs, schemes
}
