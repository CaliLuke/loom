package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testprocess"
)

func TestSecurityCredentialRootScopePartialImport(t *testing.T) {
	for _, paths := range [][]string{{"/public"}, {"/oauth_roles"}, {"/public", "/oauth_roles"}, {}} {
		for _, lossy := range []bool{false, true} {
			source := partialCredentialSource(t, paths)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)
			fatal, _ := diagnostics.Classify(lossy)
			require.NotEmpty(t, fatal, "strict import must reject unsupported root requirements")
			_, err = Render(document, Options{})
			require.ErrorContains(t, err, "use different scopes")

			partial, _, err := AnalyzePartial(source, Selection{}, lossy)
			if err != nil {
				t.Errorf("paths=%v lossy=%v: %v", paths, lossy, err)
				continue
			}
			require.Len(t, partial.Skipped, 1)
			require.Equal(t, "/inherit", partial.Skipped[0].Path)
			require.Len(t, partial.Skipped[0].Diagnostics, 1)
			require.Equal(t, "#/security/0", partial.Skipped[0].Diagnostics[0].Path)
			require.Equal(t, "security-requirement", partial.Skipped[0].Diagnostics[0].Code)
			require.Len(t, partial.Document.Operations, len(paths))
			require.False(t, partial.Document.SecurityDefined)
			require.Empty(t, partial.Document.Security)
			for _, operation := range partial.Document.Operations {
				require.True(t, operation.SecurityDefined)
				if operation.Path == "/public" {
					require.Empty(t, operation.Security)
				} else {
					require.Len(t, operation.Security, 1)
					require.Len(t, operation.Security[0].Schemes, 2)
					for _, scheme := range operation.Security[0].Schemes {
						require.Equal(t, []string{"read"}, scheme.Scopes)
					}
				}
			}
			_, err = Render(partial.Document, Options{PackageName: "design"})
			require.NoError(t, err)
		}
	}
}

func TestImportedPartialSecurityCredentialsGenerated(t *testing.T) {
	source := partialCredentialSource(t, []string{"/public", "/oauth_roles"})
	partial, _, err := AnalyzePartial(source, Selection{}, false)
	require.NoError(t, err)
	rendered, err := Render(partial.Document, Options{PackageName: "design"})
	require.NoError(t, err)
	dir := requireRenderedDesignGenerates(t, rendered)
	harness, err := os.ReadFile("testdata/security_credentials/partial_wire.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), harness, 0o600))
	for _, args := range [][]string{{"test", "-mod=mod", "-v", "./..."}, {"build", "./..."}, {"vet", "./..."}} {
		cmd := testprocess.CommandContext(t.Context(), "go", args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		t.Log(string(output))
	}
	generated, err := os.ReadFile(filepath.Join(dir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(generated)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)
	var contract, input map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	require.NoError(t, json.Unmarshal(source, &input))
	require.Empty(t, contract["security"])
	paths := contract["paths"].(map[string]any)
	require.Len(t, paths, 2)
	for _, path := range []string{"/public", "/oauth_roles"} {
		want := input["paths"].(map[string]any)[path].(map[string]any)["get"].(map[string]any)["security"]
		require.Equal(t, want, operationFromImportedSpec(t, contract, path, "get")["security"], path)
	}
}

func partialCredentialSource(t *testing.T, retained []string) []byte {
	t.Helper()
	source, err := os.ReadFile("testdata/security_credentials/imported.json")
	require.NoError(t, err)
	var input map[string]any
	require.NoError(t, json.Unmarshal(source, &input))
	input["security"] = []any{map[string]any{"Bearer": []any{}, "OAuth": []any{"read"}}}
	paths := input["paths"].(map[string]any)
	selected := map[string]any{"/inherit": paths["/inherit"]}
	for _, path := range retained {
		selected[path] = paths[path]
	}
	input["paths"] = selected
	source, err = json.Marshal(input, json.Deterministic(true))
	require.NoError(t, err)
	return source
}
