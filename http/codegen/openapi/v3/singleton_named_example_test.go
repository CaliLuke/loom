package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedSingletonExplicitExampleNameCreatesComponent(t *testing.T) {
	for _, version := range []struct {
		target, number string
	}{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(
				t,
				testdata.OpenAPINamedRequestBodyExamplesDSL,
				version.target,
				version.number,
			)
			for _, rendered := range []struct {
				name     string
				document map[string]any
			}{
				{name: "json", document: decodeOpenAPIJSON(t, artifacts.JSON)},
				{name: "yaml", document: decodeOpenAPIExampleYAML(t, artifacts.YAML)},
			} {
				t.Run(rendered.name, func(t *testing.T) {
					namedOperation := requireOperation(t, rendered.document, "/singleton/named", "post")
					namedMedia := requestMediaType(t, rendered.document, namedOperation)
					require.NotContains(t, namedMedia, "example")
					namedExamples := requireMap(t, namedMedia["examples"], "named singleton examples")
					require.Contains(t, namedExamples, "primary")
					namedRef := requireMap(t, namedExamples["primary"], "named singleton example ref")
					require.Equal(t, "#/components/examples/SingletonSearchExample", namedRef["$ref"])

					components := requireMap(t, rendered.document["components"], "components")
					examples := requireMap(t, components["examples"], "component examples")
					named := requireMap(t, examples["SingletonSearchExample"], "named singleton component")
					require.Equal(t, "Authored singleton summary", named["summary"])
					require.Equal(t, "Retained singleton description.", named["description"])
					require.Equal(t, map[string]any{"query": "named soup"}, named["value"])

					inlineOperation := requireOperation(t, rendered.document, "/singleton/inline", "post")
					inlineMedia := requestMediaType(t, rendered.document, inlineOperation)
					require.NotContains(t, inlineMedia, "examples")
					require.Equal(t, map[string]any{"query": "inline soup"}, inlineMedia["example"])
				})
			}
		})
	}
}
