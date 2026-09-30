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

func TestResponsePublicIdentityPreservesReferenceAnnotations(t *testing.T) {
	root := codegen.RunDSL(t, testdata.UnionResponseBodyDSL)
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
	components := document["components"].(map[string]any)["responses"].(map[string]any)
	// Literal parent-generated identity; do not derive this oracle from current hashes.
	require.Contains(t, components, "BadRequestError_fd4a3924")
	badRequests := 0
	for name := range components {
		if strings.HasPrefix(name, "BadRequestError") {
			badRequests++
		}
	}
	require.Equal(t, 2, badRequests)
	paths := document["paths"].(map[string]any)
	for _, path := range []string{"/required_nullable", "/optional_nullable", "/required_named_nullable", "/optional_named_nullable"} {
		response := paths[path].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)["400"].(map[string]any)
		require.Equal(t, "#/components/responses/BadRequestError_fd4a3924", response["$ref"])
	}
}
