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
	// The fixed-seed representative now comes from shared preparation; the
	// selected occurrence still owns its allocated component context.
	require.Contains(t, components, "Body_99e66af2c29e5c06")
	body := components["Body_99e66af2c29e5c06"].(map[string]any)
	require.Equal(t, map[string]any{"a": "Dicta harum."}, body["example"])
	require.Equal(t, "Libero aut et temporibus id officiis.", body["properties"].(map[string]any)["a"].(map[string]any)["example"])
	require.Equal(t, "#/components/schemas/Body_99e66af2c29e5c06", components["body"].(map[string]any)["$ref"])
}
