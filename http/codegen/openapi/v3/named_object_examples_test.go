package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestNamedObjectExamplesRetainValues(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.NamedObjectExamplesDSL)
	spec := decodeOpenAPIJSON(t, artifacts.JSON)
	example := requireComponentSchema(t, spec, "ExampleAddresses")["example"]
	require.Equal(t, map[string]any{
		"direct":  map[string]any{"street": "Direct Street"},
		"alias":   map[string]any{"street": "Alias Street"},
		"list":    []any{map[string]any{"street": "List Street"}},
		"by_name": map[string]any{"home": map[string]any{"street": "Map Street"}},
	}, example)
}
