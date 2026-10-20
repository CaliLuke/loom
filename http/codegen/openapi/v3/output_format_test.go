package openapiv3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestFilesSelectOutputFormat(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		wantPaths []string
	}{
		{
			name: "default both",
			wantPaths: []string{
				filepath.Join(codegen.Gendir, "http", "openapi.json"),
				filepath.Join(codegen.Gendir, "http", "openapi.yaml"),
			},
		},
		{
			name:      "JSON only",
			format:    "json",
			wantPaths: []string{filepath.Join(codegen.Gendir, "http", "openapi.json")},
		},
		{
			name:      "YAML only",
			format:    "yaml",
			wantPaths: []string{filepath.Join(codegen.Gendir, "http", "openapi.yaml")},
		},
		{
			name:   "explicit both",
			format: "both",
			wantPaths: []string{
				filepath.Join(codegen.Gendir, "http", "openapi.json"),
				filepath.Join(codegen.Gendir, "http", "openapi.yaml"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := openAPIOutputRoot(t, test.format)

			files, err := openapiv3.Files(root)

			require.NoError(t, err)
			paths := make([]string, len(files))
			for i, file := range files {
				paths[i] = file.Path
			}
			require.Equal(t, test.wantPaths, paths)
		})
	}
}

func TestFilesRejectUnsupportedOutputFormat(t *testing.T) {
	root := openAPIOutputRoot(t, "toml")

	_, err := openapiv3.Files(root)

	require.ErrorContains(t, err, `openapi output format "toml" must be one of json, yaml, or both`)
}

func TestFilesValidatesOpenAPITagMetadata(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		values  []string
		wantErr string
	}{
		{name: "empty annotation", key: "openapi:tag:billing:desc", wantErr: "requires exactly one value, got 0"},
		{name: "multiple annotation values", key: "openapi:tag:billing:desc", values: []string{"one", "two"}, wantErr: "requires exactly one value, got 2"},
		{name: "malformed encoded name", key: "openapi:tag-id:not-base64!:desc", values: []string{"bad"}, wantErr: "malformed encoded OpenAPI tag name"},
		{name: "encoded ordinary name collision", key: "openapi:tag-id:YmlsbGluZw", wantErr: "must be valid UTF-8 and contain a colon"},
		{name: "identity has a value", key: "openapi:tag:billing", values: []string{"ambiguous"}, wantErr: "must not have a value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, func() {
				API("tags", func() {})
				Service("items", func() {
					Method("list", func() {
						HTTP(func() {
							Meta(test.key, test.values...)
							GET("/items")
						})
					})
				})
			})

			_, err := openapiv3.Files(root)

			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestFilesAcceptsCanonicalColonTagIdentityWithoutValue(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("tags", func() {
			Meta("openapi:tag-id:YmlsbGluZzpkZXNj")
			Meta("openapi:tag-id:YmlsbGluZzpkZXNj:desc", "Billing tag")
		})
		Service("items", func() {
			Method("list", func() {
				HTTP(func() {
					Meta("openapi:tag-id:YmlsbGluZzpkZXNj")
					GET("/items")
				})
			})
		})
	})

	files, err := openapiv3.Files(root)

	require.NoError(t, err)
	require.NotEmpty(t, files)
	require.Equal(t, 2, len(files))
	var jsonPath string
	for _, file := range files {
		if file.Path == filepath.Join("gen", "http", "openapi.json") {
			jsonPath, err = file.Render(t.TempDir())
			require.NoError(t, err)
		}
	}
	require.NotEmpty(t, jsonPath)
	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)
	validateOpenAPI(t, data)
	require.Contains(t, string(data), `"name": "billing:desc"`)
	require.Contains(t, string(data), `"description": "Billing tag"`)
	require.Contains(t, string(data), "\"tags\": [\n          \"billing:desc\"\n        ]")
}

func TestSelectedOutputMarksStaleSiblingForRemoval(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		staleName string
	}{
		{name: "JSON removes YAML", format: "json", staleName: "openapi.yaml"},
		{name: "YAML removes JSON", format: "yaml", staleName: "openapi.json"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selectedFiles, err := openapiv3.Files(openAPIOutputRoot(t, test.format))
			require.NoError(t, err)
			require.Len(t, selectedFiles, 1)
			require.Equal(t, []string{
				filepath.Join(codegen.Gendir, "http", test.staleName),
			}, selectedFiles[0].RemovePaths)
		})
	}
}

func openAPIOutputRoot(t *testing.T, format string) *expr.RootExpr {
	t.Helper()
	root := httpgen.RunHTTPDSL(t, testdata.SimpleDSL)
	if format != "" {
		if root.API.Meta == nil {
			root.API.Meta = make(expr.MetaExpr)
		}
		root.API.Meta["openapi:output"] = []string{format}
	}
	return root
}
