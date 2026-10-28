package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIndependentOAuthFlowScopes(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Flow inventories, version: "1"}
security: [{Auth: [read, write]}]
paths:
 /value:
  get:
   responses: {"204": {description: done}}
components:
 securitySchemes:
  Auth:
   type: oauth2
   flows:
    authorizationCode:
     authorizationUrl: https://example.com/authorize
     tokenUrl: https://example.com/token
     scopes: {read: Read with consent, shared: User grant}
    clientCredentials:
     tokenUrl: https://example.com/token
     scopes: {write: Machine write, shared: Machine grant}
    implicit:
     authorizationUrl: https://example.com/authorize
     scopes: {}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), `Scope("shared", "User grant")`)
	require.Contains(t, string(rendered), `Scope("shared", "Machine grant")`)
	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	want := map[string]any{
		"authorizationCode": map[string]any{"authorizationUrl": "https://example.com/authorize", "tokenUrl": "https://example.com/token", "scopes": map[string]any{"read": "Read with consent", "shared": "User grant"}},
		"clientCredentials": map[string]any{"tokenUrl": "https://example.com/token", "scopes": map[string]any{"write": "Machine write", "shared": "Machine grant"}},
		"implicit":          map[string]any{"authorizationUrl": "https://example.com/authorize", "scopes": map[string]any{}},
	}
	got := contract["components"].(map[string]any)["securitySchemes"].(map[string]any)["Auth"].(map[string]any)["flows"]
	require.Equal(t, want, got)
}
