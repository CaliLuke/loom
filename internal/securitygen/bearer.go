// Package securitygen shares credential wire classification across generators.
package securitygen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen/service"
)

// BearerData records framing and alternative membership for one credential.
type BearerData struct {
	// Framed identifies a JWT/OAuth credential in Authorization.
	Framed bool
	// AllowOtherScheme permits foreign scheme framing to leave the credential
	// absent when the payload does not require it and another requirement omits it.
	AllowOtherScheme bool
}

// PlanBearer retains requirement topology while resolving Authorization framing.
// Other mappings carry raw authored values. Requiredness belongs to the payload
// and is never weakened by an alternative authentication requirement.
func PlanBearer(
	name, attribute string,
	required bool,
	schemes service.SchemesData,
	requirements service.RequirementsData,
) BearerData {
	var plan BearerData
	if !strings.EqualFold(name, "Authorization") {
		return plan
	}
	for _, scheme := range schemes {
		if scheme.KeyAttr == attribute && (scheme.Type == "JWT" || scheme.Type == "OAuth2") {
			plan.Framed = true
			break
		}
	}
	if !plan.Framed || required {
		return plan
	}
	for _, requirement := range requirements {
		usesCredential := false
		for _, scheme := range requirement.Schemes {
			if scheme.KeyAttr == attribute || scheme.UsernameAttr == attribute || scheme.PasswordAttr == attribute {
				usesCredential = true
				break
			}
		}
		if !usesCredential {
			plan.AllowOtherScheme = true
			break
		}
	}
	return plan
}
