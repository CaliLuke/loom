package generator

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
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
	// Literal parent outputs for this RunDSL seed; the selected occurrence owns
	// its allocated context even when its declaration and shape match Body.
	require.Contains(t, components, "Body_99e66af2c29e5c06")
	body := components["Body_99e66af2c29e5c06"].(map[string]any)
	require.Equal(t, map[string]any{"a": "Magni ad cum."}, body["example"])
	require.Equal(t, "Ea id quo eum ea aut vitae.", body["properties"].(map[string]any)["a"].(map[string]any)["example"])
	require.Equal(t, "#/components/schemas/Body_99e66af2c29e5c06", components["body"].(map[string]any)["$ref"])
}
