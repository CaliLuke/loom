package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEntryAdmissionPrecedesLoading(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		unsupported  bool
	}{
		{"missing", "info: {title: Missing, version: '1'}\npaths: {}\n", true},
		{"unsupported", "openapi: 3.3.0\ninfo: {title: Future, version: '1'}\npaths: {}\n", true},
		{"syntax", "{", false},
		{"duplicate", "openapi: 3.2.0\ninfo: {title: Duplicate, version: '1'}\npaths: {}\npaths: {}\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := Source{Bytes: []byte(tc.source), Documents: []DocumentMapping{{URI: "https://external.test/", Path: "/does-not-exist-source-order"}}}
			document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
			require.Error(t, err)
			require.Nil(t, document)
			require.Empty(t, diagnostics)
			require.NotContains(t, err.Error(), "does-not-exist-source-order")
			if tc.unsupported {
				require.ErrorIs(t, err, ErrUnsupportedVersion)
			}
		})
	}
}

func TestDocumentMappingErrorBoundary(t *testing.T) {
	for _, path := range []string{"contact@email", "relative folder/contact@email", "path=with=equals@name"} {
		mapping, err := ParseDocumentMapping("https://host.example=" + path)
		require.NoError(t, err)
		require.Equal(t, "https://host.example/", mapping.URI)
		require.Equal(t, path, mapping.Path)
	}
	for _, value := range []string{"https://review-user:review-secret with=space@auth.test/path=document.yaml", "https://review-user:review-secret with-space@auth.test/path=document.yaml"} {
		_, err := ParseDocumentMapping(value)
		require.ErrorContains(t, err, "invalid URI reference syntax")
		requirePrivateOutput(t, err.Error())
	}
	_, err := ParseDocumentMapping("relative=contact@email")
	require.ErrorContains(t, err, "absolute URI")
}
