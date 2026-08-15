package openapiv3_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/http/codegen/openapi"
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
	openapi.Definitions = make(map[string]*openapi.Schema)
	root := httpgen.RunHTTPDSL(t, testdata.SimpleDSL)
	if format != "" {
		if root.API.Meta == nil {
			root.API.Meta = make(expr.MetaExpr)
		}
		root.API.Meta["openapi:output"] = []string{format}
	}
	return root
}
