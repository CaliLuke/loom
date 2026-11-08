package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockedOccurrenceSelection(t *testing.T) {
	t.Run("placeholder", func(t *testing.T) {
		document, diagnostics, err := Analyze(resolverEntry("", "missing.yaml#/X"))
		require.NoError(t, err)
		require.Len(t, diagnostics, 1)
		require.Equal(t, "external-reference", diagnostics[0].Code)
		require.Equal(t, "#/components/schemas/x-loom-unresolved-1", diagnostics[0].Path)
		require.Equal(t, &SourceLocation{Document: "entry", Pointer: "#/components/schemas/R0/$ref", Authored: "missing.yaml#/X"}, diagnostics[0].Origin)
		rendered, err := Render(document, Options{})
		require.Error(t, err)
		require.Empty(t, rendered)
	})
	t.Run("originFilter", func(t *testing.T) {
		source := resolverFiles(t, []byte(`openapi: 3.2.0
info: {title: Origin, version: '1'}
paths:
  /pets:
    get:
      tags: [pets]
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {$ref: 'models.yaml#/X'}
components:
  schemas:
    X: {type: string}
`), map[string]string{"models.yaml": "X: {type: object, properties: {child: {$ref: 'missing.yaml#/Child'}}}"})
		_, diagnostics, _, err := AnalyzeSelected(source, Selection{Tags: []string{"pets"}})
		require.NoError(t, err)
		require.Len(t, diagnostics, 1)
		require.Equal(t, "external-reference", diagnostics[0].Code)
		require.Equal(t, "#/components/schemas/x-loom-unresolved-1", diagnostics[0].Path)
		require.Equal(t, "root:models.yaml", diagnostics[0].Origin.Document)
		require.Equal(t, "#/X/properties/child/$ref", diagnostics[0].Origin.Pointer)
	})
	for _, tc := range []struct {
		name        string
		selection   Selection
		wantBlocked bool
	}{
		{"assumeUnselected", Selection{Tags: []string{"good"}}, true},
		{"unmatchedPath", Selection{PathPrefixes: []string{"/good"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := Source{Bytes: []byte(`openapi: 3.2.0
info: {title: Unknown paths, version: '1'}
paths:
  /good:
    get: {tags: [good], responses: {'204': {description: OK}}}
  /unknown: {$ref: 'missing.yaml'}
`)}
			document, diagnostics, report, err := AnalyzeSelected(source, tc.selection)
			require.NoError(t, err)
			if tc.wantBlocked {
				require.Len(t, diagnostics, 1)
				require.Equal(t, "path-reference", diagnostics[0].Code)
				require.Equal(t, "#/paths/~1unknown", diagnostics[0].Path)
				require.NotNil(t, diagnostics[0].Origin)
				require.Equal(t, []string{"/unknown"}, report.UnresolvedPaths)
				require.Empty(t, report.UnclaimedPaths, "unknown tags are not evidence that a path is unclaimed")
				_, err = Render(document, Options{})
				require.ErrorContains(t, err, "path-reference")
				partial, _, err := AnalyzePartial(source, tc.selection, false)
				require.NoError(t, err)
				require.Len(t, partial.Document.Operations, 1)
				require.Len(t, partial.Skipped, 1)
				require.Equal(t, "/unknown", partial.Skipped[0].Path)
			} else {
				require.Empty(t, diagnostics)
				require.Empty(t, report.UnresolvedPaths)
			}
		})
	}
	t.Run("fiveKinds", func(t *testing.T) {
		analysis, _, err := AnalyzePartial(Source{Bytes: blockedExampleSource()}, Selection{}, true)
		require.NoError(t, err)
		require.Len(t, analysis.Document.Operations, 1)
		require.Equal(t, "/good", analysis.Document.Operations[0].Path)
		require.Len(t, analysis.Skipped, 1)
		require.Equal(t, "/bad", analysis.Skipped[0].Path)
		requireDiagnosticCode(t, analysis.Skipped[0].Diagnostics, "external-reference")
	})
	t.Run("unfilteredExamples", func(t *testing.T) {
		_, diagnostics, _, err := AnalyzeSelected(Source{Bytes: blockedExampleSource()}, Selection{PathPrefixes: []string{"/good"}})
		require.NoError(t, err)
		require.Empty(t, diagnostics)
	})
}

func TestBlockedSecurityKey(t *testing.T) {
	source := Source{Bytes: []byte(`openapi: 3.2.0
info: {title: Security, version: '1'}
paths:
  /pets:
    get:
      security:
        - 'https://e.test/missing#/components/securitySchemes/Auth': []
      responses: {'204': {description: OK}}
`)}
	document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)
	require.Equal(t, "external-reference", diagnostics[0].Code)
	require.Equal(t, "#/paths/~1pets/get/security/0/https:~1~1e.test~1missing#~1components~1securitySchemes~1Auth", diagnostics[0].Path)
	require.NotNil(t, diagnostics[0].Origin)
	_, err = Render(document, Options{})
	require.ErrorContains(t, err, "external-reference")
}

func TestBlockedClosureKinds(t *testing.T) {
	document := &Document{
		SecurityDefined: true,
		Security:        SecurityRequirements{{Schemes: []SecurityRequirementScheme{{Name: "Auth"}}}},
		Operations: []Operation{{
			Method: "GET", Path: "/pets", GoName: "Pets",
			Responses: []StatusResponse{{Status: "200", Response: Response{
				Description: "OK", ContentType: "application/json", Schema: &Schema{Type: "string"},
				Examples: []Example{{ComponentName: "PetExample", Value: "pet"}},
			}}},
		}},
	}
	closure := pruneComponents(document)
	require.Contains(t, closure.examples, "PetExample")
	require.Contains(t, closure.securitySchemes, "Auth")
}

func blockedExampleSource() []byte {
	return []byte(`openapi: 3.2.0
info: {title: Examples, version: '1'}
paths:
  /good:
    get: {responses: {'204': {description: OK}}}
  /bad:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {type: string}
              examples:
                bad: {$ref: 'missing.yaml#/Example'}
`)
}
