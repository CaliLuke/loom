package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalPathItemDiagnosticOrigins(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(`openapi: 3.2.0
info: {title: Local paths, version: '1'}
paths:
  /copy: {$ref: '#/paths/~1source'}
  /source:
    summary: Shared path
    get:
      responses:
        '204': {description: OK}
`))
	require.NoError(t, err)
	require.Len(t, diagnostics, 2)
	for _, diagnostic := range diagnostics {
		if diagnostic.Path == "#/paths/~1copy" {
			require.Equal(t, &SourceLocation{Document: "entry", Pointer: "#/paths/~1source", Authored: "#/paths/~1source"}, diagnostic.Origin)
		} else {
			require.Equal(t, "#/paths/~1source", diagnostic.Path)
			require.Nil(t, diagnostic.Origin)
		}
	}
	nested := Diagnostics{{Path: "#/paths/~1copy/get/responses/204/description"}}
	document.bundle.attachOrigins(nested)
	require.Equal(t, &SourceLocation{Document: "entry", Pointer: "#/paths/~1source/get/responses/204/description"}, nested[0].Origin)
}

func TestPlannerDiagnosticOrigins(t *testing.T) {
	for _, external := range []bool{false, true} {
		for _, additional := range []bool{false, true} {
			t.Run(fmt.Sprintf("external=%t/additional=%t", external, additional), func(t *testing.T) {
				operation, indent, methodPath := "    get:\n", "      ", "get"
				if additional {
					operation, indent, methodPath = "    additionalOperations:\n      PURGE:\n", "        ", "additionalOperations/PURGE"
				}
				sourcePath := "  /source:\n" + operation + indent + "parameters: [{name: filter, in: cookie, schema: {type: array, items: {type: string}}}]\n" + indent + "responses: {'204': {description: OK}}\n"
				header := "openapi: 3.2.0\ninfo: {title: Planner origins, version: '1'}\npaths:\n"
				entry := header + "  /copy: {$ref: '#/paths/~1source'}\n" + sourcePath
				files := map[string]string{}
				originDocument := "entry"
				if external {
					files["path.yaml"] = header + sourcePath
					entry = header + "  /copy: {$ref: 'path.yaml#/paths/~1source'}\n"
					originDocument = "root:path.yaml"
				}
				source := resolverFiles(t, []byte(entry), files)
				for _, selection := range []Selection{{}, {PathPrefixes: []string{"/copy"}}} {
					_, diagnostics, _, err := AnalyzeSelected(source, selection)
					require.NoError(t, err)
					assertPlannerOrigins(t, diagnostics, methodPath, originDocument)
					partial, _, err := AnalyzePartial(source, selection, false)
					require.NoError(t, err)
					assertPlannerOrigins(t, partial.Blocked, methodPath, originDocument)
					for _, skipped := range partial.Skipped {
						assertPlannerOrigins(t, skipped.Diagnostics, methodPath, originDocument)
					}
				}
			})
		}
	}
}

func assertPlannerOrigins(t *testing.T, diagnostics Diagnostics, methodPath, document string) {
	t.Helper()
	require.NotEmpty(t, diagnostics)
	for _, diagnostic := range diagnostics {
		require.Equal(t, "parameter-serialization", diagnostic.Code)
		if strings.HasPrefix(diagnostic.Path, "#/paths/~1copy/") {
			require.Equal(t, &SourceLocation{Document: document, Pointer: "#/paths/~1source/" + methodPath + "/parameters/0"}, diagnostic.Origin)
		} else {
			require.Equal(t, "#/paths/~1source/"+methodPath+"/parameters/0", diagnostic.Path)
			require.Nil(t, diagnostic.Origin)
		}
	}
}
