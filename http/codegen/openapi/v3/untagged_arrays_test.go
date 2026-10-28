package openapiv3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestUntaggedArrayObjectContract(t *testing.T) {
	var vacuum contractVacuum
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
		vacuum = installContractVacuum(t)
	}
	for _, version := range []struct{ target, want string }{
		{"3.1", openapiv3.OpenAPICompatibilityVersion}, {"3.2", openapiv3.OpenAPIVersion},
	} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.UntaggedArraysDSL, version.target, version.want)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			schemas := spec["components"].(map[string]any)["schemas"].(map[string]any)
			found := false
			for _, raw := range schemas {
				schema := raw.(map[string]any)
				branches, ok := schema["oneOf"].([]any)
				if !ok || len(branches) != 2 {
					continue
				}
				kinds := make(map[string]bool)
				for _, branch := range branches {
					ref := branch.(map[string]any)["$ref"].(string)
					component := schemas[filepath.Base(ref)].(map[string]any)
					kind, ok := component["type"].(string)
					if ok {
						kinds[kind] = true
					}
				}
				if kinds["array"] && kinds["object"] {
					found = true
					require.NotContains(t, schema, "discriminator")
				}
			}
			require.True(t, found, "missing mixed array/object oneOf")
			foundRecord := false
			for name, raw := range schemas {
				schema := raw.(map[string]any)
				properties, ok := schema["properties"].(map[string]any)
				if !ok || properties["status"] == nil {
					continue
				}
				foundRecord = true
				root := map[string]any{"components": spec["components"], "$ref": "#/components/schemas/" + name}
				for _, value := range []struct {
					wire string
					want bool
				}{
					{`{"status":null}`, true}, {`{"status":"open"}`, true},
					{`{"status":"closed"}`, false}, {`{"status":"other"}`, false}, {`{}`, false},
				} {
					result, err := gojsonschema.Validate(gojsonschema.NewGoLoader(root), gojsonschema.NewStringLoader(value.wire))
					require.NoError(t, err)
					require.Equal(t, value.want, result.Valid(), "%s: %s", name, value.wire)
				}
			}
			require.True(t, foundRecord, "missing nullable enum record")
			foundMap := false
			for name, raw := range schemas {
				schema := raw.(map[string]any)
				properties, ok := schema["properties"].(map[string]any)
				if !ok || properties["lookup"] == nil {
					continue
				}
				foundMap = true
				lookup := properties["lookup"].(map[string]any)
				require.Equal(t, "object", lookup["type"])
				require.IsType(t, map[string]any{}, lookup["additionalProperties"])
				root := map[string]any{"components": spec["components"], "$ref": "#/components/schemas/" + name}
				for _, value := range []struct {
					wire string
					want bool
				}{
					{"{\"total\":0,\"properties\":{}}", true},
					{"{\"total\":1,\"properties\":{\"a\":[1,null,{\"nested\":true}]},\"labels\":{\"first\":\"ok\"},\"lookup\":{\"item\":{\"name\":\"ok\",\"label\":null}}}", true},
					{"{\"total\":0,\"nested\":{\"outer\":{\"inner\":[{\"name\":\"ok\"}]}}}", true},
					{"{\"total\":0,\"nullable\":null}", true},
					{"{\"total\":0,\"nullable\":{\"a\":null,\"b\":{\"name\":\"ok\"}}}", true},
					{"{\"total\":0,\"tree\":{\"name\":\"root\",\"children\":{\"leaf\":{\"name\":\"leaf\"}}}}", true},
					{"{\"total\":0,\"properties\":null}", false},
					{"{\"total\":0,\"properties\":[]}", false},
					{"{\"total\":0,\"labels\":{}}", false},
					{"{\"total\":0,\"labels\":{\"a\":\"ok\",\"b\":\"ok\",\"c\":\"ok\"}}", false},
					{"{\"total\":0,\"labels\":{\"a\":\"x\"}}", false},
					{"{\"total\":0,\"lookup\":{\"a\":{\"name\":\"ok\"},\"b\":{\"name\":\"x\"}}}", false},
					{"{\"total\":0,\"lookup\":{\"a\":null}}", false},
					{"{\"total\":0,\"lookup\":{\"a\":{\"name\":\"ok\",\"extra\":1}}}", false},
					{"{\"total\":0,\"nested\":{\"outer\":{\"inner\":[{\"name\":\"x\"}]}}}", false},
					{"{\"total\":0,\"tree\":{\"name\":\"root\",\"children\":{\"leaf\":{}}}}", false},
					{"{\"total\":0,\"nullable\":{\"a\":{\"name\":\"x\"}}}", false},
				} {
					result, err := gojsonschema.Validate(gojsonschema.NewGoLoader(root), gojsonschema.NewStringLoader(value.wire))
					require.NoError(t, err)
					require.Equal(t, value.want, result.Valid(), "%s: %s", name, value.wire)
				}
			}
			require.True(t, foundMap, "missing map-valued page")

			if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
				dir := t.TempDir()
				path := filepath.Join(dir, "openapi.yaml")
				require.NoError(t, os.WriteFile(path, artifacts.YAML, 0o600))
				require.NoError(t, vacuum.lint(path))
			}
		})
	}
}
