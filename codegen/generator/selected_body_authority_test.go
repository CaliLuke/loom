package generator

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestSelectedBodyPreservesAllocatedAnnotationOwner(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ResultBodyUserRequiredDSL)
	roots := []eval.Root{root}
	_, err := Service("example.com/valueprobe/gen", roots)
	require.NoError(t, err)
	_, err = Transport("example.com/valueprobe/gen", roots)
	require.NoError(t, err)
	files, err := OpenAPI("example.com/valueprobe/gen", roots)
	require.NoError(t, err)
	var document map[string]any
	for _, file := range files {
		if !strings.HasSuffix(file.Path, "openapi.json") {
			continue
		}
		var data bytes.Buffer
		for _, section := range file.AllSections() {
			require.NoError(t, section.Write(&data))
		}
		require.NoError(t, json.Unmarshal(data.Bytes(), &document))
	}
	require.NotNil(t, document)
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	// Follow the selected occurrence's published binding: canonical acquisition
	// may reuse the unsuffixed name without changing either annotation owner.
	selected := asyncAnnotationMap(t, components, "body")
	ref, ok := selected["$ref"].(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(ref, "#/components/schemas/"))
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	require.NotEqual(t, "body", name, "the occurrence must not resolve to itself")
	body := asyncAnnotationMap(t, components, name)
	require.Equal(t, map[string]any{"a": "Dicta harum."}, body["example"])
	require.Equal(t, "Libero aut et temporibus id officiis.", asyncAnnotationMap(t, asyncAnnotationMap(t, body, "properties"), "a")["example"])
	require.Equal(t, map[string]any{"a": "Qui iusto."}, selected["example"], "the selected occurrence keeps its independent sample")
}
