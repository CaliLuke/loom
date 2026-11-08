package openapiimport

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Synthetic seam, not public input: with a finalized bundle, a component
// retained through a normalized Ref also retains its raw dependencies,
// transitively across the C/U/X cycle, and the walk terminates. Repeating the
// retention is idempotent. Without a bundle there is no raw source.
func TestComponentPrunerRetainedSource(t *testing.T) {
	components := Components{Schemas: []NamedSchema{
		{Name: "C", Schema: &Schema{Type: "object"}},
		{Name: "U", Schema: &Schema{Type: "object"}},
		{Name: "X", Schema: &Schema{Type: "object"}},
		{Name: "Unused", Schema: &Schema{Type: "object"}},
	}}
	bundle := &bundledDescription{references: map[string]string{
		"#/components/schemas/C/properties/u":      "#/components/schemas/U",
		"#/components/schemas/U/properties/x":      "#/components/schemas/X",
		"#/components/schemas/X/properties/c":      "#/components/schemas/C",
		"#/components/schemas/Unused/properties/x": "#/components/schemas/X",
	}}
	bundle.indexComponentReferences()
	retainedSchemas := func(pruner *componentPruner) []string {
		return slices.Sorted(maps.Keys(pruner.closure.schemas))
	}
	for _, tc := range []struct {
		bundle   *bundledDescription
		retained []string
	}{
		{retained: []string{"C"}},
		{bundle: bundle, retained: []string{"C", "U", "X"}},
	} {
		pruner := newComponentPruner(components)
		pruner.bundle = tc.bundle
		pruner.visitSchema(&Schema{Ref: componentDeclaration("C")})
		require.Equal(t, tc.retained, retainedSchemas(pruner))
		bodies, sources := maps.Clone(pruner.bodies), maps.Clone(pruner.sources)
		for _, name := range tc.retained {
			pruner.visitSchema(&Schema{Ref: componentDeclaration(name)})
		}
		require.Equal(t, tc.retained, retainedSchemas(pruner), "repeating a retention adds nothing")
		require.Equal(t, bodies, pruner.bodies)
		require.Equal(t, sources, pruner.sources)
	}

	// References are never written after bundling. This deliberate late write
	// shows that walks read the finalized index rather than rescanning or
	// rebuilding it from the references.
	bundle.references["#/components/schemas/X/properties/late"] = "#/components/schemas/Unused"
	pruner := newComponentPruner(components)
	pruner.bundle = bundle
	pruner.visitSchema(&Schema{Ref: componentDeclaration("C")})
	require.Equal(t, []string{"C", "U", "X"}, retainedSchemas(pruner))
}

// A bundle from the real resolver owns one finalized component-reference
// index: it agrees with a prefix scan of the references, and every projection
// copy and walk of the analysis shares it unchanged.
func TestBundleComponentReferencesShared(t *testing.T) {
	full, _, err := Analyze(referenceRetentionSource("weak", "raw", true))
	require.NoError(t, err)
	bundle := full.bundle
	require.NotNil(t, bundle)
	index := bundle.componentReferences
	require.NotEmpty(t, index)
	for _, owner := range []string{"#/components/schemas/C", "#/components/schemas/U", "#/components/schemas/C2"} {
		var scanned []string
		for occurrence, target := range bundle.references {
			if occurrence == owner || strings.HasPrefix(occurrence, owner+"/") {
				scanned = append(scanned, target)
			}
		}
		require.NotEmpty(t, scanned, owner)
		require.ElementsMatch(t, scanned, index[owner], owner)
	}
	require.Contains(t, index["#/components/schemas/U"], retainedBlocker)

	snapshot := make(map[string][]string, len(index))
	for owner, targets := range index {
		snapshot[owner] = slices.Clone(targets)
	}
	identity := reflect.ValueOf(index).Pointer()
	for _, paths := range [][]string{{"/p"}, {"/c"}, {"/p", "/c", "/ok"}} {
		var operations []Operation
		for _, operation := range full.Operations {
			if slices.Contains(paths, operation.Path) {
				operations = append(operations, operation)
			}
		}
		candidate := documentForOperations(full, operations)
		pruneComponents(candidate)
		documentSchemaOccurrences(candidate)
		require.Same(t, bundle, candidate.bundle, "projection copies share the analysis bundle")
	}
	require.Equal(t, identity, reflect.ValueOf(bundle.componentReferences).Pointer(), "no walk replaces the index")
	require.Equal(t, snapshot, bundle.componentReferences, "no walk writes the index")
}
