package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublicDiagnosticPrivacy(t *testing.T) {
	for _, uri := range []string{"https://review-user:review-secret@auth.test/a#/components/securitySchemes/K", "https://review-user:review-secret@auth.test/%ZZ#/components/securitySchemes/K", "https://review-user:review-secret with-space@auth.test/a#/components/securitySchemes/K"} {
		t.Run(uri, func(t *testing.T) {
			source := Source{Bytes: []byte(fmt.Sprintf("openapi: 3.2.0\ninfo: {title: Privacy, version: '1'}\npaths:\n  /pets:\n    get:\n      security: [{%q: []}]\n      responses: {'204': {description: OK}}\n", uri))}
			document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)
			requirePrivateOutput(t, diagnostics)
			requirePrivateOutput(t, diagnostics.Error())
			requirePrivateOutput(t, document.Provenance())
			_, err = Render(document, Options{})
			require.Error(t, err)
			requirePrivateOutput(t, err.Error())
			partial, _, err := AnalyzePartial(source, Selection{}, false)
			require.NoError(t, err)
			require.Empty(t, partial.Document.Operations)
			require.Len(t, partial.Skipped, 1)
			requirePrivateOutput(t, partial.Blocked)
			requirePrivateOutput(t, partial.Skipped)
		})
	}
	for _, key := range []string{"https://review-user:review-secret@auth.test/key", "https://review-user:review-secret with-space@auth.test/key"} {
		t.Run("duplicateEntryKey", func(t *testing.T) {
			_, _, err := Analyze([]byte(fmt.Sprintf("openapi: 3.2.0\ninfo: {title: Duplicate, version: '1'}\npaths: {}\nsecurity:\n  - %q: []\n    %q: []\n", key, key)))
			require.Error(t, err)
			require.Contains(t, err.Error(), "already defined")
			requirePrivateOutput(t, err.Error())
		})

		t.Run("duplicateExternalKey", func(t *testing.T) {
			source := resolverFiles(t, []byte("openapi: 3.2.0\ninfo: {title: Duplicate, version: '1'}\npaths:\n  /pets: {$ref: 'path.yaml'}\n"), map[string]string{"path.yaml": fmt.Sprintf("get:\n  security:\n    - %q: []\n      %q: []\n  responses: {'204': {description: OK}}\n", key, key)})
			_, diagnostics, _, err := AnalyzeSelected(source, Selection{})
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)
			require.Contains(t, diagnostics.Error(), "already defined")
			requirePrivateOutput(t, diagnostics)
		})
	}
}

func requirePrivateOutput(t *testing.T, value any) {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "review-user")
	require.NotContains(t, string(encoded), "review-secret")
}

func TestDuplicateURIKeySourceRoles(t *testing.T) {
	const key = "https://review-user:review-secret with-space@auth.test/key"
	for _, tc := range []struct{ name, tail string }{
		{"direct", fmt.Sprintf("security:\n  - %q: []\n    %q: []\n", key, key)},
		{"alias", fmt.Sprintf("x-shared: &shared\n  %q: []\n  %q: []\nsecurity: [*shared]\n", key, key)},
		{"merge", fmt.Sprintf("x-shared: &shared\n  %q: []\n  %q: []\nsecurity: [{<<: *shared}]\n", key, key)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Analyze([]byte("openapi: 3.2.0\ninfo: {title: Privacy roles, version: '1'}\npaths: {}\n" + tc.tail))
			require.ErrorContains(t, err, "already defined")
			requirePrivateOutput(t, err.Error())
		})
	}
}
