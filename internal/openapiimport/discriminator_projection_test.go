package openapiimport

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func discriminatorTransitiveParentSource() []byte {
	return []byte(`openapi: 3.2.0
info: {title: Transitive weak child, version: '1'}
paths:
  /p:
    get:
      operationId: p.get
      responses:
        '200':
          description: parent
          content:
            application/json:
              schema: {$ref: '#/components/schemas/P'}
components:
  schemas:
    P:
      type: object
      properties: {kind: {type: string}}
      required: [kind]
      discriminator: {propertyName: kind}
    C1:
      allOf:
        - {$ref: '#/components/schemas/P'}
        - {type: object, properties: {q: {$ref: '#/components/schemas/Q'}}}
    C2:
      allOf:
        - {$ref: '#/components/schemas/P'}
        - {type: object, properties: {name: {type: string}}}
    Q:
      type: object
      properties: {kind: {type: string}}
      required: [kind]
      discriminator: {propertyName: kind}
    QBroken:
      allOf:
        - {$ref: '#/components/schemas/Q'}
        - {type: object, properties: {a: {type: string}}}
        - {type: object, properties: {b: {type: string}}}
`)
}

func TestDiscriminatorTransitiveChildQualification(t *testing.T) {
	for _, tc := range []struct {
		name       string
		change     func(string) string
		retained   []string
		projection bool
	}{
		{name: "unsupported descendant", retained: []string{"P", "C2"}},
		{name: "no dependency", change: func(source string) string {
			return strings.Replace(source, "q: {$ref: '#/components/schemas/Q'}", "q: {type: string}", 1)
		}, retained: []string{"P", "C1", "C2"}},
		{name: "supported descendant", change: func(source string) string {
			return strings.Replace(source, "        - {type: object, properties: {b: {type: string}}}\n", "", 1)
		}, retained: []string{"P", "C1", "C2", "Q", "QBroken"}},
		{name: "explicit stays strong", change: func(source string) string {
			return strings.Replace(source, "discriminator: {propertyName: kind}", "discriminator: {propertyName: kind, mapping: {child: C1}}", 1)
		}, retained: []string{"P", "C1", "C2", "Q"}, projection: true},
		{name: "default stays strong", change: func(source string) string {
			return strings.Replace(source, "discriminator: {propertyName: kind}", "discriminator: {propertyName: kind, defaultMapping: C1}", 1)
		}, retained: []string{"P", "C1", "C2", "Q"}, projection: true},
		{name: "owned inline witness", change: func(source string) string {
			return strings.Replace(source, "q: {$ref: '#/components/schemas/Q'}", "q: {allOf: [{$ref: '#/components/schemas/Q'}, {type: object, properties: {name: {type: string}}}]}", 1)
		}, retained: []string{"P", "C1", "C2", "Q"}},
		{name: "unsupported seed propagates", change: func(source string) string {
			source = strings.Replace(source, "        - {type: object, properties: {b: {type: string}}}\n", "", 1)
			source = strings.Replace(source, "a: {type: string}", "a: {$ref: '#/components/schemas/R'}", 1)
			return source + `    R:
      type: object
      properties: {kind: {type: string}}
      required: [kind]
      discriminator: {propertyName: kind}
    RBroken:
      allOf:
        - {$ref: '#/components/schemas/R'}
        - {type: object, properties: {a: {type: string}}}
        - {type: object, properties: {b: {type: string}}}
`
		}, retained: []string{"P", "C2"}},
		{name: "productive cycle", change: func(source string) string {
			source = strings.Replace(source, "        - {type: object, properties: {b: {type: string}}}\n", "", 1)
			return strings.Replace(source, "a: {type: string}", "a: {$ref: '#/components/schemas/P'}", 1)
		}, retained: []string{"P", "C1", "C2", "Q", "QBroken"}},
		{name: "cycle without independent sibling", change: func(source string) string {
			source = strings.Replace(source, "        - {type: object, properties: {b: {type: string}}}\n", "", 1)
			source = strings.Replace(source, "a: {type: string}", "a: {$ref: '#/components/schemas/P'}", 1)
			return strings.Replace(source, `    C2:
      allOf:
        - {$ref: '#/components/schemas/P'}
        - {type: object, properties: {name: {type: string}}}
`, "", 1)
		}, retained: []string{"P", "C1", "Q", "QBroken"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := string(discriminatorTransitiveParentSource())
			if tc.change != nil {
				source = tc.change(source)
			}
			for _, reverse := range []bool{false, true} {
				input := source
				if reverse {
					header, schemas, found := strings.Cut(input, "  schemas:\n")
					require.True(t, found)
					var declarations []string
					for _, line := range strings.Split(schemas, "\n") {
						if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     ") {
							declarations = append(declarations, line+"\n")
						} else if len(declarations) > 0 && line != "" {
							declarations[len(declarations)-1] += line + "\n"
						}
					}
					slices.Reverse(declarations)
					input = header + "  schemas:\n" + strings.Join(declarations, "")
				}
				selection := Selection{Paths: []string{"/p"}}
				document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: []byte(input)}, selection)
				require.NoError(t, err)
				require.ElementsMatch(t, tc.retained, discriminatorSelectionNames(document), "reverse=%t", reverse)
				if tc.projection {
					requireDiagnosticCode(t, diagnostics, "schema-discriminator-projection")
				} else {
					require.Empty(t, diagnostics)
				}
				partial, _, err := AnalyzePartial(Source{Bytes: []byte(input)}, selection, false)
				require.NoError(t, err)
				require.Empty(t, partial.Skipped)
				require.ElementsMatch(t, tc.retained, discriminatorSelectionNames(partial.Document))
				if tc.projection {
					require.Len(t, partial.OperationOmissions, 1)
					requireDiagnosticCode(t, partial.OperationOmissions[0].Diagnostics, "schema-discriminator-omission")
				} else {
					require.Empty(t, partial.OperationOmissions)
				}
			}
		})
	}
}

