package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExampleUnresolvedValueState(t *testing.T) {
	const declaration = "#/components/examples/Blocked"
	a := &analyzer{bundle: &bundledDescription{blocked: map[string]Diagnostic{declaration: {Code: "unresolved-reference", Path: declaration}}}}
	placeholder, ok := a.componentExample("Blocked", nil, declaration)
	require.True(t, ok)
	a.examples = map[string]Example{"Blocked": placeholder}
	owner := &Schema{Type: "string"}
	const occurrence = "#/paths/~1bad/get/responses/200/content/application~1json/examples/blocked"
	copy, ok := a.referencedExample(declaration, owner, occurrence)
	require.True(t, ok)
	require.Same(t, owner, copy.source.owner)
	require.Equal(t, occurrence, copy.source.path)
	for _, value := range []Example{placeholder, copy} {
		graph := &exampleGraph{}
		retained := graph.retain([]Example{value}, owner)
		require.Len(t, retained, 1, "an unresolved placeholder is not a decoded null")
		require.Empty(t, graph.diagnostics, "external resolution owns the existing failure")
	}
	for _, tc := range []struct {
		name     string
		value    any
		nullable bool
		kept     bool
	}{
		{name: "decoded null rejected"},
		{name: "nullable null retained", nullable: true, kept: true},
		{name: "decoded string retained", value: "valid", kept: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := &Schema{Type: "string", Nullable: tc.nullable}
			graph := &exampleGraph{}
			retained := graph.retain([]Example{{Value: tc.value, source: exampleSource{owner: schema, path: "#/example"}}}, schema)
			require.Equal(t, tc.kept, len(retained) == 1)
			require.Equal(t, !tc.kept, len(graph.diagnostics) == 1)
		})
	}
}

func TestExampleUnresolvedSelectionProvenance(t *testing.T) {
	document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: blockedExampleSource()}, Selection{Paths: []string{"/bad"}})
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)
	require.Equal(t, "external-reference", diagnostics[0].Code)
	require.NotNil(t, diagnostics[0].Origin)
	require.Equal(t, "missing.yaml#/Example", diagnostics[0].Origin.Authored)
	require.Equal(t, "#/paths/~1bad/get/responses/200/content/application~1json/examples/bad/$ref", diagnostics[0].Origin.Pointer)
	_, err = Render(document, Options{})
	require.ErrorContains(t, err, "external-reference")
}
