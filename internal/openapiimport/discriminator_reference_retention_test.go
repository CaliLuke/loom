package openapiimport

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

const retainedBlocker = "#/components/schemas/x-loom-unresolved-1"

// referenceRetentionSource links C to ParentHint P as a weak child, an explicit
// mapping target or the defaultMapping target. With raw placement, C reaches U,
// whose redundant closed-union sibling holds the only unresolved reference; with
// normalized placement the unresolved reference survives inside C. /c returns C
// directly, and /ok is unrelated.
func referenceRetentionSource(link, placement string, sibling bool) []byte {
	hint := map[string]string{"weak": "", "explicit": ", mapping: {c: C}", "default": ", defaultMapping: C"}[link]
	child := "u: {$ref: '#/components/schemas/U'}"
	union := `    U:
      oneOf: [{$ref: '#/components/schemas/A'}, {$ref: '#/components/schemas/B'}]
      properties: {x: {$ref: 'missing.yaml#/X'}}
    A: {type: object, properties: {a: {type: string}}, required: [a], additionalProperties: false}
    B: {type: object, properties: {b: {type: string}}, required: [b], additionalProperties: false}
`
	if placement == "normalized" {
		child, union = "u: {$ref: 'missing.yaml#/X'}", ""
	}
	if sibling {
		union += "    C2: {allOf: [{$ref: '#/components/schemas/P'}, {type: object, properties: {other: {type: string}}}]}\n"
	}
	operation := func(path, id, schema string) string {
		return "  " + path + ":\n    get:\n      operationId: " + id + "\n      responses:\n        '200': {description: ok, content: {application/json: {schema: " + schema + "}}}\n"
	}
	return []byte("openapi: 3.2.0\ninfo: {title: Reference retention, version: '1'}\npaths:\n" +
		operation("/p", "p.get", "{$ref: '#/components/schemas/P'}") +
		operation("/c", "c.get", "{$ref: '#/components/schemas/C'}") +
		operation("/ok", "ok.get", "{type: string}") +
		"components:\n  schemas:\n" +
		"    P: {type: object, properties: {kind: {type: string}}, required: [kind], discriminator: {propertyName: kind" + hint + "}}\n" +
		"    C: {allOf: [{$ref: '#/components/schemas/P'}, {type: object, properties: {name: {type: string}, " + child + "}}]}\n" +
		union)
}

