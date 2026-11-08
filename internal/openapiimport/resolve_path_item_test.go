package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePathItems(t *testing.T) {
	for _, tc := range []struct {
		name, item, other, second, message string
		fields                             []string
	}{
		{"externalRoot", "{$ref: 'path.yaml'}", "get: {responses: {}}", "", "", []string{"get"}},
		{"chain", "{$ref: 'path.yaml'}", "$ref: second.yaml", "post: {responses: {}}", "", []string{"post"}},
		{"siblings", "{$ref: 'path.yaml', summary: Here}", "get: {responses: {}}", "", "", []string{"summary", "get"}},
		{"overlap", "{$ref: 'path.yaml', summary: Here}", "summary: There", "", "appears in both", nil},
		{"cycle", "{$ref: 'path.yaml'}", "$ref: second.yaml", "$ref: path.yaml", "cycle", nil},
		{"missing", "{$ref: 'absent.yaml'}", "{}", "", "", nil},
		{"local", "{$ref: '#/paths/~1other'}", "{}", "", "", []string{"get"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := resolverFiles(t, []byte("openapi: 3.2.0\ninfo: {title: Paths, version: '1'}\npaths:\n  /pets: "+tc.item+"\n  /other: {get: {responses: {}}}\n"), map[string]string{"path.yaml": tc.other, "second.yaml": tc.second})
			resolved, _, err := resolveDescription(source)
			require.NoError(t, err)
			fields, diagnostic, err := effectivePathItem(resolved, targetKey{"entry", "/paths/~1pets"})
			require.NoError(t, err)
			if tc.fields == nil {
				require.NotNil(t, diagnostic)
				require.Contains(t, diagnostic.Message, tc.message)
				require.Empty(t, fields)
			} else {
				require.Nil(t, diagnostic)
				names := make([]string, 0, len(fields))
				for _, field := range fields {
					names = append(names, field.name)
				}
				require.Equal(t, tc.fields, names)
			}
		})
	}
}

func TestResolvePathItemsOperationIdentity(t *testing.T) {
	for _, operationID := range []string{"", "listPets"} {
		source := resolverFiles(t, []byte("openapi: 3.2.0\ninfo: {title: Shared, version: '1'}\npaths:\n  /a: {$ref: 'path.yaml'}\n  /b: {$ref: 'path.yaml'}\n"), map[string]string{"path.yaml": "get:\n  operationId: '" + operationID + "'\n  responses: {'204': {description: OK}}\n"})
		document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
		require.NoError(t, err)
		require.Len(t, document.Operations, 2)
		if operationID == "" {
			require.Empty(t, diagnostics)
			continue
		}
		require.Len(t, diagnostics, 2)
		require.Equal(t, "operation-id", diagnostics[0].Code)
		require.Equal(t, "#/paths/~1a/get/operationId", diagnostics[0].Path)
		require.Equal(t, "#/paths/~1b/get/operationId", diagnostics[1].Path)
		require.NotNil(t, diagnostics[0].Origin)
		_, diagnostics, _, err = AnalyzeSelected(source, Selection{Paths: []string{"/a"}})
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		partial, _, err := AnalyzePartial(source, Selection{}, false)
		require.NoError(t, err)
		require.Empty(t, partial.Document.Operations)
		require.Len(t, partial.Skipped, 2)
	}
}
