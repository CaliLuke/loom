package openapiimport

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecurityURIRequirements(t *testing.T) {
	const target = "https://e.test/auth.yaml#/components/securitySchemes/Auth"
	const auth = "openapi: 3.2.0\ninfo: {title: Auth, version: '1'}\npaths: {}\ncomponents:\n  securitySchemes:\n    Auth: {type: apiKey, in: header, name: X-Key}\n"
	for _, tc := range []struct {
		name, self, requirements, wantURI string
		wantScopes                        [][]string
	}{
		{"singleSpelling", "", "- 'https://E.TEST:443/auth.yaml#/components/securitySchemes/Auth': [reader]", "https://E.TEST:443/auth.yaml#/components/securitySchemes/Auth", [][]string{{"reader"}}},
		{"stableRelative", "https://e.test/openapi.yaml", "- './auth.yaml#/components/securitySchemes/Auth': [reader]", "./auth.yaml#/components/securitySchemes/Auth", [][]string{{"reader"}}},
		{"scopeUnion", "https://e.test/openapi.yaml", "- 'https://e.test/auth.yaml#/components/securitySchemes/Auth': [a]\n        './auth.yaml#/components/securitySchemes/Auth': [b, a]", target, [][]string{{"a", "b"}}},
		{"alternatives", "https://e.test/openapi.yaml", "- './auth.yaml#/components/securitySchemes/Auth': [b]\n      - 'https://e.test/auth.yaml#/components/securitySchemes/Auth': [a]", target, [][]string{{"b"}, {"a"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := ""
			if tc.self != "" {
				self = fmt.Sprintf("$self: %q\n", tc.self)
			}
			entry := []byte("openapi: 3.2.0\n" + self + "info: {title: Security, version: '1'}\npaths:\n  /pets:\n    get:\n      security:\n      " + tc.requirements + "\n      responses: {'204': {description: OK}}\n")
			source := resolverFiles(t, entry, map[string]string{"auth.yaml": auth})
			source.Documents = []DocumentMapping{{URI: "https://e.test/auth.yaml", Path: filepath.Join(filepath.Dir(source.RetrievalPath), "auth.yaml")}}
			document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Len(t, document.Operations[0].Security, len(tc.wantScopes))
			for i, requirement := range document.Operations[0].Security {
				require.Len(t, requirement.Schemes, 1)
				require.Equal(t, tc.wantScopes[i], requirement.Schemes[0].Scopes)
			}
			design, err := Render(document, Options{})
			require.NoError(t, err)
			require.Contains(t, string(design), fmt.Sprintf("Meta(\"openapi:security:uri\", %q)", tc.wantURI))
		})
	}
	t.Run("keepMixed", func(t *testing.T) {
		source := []byte(`openapi: 3.2.0
$self: https://e.test/openapi.yaml
info: {title: Local auth, version: '1'}
security:
  - Auth: [a]
    https://e.test/openapi.yaml#/components/securitySchemes/Auth: [b]
paths:
  /pets:
    get: {responses: {'204': {description: OK}}}
components:
  securitySchemes:
    Auth: {type: apiKey, in: header, name: X-Key}
`)
		document, diagnostics, err := Analyze(source)
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		require.Len(t, document.Security[0].Schemes, 1)
		require.Equal(t, []string{"a", "b"}, document.Security[0].Schemes[0].Scopes)
		design, err := Render(document, Options{})
		require.NoError(t, err)
		require.NotContains(t, string(design), "openapi:security:uri")
	})
}

func TestSecurityURIContainingDocument(t *testing.T) {
	const auth = "openapi: 3.2.0\ninfo: {title: Auth, version: '1'}\npaths: {}\ncomponents:\n  securitySchemes:\n    Auth: {type: apiKey, in: header, name: X-Key}\n"
	for _, tc := range []struct{ name, self string }{
		{"authoredCollapse", ""},
		{"entryBase", "$self: https://entry.test/api.yaml\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := []byte("openapi: 3.2.0\n" + tc.self + "info: {title: Path base, version: '1'}\npaths:\n  /pets: {$ref: 'https://paths.test/pets.yaml'}\n")
			source := resolverFiles(t, entry, map[string]string{
				"path.yaml": "get:\n  security:\n    - './auth.yaml#/components/securitySchemes/Auth': [reader]\n  responses: {'204': {description: OK}}\n",
				"auth.yaml": auth,
			})
			root := filepath.Dir(source.RetrievalPath)
			source.Documents = []DocumentMapping{
				{URI: "https://paths.test/pets.yaml", Path: filepath.Join(root, "path.yaml")},
				{URI: "https://paths.test/auth.yaml", Path: filepath.Join(root, "auth.yaml")},
			}
			document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Len(t, document.Components.SecuritySchemes, 1)
			require.Equal(t, "X-Key", document.Components.SecuritySchemes[0].ParameterName)
			require.Equal(t, "https://paths.test/auth.yaml#/components/securitySchemes/Auth", document.Components.SecuritySchemes[0].URI)
		})
	}
	t.Run("fileTarget", func(t *testing.T) {
		source := resolverFiles(t, []byte("openapi: 3.2.0\ninfo: {title: Files, version: '1'}\nsecurity:\n  - 'auth.yaml#/components/securitySchemes/Auth': []\npaths:\n  /pets:\n    get: {responses: {'204': {description: OK}}}\n"), map[string]string{"auth.yaml": auth})
		document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		require.Equal(t, "Auth", document.Security[0].Schemes[0].Name)
		require.Empty(t, document.Components.SecuritySchemes[0].URI)
	})
}

func TestSecurityURINamePrecedenceAndVersion(t *testing.T) {
	for _, entryScheme := range []bool{false, true} {
		entry := "openapi: 3.2.0\ninfo: {title: Names, version: '1'}\npaths:\n  /pets: {$ref: 'other.yaml#/paths/~1pets'}\n"
		if entryScheme {
			entry += "components:\n  securitySchemes:\n    Auth: {type: apiKey, in: header, name: X-Entry}\n"
		}
		source := resolverFiles(t, []byte(entry), map[string]string{"other.yaml": `openapi: 3.2.0
info: {title: Other, version: '1'}
paths:
  /pets:
    get:
      security: [{Auth: []}]
      responses: {'204': {description: OK}}
components:
  securitySchemes:
    Auth: {type: apiKey, in: header, name: X-Other}
`})
		document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
		require.NoError(t, err)
		if entryScheme {
			require.Empty(t, diagnostics)
			require.Equal(t, "X-Entry", document.Components.SecuritySchemes[0].ParameterName)
		} else {
			require.Contains(t, diagnostics.Error(), "entry component")
		}
	}
	for _, version := range []string{"3.0.4", "3.1.1"} {
		_, diagnostics, err := Analyze([]byte(fmt.Sprintf("openapi: %s\ninfo: {title: Versions, version: '1'}\nsecurity:\n  - '#/components/securitySchemes/Auth': []\npaths: {}\ncomponents:\n  securitySchemes:\n    Auth: {type: apiKey, in: header, name: X-Key}\n", version)))
		require.NoError(t, err)
		require.Len(t, diagnostics, 1)
		require.Equal(t, "security-requirement", diagnostics[0].Code)
	}
}
