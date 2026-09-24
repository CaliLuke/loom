package openapiv3_test

import (
	"encoding/json/v2"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestRenderedRecursiveNamedArrayExamplesHaveNoNull checks that example
// generation for a named array of objects that reach it through array
// elements and map values succeeds, that the examples hold no null object
// placeholder, which the non-nullable schemas do not admit, and that every
// rendered file is a valid OpenAPI document. The nil-slice placeholder is not
// rendered here; expr TestRecursiveNamedArrayExample covers it.
func TestRenderedRecursiveNamedArrayExamplesHaveNoNull(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, testdata.RecursiveNamedArrayDSL)
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		sections := file.AllSections()
		require.Len(t, sections, 1)
		buf := renderSection(t, sections[0])
		validateOpenAPI(t, buf.Bytes())
		if filepath.Ext(file.Path) != ".json" {
			continue
		}
		var document any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &document))
		grids := requireNoNull(t, document, "$")
		require.Positive(t, grids, "no example of the recursive grid attribute")
	}
}

// requireNoNull fails when value holds a JSON null at any depth, reporting
// its path, and returns the number of "grid" members it holds that are
// arrays of arrays.
func requireNoNull(t *testing.T, value any, path string) int {
	t.Helper()
	require.NotNil(t, value, "null at %s", path)
	grids := 0
	switch actual := value.(type) {
	case map[string]any:
		for key, member := range actual {
			if key == "grid" {
				if list, ok := member.([]any); ok && len(list) > 0 {
					if _, ok := list[0].([]any); ok {
						grids++
					}
				}
			}
			grids += requireNoNull(t, member, path+"."+key)
		}
	case []any:
		for _, item := range actual {
			grids += requireNoNull(t, item, path+"[]")
		}
	}
	return grids
}
