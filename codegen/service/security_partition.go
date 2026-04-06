package service

import "github.com/CaliLuke/loom/expr"

// ExpandRequirementSchemes clones the scheme data referenced by the transport
// requirements while preserving transport-specific location overrides.
func ExpandRequirementSchemes(requirements []*expr.SecurityExpr, base RequirementsData) SchemesData {
	var schemes SchemesData
	for _, req := range requirements {
		for _, scheme := range req.Schemes {
			projected := base.Scheme(scheme.SchemeName)
			if projected == nil {
				continue
			}
			cloned := projected.Dup()
			cloned.In = scheme.In
			schemes = schemes.Append(cloned)
		}
	}
	return schemes
}

// PartitionSchemesByIn groups schemes by their transport location and returns
// non-location schemes separately.
func PartitionSchemesByIn(schemes SchemesData) (*SchemeData, map[string]SchemesData, SchemesData) {
	grouped := make(map[string]SchemesData)
	var (
		basic    *SchemeData
		fallback SchemesData
	)
	for _, scheme := range schemes {
		switch scheme.Type {
		case "Basic":
			basic = scheme
		default:
			if scheme.In == "" {
				fallback = fallback.Append(scheme)
				continue
			}
			grouped[scheme.In] = grouped[scheme.In].Append(scheme)
		}
	}
	return basic, grouped, fallback
}
