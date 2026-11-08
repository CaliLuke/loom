package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelfProjection(t *testing.T) {
	for _, self := range []string{"https://e.test/api/openapi.yaml", "openapi.yaml", "#description"} {
		document, diagnostics, err := Analyze(resolverEntry(self))
		require.NoError(t, err)
		require.Empty(t, diagnostics, self)
		design, err := Render(document, Options{})
		require.NoError(t, err)
		require.Contains(t, string(design), fmt.Sprintf("Meta(\"openapi:self\", %q)", self))
	}
	for _, version := range []string{"3.0.4", "3.1.1"} {
		_, diagnostics, err := Analyze([]byte(fmt.Sprintf("openapi: %s\n$self: https://e.test/api\ninfo: {title: Self, version: '1'}\npaths: {}\n", version)))
		require.NoError(t, err)
		requireDiagnosticCode(t, diagnostics, "versioned-field")
	}
	for _, self := range []string{"https://user:password@e.test/api", "https://e.test/%zz", "has space"} {
		document, diagnostics, err := Analyze(resolverEntry(self))
		require.NoError(t, err)
		require.Nil(t, document)
		requireDiagnosticCode(t, diagnostics, "document-identity")
		require.NotContains(t, diagnostics.Error(), "password")
	}
	_, diagnostics, err := Analyze([]byte("openapi: 3.2.0\ninfo: {title: Dialect, version: '1'}\njsonSchemaDialect: https://e.test/dialect\npaths: {}\n"))
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)
	require.Equal(t, "json-schema-dialect", diagnostics[0].Code)
	require.True(t, isDocumentOmission(diagnostics[0]))
	document, diagnostics, err := Analyze([]byte("openapi: 3.1.1\ninfo: {title: Older, version: '1'}\npaths: {}\n"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{})
	require.NoError(t, err)
	require.NotContains(t, string(design), "openapi:self")
}
