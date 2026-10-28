package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecurityPerSchemeRequirements(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Per-scheme requirements, version: "1"}
security: [{Read: [read], Write: [], Key: [admin]}, {}]
paths:
 /secure:
  get:
   responses: {"204": {description: done}}
components:
 securitySchemes:
  Read:
   type: oauth2
   flows:
    clientCredentials: {tokenUrl: https://example.com/token, scopes: {read: Read, write: Write}}
  Write:
   type: oauth2
   flows:
    clientCredentials: {tokenUrl: https://example.com/token, scopes: {read: Read, write: Write}}
  Key: {type: apiKey, in: header, name: X-Key}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), `ScopeFor(ImportedReadSecurity, "read")`)
	require.Contains(t, string(rendered), `RoleFor(ImportedKeySecurity, "admin")`)
	require.NotContains(t, string(rendered), `ScopeFor(ImportedWriteSecurity, "read")`)
	for _, tc := range []struct {
		name string
		refs []SecurityRequirementScheme
		want string
	}{
		{"unknown scheme", []SecurityRequirementScheme{{Name: "missing"}}, "not defined"},
		{"unknown scope", []SecurityRequirementScheme{{Name: "Read", Scopes: []string{"missing"}}}, "not defined by scheme"},
		{"duplicate scheme", []SecurityRequirementScheme{{Name: "Read"}, {Name: "Read"}}, "occurs twice"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copy := *document
			copy.Security = SecurityRequirements{{Schemes: tc.refs}}
			_, err := Render(&copy, Options{})
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestSecurityRoleVersionBoundary(t *testing.T) {
	for _, version := range []string{"3.0.4", "3.1.1", "3.2.0"} {
		for _, roles := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/roles=%t", version, roles), func(t *testing.T) {
				keyNames := "[]"
				if roles {
					keyNames = "[admin]"
				}
				source := []byte(fmt.Sprintf(`openapi: %s
info: {title: Versioned requirements, version: "1"}
security: [{Read: [read], Write: [], Key: %s}]
paths:
 /secure:
  get:
   responses: {"204": {description: done}}
components:
 securitySchemes:
  Read:
   type: oauth2
   flows:
    clientCredentials: {tokenUrl: https://example.com/token, scopes: {read: Read}}
  Write:
   type: oauth2
   flows:
    clientCredentials: {tokenUrl: https://example.com/token, scopes: {read: Read}}
  Key: {type: apiKey, in: header, name: X-Key}
`, version, keyNames))
				document, diagnostics, err := Analyze(source)
				require.NoError(t, err)
				_, renderErr := Render(document, Options{})
				if roles && strings.HasPrefix(version, "3.0.") {
					require.Contains(t, diagnostics.Error(), "must be empty in OpenAPI 3.0")
					require.ErrorContains(t, renderErr, "must be empty in OpenAPI 3.0")
				} else {
					require.Empty(t, diagnostics)
					require.NoError(t, renderErr)
				}
			})
		}
	}
}
