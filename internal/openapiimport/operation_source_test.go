package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOperationSourceSecurity(t *testing.T) {
	const auth = "openapi: 3.2.0\ninfo: {title: Auth, version: '1'}\npaths: {}\ncomponents:\n  securitySchemes:\n    K: {type: apiKey, in: header, name: X-Key}\n"
	for _, location := range []string{"get", "PURGE", "purge"} {
		for _, supplied := range []bool{false, true} {
			for _, selected := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/supplied=%t/selected=%t", location, supplied, selected), func(t *testing.T) {
					operation := "    get:\n"
					indent := "      "
					pointer := "#/paths/~1cache/get"
					if location != "get" {
						operation = "    additionalOperations:\n      " + location + ":\n"
						indent = "        "
						pointer = "#/paths/~1cache/additionalOperations/" + location
					}
					entry := "openapi: 3.2.0\ninfo: {title: Operation source, version: '1'}\npaths:\n  /cache:\n" + operation + indent + "security: [{'./auth.yaml#/components/securitySchemes/K': []}]\n" + indent + "responses: {'204': {description: OK}}\n  /public:\n    get: {security: [], responses: {'204': {description: OK}}}\n"
					files := map[string]string{}
					if supplied {
						files["auth.yaml"] = auth
					}
					source := resolverFiles(t, []byte(entry), files)
					selection := Selection{}
					if selected {
						selection.PathPrefixes = []string{"/cache"}
					}
					document, diagnostics, _, err := AnalyzeSelected(source, selection)
					require.NoError(t, err)
					partial, _, err := AnalyzePartial(source, selection, false)
					require.NoError(t, err)
					if supplied {
						require.Empty(t, diagnostics)
						require.Empty(t, partial.Skipped)
						require.NotEmpty(t, document.Operations[0].Security)
						require.Equal(t, "K", document.Operations[0].Security[0].Schemes[0].Name)
						for _, value := range []*Document{document, partial.Document} {
							rendered, renderErr := Render(value, Options{})
							require.NoError(t, renderErr)
							require.Contains(t, string(rendered), "APIKeySecurity(")
							require.Contains(t, string(rendered), "Security(ImportedKSecurity)")
						}
					} else {
						require.NotEmpty(t, diagnostics)
						require.True(t, strings.HasPrefix(diagnostics[0].Path, pointer+"/security/"), diagnostics)
						_, renderErr := Render(document, Options{})
						require.Error(t, renderErr)
						require.Len(t, partial.Skipped, 1)
						require.Equal(t, "/cache", partial.Skipped[0].Path)
						for _, op := range partial.Document.Operations {
							require.NotEqual(t, "/cache", op.Path)
						}
						if selected {
							require.Empty(t, partial.Document.Operations)
						} else {
							require.Len(t, partial.Document.Operations, 1)
						}
					}
				})
			}
		}
	}
}

func TestPartialSecurityProjection(t *testing.T) {
	source := Source{Bytes: []byte(`openapi: 3.2.0
info: {title: Root security, version: '1'}
security: [{'https://missing.test/auth#/components/securitySchemes/K': []}]
paths:
  /protected:
    get: {responses: {'204': {description: OK}}}
  /public:
    get: {security: [], responses: {'204': {description: OK}}}
`)}
	selected, diagnostics, _, err := AnalyzeSelected(source, Selection{PathPrefixes: []string{"/public"}})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, selected.Operations, 1)
	partial, _, err := AnalyzePartial(source, Selection{}, false)
	require.NoError(t, err)
	require.Len(t, partial.Document.Operations, 1)
	require.Equal(t, "/public", partial.Document.Operations[0].Path)
	require.Len(t, partial.Skipped, 1)
	require.Equal(t, "/protected", partial.Skipped[0].Path)
	require.NotEmpty(t, partial.Skipped[0].Diagnostics)
	rendered, err := Render(partial.Document, Options{})
	require.NoError(t, err)
	require.Contains(t, string(rendered), "Security()")
}

func TestOperationSourceOverrides(t *testing.T) {
	const auth = "openapi: 3.2.0\ninfo: {title: Auth, version: '1'}\npaths: {}\ncomponents:\n  securitySchemes:\n    K: {type: apiKey, in: header, name: X-Key}\n"
	for _, additional := range []bool{false, true} {
		for _, rootSupplied := range []bool{false, true} {
			for _, override := range []string{"[]", "[{'local.yaml#/components/securitySchemes/K': []}]", "[{'missing.yaml#/components/securitySchemes/K': []}]"} {
				t.Run(fmt.Sprintf("additional=%t/root=%t/override=%s", additional, rootSupplied, override), func(t *testing.T) {
					operation, indent := "    get:\n", "      "
					if additional {
						operation, indent = "    additionalOperations:\n      PURGE:\n", "        "
					}
					entry := "openapi: 3.2.0\ninfo: {title: Overrides, version: '1'}\nsecurity: [{'root.yaml#/components/securitySchemes/K': []}]\npaths:\n  /override:\n" + operation + indent + "security: " + override + "\n" + indent + "responses: {'204': {description: OK}}\n  /inherited:\n    get: {responses: {'204': {description: OK}}}\n"
					files := map[string]string{"local.yaml": auth}
					if rootSupplied {
						files["root.yaml"] = auth
					}
					source := resolverFiles(t, []byte(entry), files)
					partial, _, err := AnalyzePartial(source, Selection{}, false)
					require.NoError(t, err)
					want := 0
					if rootSupplied {
						want++
					}
					validOverride := !strings.Contains(override, "missing.yaml")
					if validOverride {
						want++
					}
					require.Len(t, partial.Document.Operations, want)
					require.Len(t, partial.Skipped, 2-want)
					selected, diagnostics, _, err := AnalyzeSelected(source, Selection{PathPrefixes: []string{"/override"}})
					require.NoError(t, err)
					if validOverride {
						require.Empty(t, diagnostics)
						rendered, renderErr := Render(selected, Options{})
						require.NoError(t, renderErr)
						if override == "[]" {
							require.Contains(t, string(rendered), "NoSecurity()")
						} else {
							require.Contains(t, string(rendered), "Security(ImportedKSecurity)")
						}
					} else {
						require.NotEmpty(t, diagnostics)
						_, err = Render(selected, Options{})
						require.Error(t, err)
					}
				})
			}
		}
	}
}

func TestAdditionalOperationIdentityDiagnostics(t *testing.T) {
	_, diagnostics, err := Analyze([]byte(`openapi: 3.2.0
info: {title: Identities, version: '1'}
paths:
  /cache:
    get: {operationId: same, responses: {'204': {description: OK}}}
    additionalOperations:
      PuRgE: {operationId: same, responses: {'204': {description: OK}}}
`))
	require.NoError(t, err)
	var paths []string
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "operation-id" {
			paths = append(paths, diagnostic.Path)
		}
	}
	require.ElementsMatch(t, []string{"#/paths/~1cache/get/operationId", "#/paths/~1cache/additionalOperations/PuRgE/operationId"}, paths)
}