func TestDiscriminatorProjectionKeepsPlannerDiagnostics(t *testing.T) {
	document, diagnostics, err := Analyze(discriminatorParentSelectionSource(true, false, ""))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	document.Components.Schemas = slices.DeleteFunc(document.Components.Schemas, func(named NamedSchema) bool {
		return named.Name != "Parent"
	})
	document.OpenAPIVersion = "unsupported"
	plan := planDocument(document)
	requireDiagnosticCode(t, plan.diagnostics, "openapi-version")
	requireDiagnosticCode(t, plan.diagnostics, "schema-discriminator-projection")
}

func TestDiscriminatorSharedInlineWitness(t *testing.T) {
	source := string(discriminatorTransitiveParentSource())
	source = strings.Replace(source, "name: {type: string}", "inline: {allOf: [{$ref: '#/components/schemas/Q'}, {type: object, properties: {extra: {type: string}}}]}", 1)
	source, _, _ = strings.Cut(source, "    QBroken:\n")
	for _, sibling := range []bool{true, false} {
		t.Run(fmt.Sprintf("sibling=%t", sibling), func(t *testing.T) {
			input := source
			if !sibling {
				input = strings.Replace(input, "    C2:\n      allOf:\n        - {$ref: '#/components/schemas/P'}\n", "    C2:\n      allOf:\n", 1)
			}
			document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: []byte(input)}, Selection{Paths: []string{"/p"}})
			require.NoError(t, err)
			if sibling {
				require.Empty(t, diagnostics)
				require.ElementsMatch(t, []string{"P", "C1", "C2", "Q"}, discriminatorSelectionNames(document))
			} else {
				requireDiagnosticCode(t, diagnostics, "schema-discriminator-projection")
				require.Equal(t, []string{"P"}, discriminatorSelectionNames(document))
			}
		})
	}
}

func discriminatorOperationInlineSource() string {
	source := string(discriminatorTransitiveParentSource())
	source, _, _ = strings.Cut(source, "    QBroken:\n")
	source = strings.Replace(source, "components:\n", `  /witness:
    get:
      operationId: witness.get
      responses:
        '200':
          description: inline witness
          content:
            application/json:
              schema:
                allOf:
                  - {$ref: '#/components/schemas/Q'}
                  - {type: object, properties: {name: {type: string}}}
components:
`, 1)
	return source
}

