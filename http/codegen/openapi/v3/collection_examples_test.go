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

func TestRenderedCollectionExamples(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, testdata.CollectionExamplesDSL)
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
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
		arrays, maps := requireCollectionExamples(t, document)
		require.Positive(t, arrays, "no labels example rendered")
		require.Positive(t, maps, "no children example rendered")
	}
}

func requireCollectionExamples(t *testing.T, value any) (arrays, maps int) {
	t.Helper()
	switch actual := value.(type) {
	case map[string]any:
		for key, member := range actual {
			if labels, ok := member.([]any); key == "labels" && ok {
				arrays++
				require.NotEmpty(t, labels)
				for _, label := range labels {
					require.Equal(t, "", label)
				}
			}
			if children, ok := member.(map[string]any); key == "children" && ok {
				// Schema properties also use this name; only inspect examples.
				if _, schema := children["type"]; !schema {
					maps++
					require.NotEmpty(t, children)
					for _, child := range children {
						require.IsType(t, map[string]any{}, child)
						require.NotNil(t, child)
					}
				}
			}
			a, m := requireCollectionExamples(t, member)
			arrays += a
			maps += m
		}
	case []any:
		for _, member := range actual {
			a, m := requireCollectionExamples(t, member)
			arrays += a
			maps += m
		}
	}
	return arrays, maps
}
