package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestImportedMultipartBodyRequirednessRoundTrip(t *testing.T) {
	fixture := readFixture(t, "discriminator/media.yaml")
	var input, requiredControl, requiredMemberControl map[string]any
	require.NoError(t, yaml.Unmarshal(fixture, &input))
	require.NoError(t, yaml.Unmarshal(fixture, &requiredControl))
	require.NoError(t, yaml.Unmarshal(fixture, &requiredMemberControl))
	requiredPath := requiredControl["paths"].(map[string]any)["/multipart"].(map[string]any)
	requiredOperation := requiredPath["post"].(map[string]any)
	requiredOperation["operationId"] = "multipart.required"
	requiredOperation["requestBody"].(map[string]any)["required"] = true
	input["paths"].(map[string]any)["/multipart-required"] = requiredPath
	requiredMemberPath := requiredMemberControl["paths"].(map[string]any)["/multipart"].(map[string]any)
	requiredMemberOperation := requiredMemberPath["post"].(map[string]any)
	requiredMemberOperation["operationId"] = "multipart.requiredMember"
	requiredMemberBody := requiredMemberOperation["requestBody"].(map[string]any)
	requiredMemberBody["content"].(map[string]any)["multipart/form-data"].(map[string]any)["schema"].(map[string]any)["required"] = []any{"event"}
	input["paths"].(map[string]any)["/multipart-required-member"] = requiredMemberPath
	source, err := json.Marshal(input, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	cases := []struct {
		name     string
		path     string
		required bool
	}{
		{"optional_multipart", "/multipart", false},
		{"required_multipart", "/multipart-required", true},
		{"optional_multipart_required_member", "/multipart-required-member", false},
		{"optional_form", "/form", false},
	}
	for _, tc := range cases {
		t.Run("analyzed/"+tc.name, func(t *testing.T) {
			found := false
			for _, operation := range document.Operations {
				if operation.Path != tc.path {
					continue
				}
				found = true
				require.NotNil(t, operation.RequestBody)
				require.Equal(t, tc.required, operation.RequestBody.Required)
				if tc.path == "/multipart-required-member" {
					require.Contains(t, operation.RequestBody.Schema.Required, "event")
				}
			}
			require.True(t, found)
		})
	}
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, design)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "source.json"), source, 0o600))
	output := readGeneratedOpenAPIContract(t, directory)
	for _, tc := range cases {
		t.Run("exported/"+tc.name, func(t *testing.T) {
			operation := operationFromImportedSpec(t, output, tc.path, "post")
			body := operation["requestBody"].(map[string]any)
			if ref, ok := body["$ref"].(string); ok {
				const prefix = "#/components/requestBodies/"
				require.True(t, strings.HasPrefix(ref, prefix))
				body = output["components"].(map[string]any)["requestBodies"].(map[string]any)[strings.TrimPrefix(ref, prefix)].(map[string]any)
			}
			require.Equal(t, tc.required, body["required"] == true, "request body presence must survive import and generation")
			if tc.path == "/multipart-required-member" {
				schema := body["content"].(map[string]any)["multipart/form-data"].(map[string]any)["schema"].(map[string]any)
				if ref, ok := schema["$ref"].(string); ok {
					const prefix = "#/components/schemas/"
					require.True(t, strings.HasPrefix(ref, prefix))
					schema = output["components"].(map[string]any)["schemas"].(map[string]any)[strings.TrimPrefix(ref, prefix)].(map[string]any)
				}
				require.Contains(t, schema["required"], "event", "optional body retains required supplied members")
			}
		})
	}
	requireDiscriminatorSpecifications(t, directory)
	requireIndependentRegeneration(t, directory)
}