func TestDiscriminatorSelectedOperationInlineWitness(t *testing.T) {
	source := discriminatorOperationInlineSource()
	for _, paths := range [][]string{{"/p"}, {"/p", "/witness"}, {"/p"}, {"/p", "/witness"}} {
		document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: []byte(source)}, Selection{Paths: paths})
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		expected := []string{"P", "C2"}
		if len(paths) == 2 {
			expected = []string{"P", "C1", "C2", "Q"}
		}
		require.ElementsMatch(t, expected, discriminatorSelectionNames(document))
		partial, _, err := AnalyzePartial(Source{Bytes: []byte(source)}, Selection{Paths: paths}, false)
		require.NoError(t, err)
		require.Empty(t, partial.Skipped)
		require.Empty(t, partial.OperationOmissions)
		require.ElementsMatch(t, expected, discriminatorSelectionNames(partial.Document))
	}
	full, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.ElementsMatch(t, []string{"P", "C1", "C2", "Q"}, discriminatorSelectionNames(full))
}

func TestDiscriminatorProjectionPlansAreLocal(t *testing.T) {
	source := discriminatorOperationInlineSource()
	full, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	original, err := Render(full, Options{PackageName: "design"})
	require.NoError(t, err)
	for _, includeWitness := range []bool{false, true, false, true} {
		var operations []Operation
		for _, operation := range full.Operations {
			if includeWitness || operation.Path == "/p" {
				operations = append(operations, operation)
			}
		}
		candidate := documentForOperations(full, operations)
		pruneComponents(candidate)
		require.Empty(t, planDocument(candidate).diagnostics)
		if includeWitness {
			require.ElementsMatch(t, []string{"P", "C1", "C2", "Q"}, discriminatorSelectionNames(candidate))
		} else {
			require.ElementsMatch(t, []string{"P", "C2"}, discriminatorSelectionNames(candidate))
		}
		require.Len(t, full.parentHintPlans, 2, "base-qualified plans remain shared and unchanged")
		after, err := Render(full, Options{PackageName: "design"})
		require.NoError(t, err)
		require.Equal(t, original, after)
	}
	// A witness belonging to an omitted operation cannot support the final graph.
	rejected := strings.Replace(source, "                  - {type: object, properties: {name: {type: string}}}", "                  - {type: object, properties: {name: {type: string}}}\n                  - {type: object, properties: {extra: {type: string}}}", 1)
	partial, _, err := AnalyzePartial(Source{Bytes: []byte(rejected)}, Selection{}, false)
	require.NoError(t, err)
	require.Len(t, partial.Skipped, 1)
	require.Equal(t, "/witness", partial.Skipped[0].Path)
	require.Empty(t, partial.OperationOmissions)
	require.ElementsMatch(t, []string{"P", "C2"}, discriminatorSelectionNames(partial.Document))
}

