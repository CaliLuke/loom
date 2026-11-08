package openapiimport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegrationExternalOneOfPromotionOrigin(t *testing.T) {
	node := `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`
	branch := fmt.Sprintf(`{"type":"object","properties":{"node":%s},"required":["node"]}`, node)
	for _, tc := range []struct{ name, schemas string }{
		{"inline", fmt.Sprintf(`"Choice":{"oneOf":[%s,{"type":"boolean"}]}`, branch)},
		{"reference", fmt.Sprintf(`"Choice":{"oneOf":[{"$ref":"#/components/schemas/Branch"},{"type":"boolean"}]},"Branch":%s`, branch)},
		{"nested inline", fmt.Sprintf(`"Choice":{"oneOf":[{"type":"object","properties":{"nested":{"oneOf":[%s,{"type":"string"}]}},"required":["nested"]},{"type":"boolean"}]}`, branch)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			remote := []byte(fmt.Sprintf(`{"openapi":"3.2.0","info":{"title":"Types","version":"1"},"paths":{},"components":{"schemas":{%s}}}`, tc.schemas))

			entry := []byte(`{"openapi":"3.2.0","info":{"title":"Entry","version":"1"},"paths":{"/value":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"https://schemas.example.test/types.json#/components/schemas/Choice"}}}},"responses":{"204":{"description":"Accepted"}}}}}}`)
			path := filepath.Join(t.TempDir(), "types.json")
			require.NoError(t, os.WriteFile(path, remote, 0o600))
			document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: entry, Documents: []DocumentMapping{{URI: "https://schemas.example.test/types.json", Path: path}}}, Selection{})
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.NotNil(t, document)
			promoted := 0
			for _, schema := range document.Components.Schemas {
				if strings.Contains(schema.Name, "Node") {
					promoted++
					require.True(t, schema.origin.introduced, "external nested promotion %q lost document ownership", schema.Name)
				}
			}
			require.Positive(t, promoted)
		})
	}
}
