package openapiv3_test

import (
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestReusableReferenceComponentsOpenAPI(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.OpenAPIReferencedComponentsDSL)
	spec := decodeOpenAPIJSON(t, artifacts.JSON)
	components := requireMap(t, spec["components"], "components")
	bodies := requireMap(t, components["requestBodies"], "requestBodies")
	require.Contains(t, bodies, "SearchInput")
	require.Equal(t, "Search filters.", requireMap(t, bodies["SearchFiltersRequest"], "body")["description"])
	responses := requireMap(t, components["responses"], "responses")
	require.Equal(t, "#/components/responses/SearchResults", requireMap(t, responses["Results"], "response")["$ref"])
	require.Equal(t, "Search results.", requireMap(t, responses["SearchResults"], "response")["description"])
	headers := requireMap(t, components["headers"], "headers")
	require.Equal(t, "Trace identifier.", requireMap(t, headers["Trace"], "header")["description"])
	lintEffectiveCompositionArtifacts(t, artifacts)
}