// A root-owned obligation is one the selected strong graph already emits. A weak
// child that reaches no other unwitnessed hint is retained; the hint's own
// projection failure is reported once, and introduced obligations still reject.
func TestDiscriminatorRootOwnedObligation(t *testing.T) {
	const (
		qHint = "#/components/schemas/Q/discriminator"
		pHint = "#/components/schemas/P/discriminator"
	)
	strongQ := `  /q:
    get:
      operationId: q.get
      responses:
        '200':
          description: strong Q root
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Q'}
components:
`
	for _, tc := range []struct {
		name     string
		change   func(string) string
		paths    []string
		retained []string
		// hint is the single strict projection path and partial omission path.
		hint    string
		omitter string
	}{
		{name: "explicit target", change: func(source string) string {
			return strings.Replace(source, "discriminator: {propertyName: kind}", "discriminator: {propertyName: kind, mapping: {child: C1}}", 1)
		}, paths: []string{"/p"}, retained: []string{"P", "C1", "C2", "Q"}, hint: qHint, omitter: "/p"},
		{name: "default target", change: func(source string) string {
			return strings.Replace(source, "discriminator: {propertyName: kind}", "discriminator: {propertyName: kind, defaultMapping: C1}", 1)
		}, paths: []string{"/p"}, retained: []string{"P", "C1", "C2", "Q"}, hint: qHint, omitter: "/p"},
		{name: "selected root", change: func(source string) string {
			return strings.Replace(source, "components:\n", strongQ, 1)
		}, paths: []string{"/p", "/q"}, retained: []string{"P", "C1", "C2", "Q"}, hint: qHint, omitter: "/q"},
		{name: "unselected root", change: func(source string) string {
			return strings.Replace(source, "components:\n", strongQ, 1)
		}, paths: []string{"/p"}, retained: []string{"P", "C2"}},
		{name: "mutually introduced", change: func(source string) string {
			return strings.Replace(source, "name: {type: string}", "other: {$ref: '#/components/schemas/Q'}", 1)
		}, paths: []string{"/p"}, retained: []string{"P"}, hint: pHint, omitter: "/p"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(tc.change(string(discriminatorTransitiveParentSource())))
			selection := Selection{Paths: tc.paths}
			document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: source}, selection)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.retained, discriminatorSelectionNames(document))
			partial, _, err := AnalyzePartial(Source{Bytes: source}, selection, false)
			require.NoError(t, err)
			require.Empty(t, partial.Skipped)
			require.Equal(t, len(tc.paths), len(partial.Document.Operations))
			require.ElementsMatch(t, tc.retained, discriminatorSelectionNames(partial.Document))
			_, err = Render(partial.Document, Options{PackageName: "design"})
			require.NoError(t, err)
			if tc.hint == "" {
				require.Empty(t, diagnostics)
				require.Empty(t, partial.OperationOmissions)
				return
			}
			require.Len(t, diagnostics, 1)
			require.Equal(t, "schema-discriminator-projection", diagnostics[0].Code)
			require.Equal(t, tc.hint, diagnostics[0].Path)
			require.Len(t, partial.OperationOmissions, 1)
			require.Equal(t, tc.omitter, partial.OperationOmissions[0].Path)
			require.Len(t, partial.OperationOmissions[0].Diagnostics, 1)
			require.Equal(t, "schema-discriminator-omission", partial.OperationOmissions[0].Diagnostics[0].Code)
			require.Equal(t, tc.hint, partial.OperationOmissions[0].Diagnostics[0].Path)
			for _, named := range partial.Document.Components.Schemas {
				omitted := "#/components/schemas/"+named.Name+"/discriminator" == tc.hint
				if named.Name == "P" || named.Name == "Q" {
					require.Equal(t, omitted, named.Schema.Discriminator == nil, named.Name)
				}
			}
		})
	}
}

// Every projection walks the selected roots once per fixed-point round, not once
// per candidate. Durations are logged as evidence; asserting them would be flaky.
func TestDiscriminatorProjectionScaling(t *testing.T) {
	for _, size := range []int{20, 60, 120} {
		t.Run(fmt.Sprintf("children=%d", size), func(t *testing.T) {
			var paths, schemas strings.Builder
			for index := range size {
				fmt.Fprintf(&paths, "  /op%d:\n    get:\n      operationId: op%d.get\n      responses:\n        '200':\n          description: op\n          content:\n            application/json:\n              schema: {$ref: '#/components/schemas/V%d'}\n", index, index, index)
				fmt.Fprintf(&schemas, "    V%d:\n      type: object\n      properties: {p: {$ref: '#/components/schemas/P'}}\n", index)
				fmt.Fprintf(&schemas, "    C%d:\n      allOf:\n        - {$ref: '#/components/schemas/P'}\n        - {type: object, properties: {name: {type: string}}}\n", index)
			}
			source := []byte("openapi: 3.2.0\ninfo: {title: Projection scaling, version: '1'}\npaths:\n" + paths.String() +
				"components:\n  schemas:\n    P:\n      type: object\n      properties: {kind: {type: string}}\n      required: [kind]\n      discriminator: {propertyName: kind}\n" + schemas.String())
			start := time.Now()
			partial, _, err := AnalyzePartial(Source{Bytes: source}, Selection{}, false)
			elapsed := time.Since(start)
			require.NoError(t, err)
			require.Empty(t, partial.Skipped)
			require.Empty(t, partial.OperationOmissions)
			require.Len(t, partial.Document.Operations, size)
			require.Len(t, partial.Document.Components.Schemas, 2*size+1)
			t.Logf("children=%d operations=%d partial=%s", size, size, elapsed)
		})
	}
}