// A component retained through a normalized-only discriminator edge keeps its
// raw grammar dependencies. A raw-only blocker disqualifies an optional weak
// child exactly as a normalized one does, leaving valid siblings; through a
// strong explicit or default target it fails closed. Qualification, selected
// and partial outcomes are independent subtests: the raw-path and unselected
// controls have correct selection outcomes even while qualification is wrong.
func TestDiscriminatorReferenceRetention(t *testing.T) {
	const (
		clean      = "clean"
		projection = "projection"
		blocked    = "blocked"
	)
	rawOrigin := &SourceLocation{Document: "entry", Pointer: "#/components/schemas/U/properties/x/$ref", Authored: "missing.yaml#/X"}
	retainedClosure := []string{"A", "B", "C", "P", "U", "x-loom-unresolved-1"}
	for _, tc := range []struct {
		name      string
		link      string
		placement string
		sibling   bool
		paths     []string
		qualified []string
		names     []string
		outcome   string
	}{
		{name: "weak raw-only", link: "weak", placement: "raw", paths: []string{"/p"}, names: []string{"P"}, outcome: projection},
		{name: "weak raw-only valid sibling", link: "weak", placement: "raw", sibling: true, paths: []string{"/p"}, qualified: []string{"C2"}, names: []string{"C2", "P"}, outcome: clean},
		{name: "weak normalized placement", link: "weak", placement: "normalized", paths: []string{"/p"}, names: []string{"P"}, outcome: projection},
		{name: "explicit raw-only", link: "explicit", placement: "raw", paths: []string{"/p"}, names: retainedClosure, outcome: blocked},
		{name: "default raw-only", link: "default", placement: "raw", paths: []string{"/p"}, names: retainedClosure, outcome: blocked},
		{name: "raw path", link: "weak", placement: "raw", paths: []string{"/c"}, names: retainedClosure, outcome: blocked},
		{name: "unselected", link: "weak", placement: "raw", paths: []string{"/ok"}, outcome: clean},
	} {
		source := referenceRetentionSource(tc.link, tc.placement, tc.sibling)
		t.Run(tc.name+"/qualification", func(t *testing.T) {
			full, _, err := Analyze(source)
			require.NoError(t, err)
			var qualified []string
			for _, name := range []string{"C", "C2"} {
				if _, exists := full.parentHintPlans[name]; exists {
					qualified = append(qualified, name)
				}
			}
			require.Equal(t, tc.qualified, qualified, "C's own closure reaches the blocker; only a valid sibling qualifies")
		})
		t.Run(tc.name+"/selected", func(t *testing.T) {
			document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: source}, Selection{Paths: tc.paths})
			require.NoError(t, err)
			names := discriminatorSelectionNames(document)
			slices.Sort(names)
			require.Equal(t, tc.names, names)
			_, renderErr := Render(document, Options{PackageName: "design"})
			switch tc.outcome {
			case clean:
				require.Empty(t, diagnostics)
				require.NoError(t, renderErr)
			case projection:
				require.Len(t, diagnostics, 1)
				require.Equal(t, "schema-discriminator-projection", diagnostics[0].Code)
				require.Equal(t, "#/components/schemas/P/discriminator", diagnostics[0].Path)
			case blocked:
				require.Len(t, diagnostics, 1)
				require.Equal(t, "external-reference", diagnostics[0].Code)
				require.Equal(t, retainedBlocker, diagnostics[0].Path)
				require.Equal(t, rawOrigin, diagnostics[0].Origin)
				require.ErrorContains(t, renderErr, "external-reference")
			}
		})
		t.Run(tc.name+"/partial", func(t *testing.T) {
			selection := Selection{Paths: append(slices.Clone(tc.paths), "/ok")}
			partial, _, err := AnalyzePartial(Source{Bytes: source}, selection, false)
			require.NoError(t, err)
			partialNames := discriminatorSelectionNames(partial.Document)
			slices.Sort(partialNames)
			_, err = Render(partial.Document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.True(t, slices.ContainsFunc(partial.Document.Operations, func(operation Operation) bool {
				return operation.Path == "/ok"
			}))
			switch tc.outcome {
			case clean:
				require.Empty(t, partial.Skipped)
				require.Empty(t, partial.OperationOmissions)
				require.Equal(t, tc.names, partialNames)
			case projection:
				require.Empty(t, partial.Skipped)
				require.Equal(t, tc.names, partialNames)
				require.Len(t, partial.OperationOmissions, 1)
				require.Equal(t, tc.paths[0], partial.OperationOmissions[0].Path)
				require.Len(t, partial.OperationOmissions[0].Diagnostics, 1)
				require.Equal(t, "schema-discriminator-omission", partial.OperationOmissions[0].Diagnostics[0].Code)
				require.Equal(t, "#/components/schemas/P/discriminator", partial.OperationOmissions[0].Diagnostics[0].Path)
			case blocked:
				require.Len(t, partial.Skipped, 1)
				require.Equal(t, tc.paths[0], partial.Skipped[0].Path)
				index := slices.IndexFunc(partial.Skipped[0].Diagnostics, func(diagnostic Diagnostic) bool {
					return diagnostic.Code == "external-reference"
				})
				require.GreaterOrEqual(t, index, 0)
				require.Equal(t, retainedBlocker, partial.Skipped[0].Diagnostics[index].Path)
				require.Equal(t, rawOrigin, partial.Skipped[0].Diagnostics[index].Origin)
				require.Empty(t, partialNames, "only the unrelated operation remains")
			}
		})
	}
}
