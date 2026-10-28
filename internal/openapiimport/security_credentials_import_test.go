package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testprocess"
)

func TestImportedSecurityCredentialsGenerated(t *testing.T) {
	source, err := os.ReadFile("testdata/security_credentials/imported.json")
	require.NoError(t, err)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	before, err := json.Marshal(document, json.Deterministic(true))
	require.NoError(t, err)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	after, err := json.Marshal(document, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, string(before), string(after), "credential planning must not mutate normalized source")
	dir := requireRenderedDesignGenerates(t, rendered)
	harness, err := os.ReadFile("testdata/security_credentials/imported_wire.go.txt")
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
	require.Equal(t, input["security"], contract["security"])
	for name, raw := range input["components"].(map[string]any)["securitySchemes"].(map[string]any) {
		want := raw.(map[string]any)
		actual := contract["components"].(map[string]any)["securitySchemes"].(map[string]any)[name].(map[string]any)
		require.Equal(t, want["type"], actual["type"], name)
		if want["type"] == "apiKey" {
			require.Equal(t, want["in"], actual["in"], name)
			if want["in"] == "header" {
				require.True(t, strings.EqualFold(want["name"].(string), actual["name"].(string)), name)
			} else {
				require.Equal(t, want["name"], actual["name"], name)
			}
		}
	}
	for path, pathItem := range input["paths"].(map[string]any) {
		want, explicit := pathItem.(map[string]any)["get"].(map[string]any)["security"]
		if !explicit {
			want = input["security"]
		}
		require.Equal(t, want, operationFromImportedSpec(t, contract, path, "get")["security"], path)
	}
}

func TestSecurityCredentialUnknownScopePartialImport(t *testing.T) {
	source, err := os.ReadFile("testdata/security_credentials/imported.json")
	require.NoError(t, err)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	// Render also rejects a normalized source rather than relying on the parser.
	document.Operations[0].SecurityDefined = true
	document.Operations[0].Security = SecurityRequirements{{Schemes: []SecurityRequirementScheme{
		{Name: "Bearer"}, {Name: "OAuth", Scopes: []string{"unknown"}},
	}}}
	_, err = Render(document, Options{})
	require.ErrorContains(t, err, "not defined by scheme")

	var input map[string]any
	require.NoError(t, json.Unmarshal(source, &input))
	input["paths"] = map[string]any{
		"/bad": map[string]any{"get": map[string]any{
			"security":  []any{map[string]any{"Bearer": []any{}, "OAuth": []any{"unknown"}}},
			"responses": map[string]any{"204": map[string]any{"description": "Done"}},
		}},
		"/good": map[string]any{"get": map[string]any{
			"security": []any{}, "responses": map[string]any{"204": map[string]any{"description": "Done"}},
		}},
	}
	source, err = json.Marshal(input, json.Deterministic(true))
	require.NoError(t, err)
	for _, lossy := range []bool{false, true} {
		partial, _, err := AnalyzePartial(source, Selection{}, lossy)
		require.NoError(t, err)
		require.Len(t, partial.Skipped, 1)
		require.Equal(t, "/bad", partial.Skipped[0].Path)
		require.Contains(t, partial.Skipped[0].Diagnostics[0].Path, "#/paths/~1bad/get/security/0")
		require.Equal(t, "security-requirement", partial.Skipped[0].Diagnostics[0].Code)
		require.Len(t, partial.Document.Operations, 1)
		require.Equal(t, "/good", partial.Document.Operations[0].Path)
	}
}
