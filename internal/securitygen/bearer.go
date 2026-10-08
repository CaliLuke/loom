// Package securitygen shares credential wire classification across generators.
package securitygen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen/service"
)

// IsBearer reports whether the attribute is a JWT/OAuth credential carried in
// Authorization. Other mappings carry the authored value without a scheme.
func IsBearer(name, attribute string, schemes service.SchemesData) bool {
	if !strings.EqualFold(name, "Authorization") {
		return false
	}
	for _, scheme := range schemes {
		if scheme.KeyAttr == attribute && (scheme.Type == "JWT" || scheme.Type == "OAuth2") {
			return true
		}
	}
	return false
}
