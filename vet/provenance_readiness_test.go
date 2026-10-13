package vet

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/designfingerprint"
)

// TestGenerationProvenanceAttributesDesignManifests retains two current trees
// and requires inspection of one design to avoid falsely invalidating the other.
func TestGenerationProvenanceAttributesDesignManifests(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/service\n\nrequire github.com/CaliLuke/loom v1.2.3\nreplace github.com/CaliLuke/loom => ../loom\n")
	a := designRootWithHTTPRoute("show", "GET", "/widgets/{id}")
	b := designRootWithHTTPRoute("list", "GET", "/gadgets")
	da, err := designfingerprint.Digest(designfingerprint.Design{Package: "example.com/service/design/a", Roots: []eval.Root{a}}, "gen", "example.com/service/a/gen", codegen.DesignVersion)
	require.NoError(t, err)
	db, err := designfingerprint.Digest(designfingerprint.Design{Package: "example.com/service/design/b", Roots: []eval.Root{b}}, "gen", "example.com/service/b/gen", codegen.DesignVersion)
	require.NoError(t, err)
	writeTestFile(t, filepath.Join(root, "a", "gen", "loom.json"), `{"loom_version":"v1.2.3","design_digest":"`+da+`","design_package":"example.com/service/design/a"}`)
	writeTestFile(t, filepath.Join(root, "b", "gen", "loom.json"), `{"loom_version":"v1.2.3","design_digest":"`+db+`","design_package":"example.com/service/design/b"}`)
	diagnostics, err := analyzeGeneratedManifests(root, &designfingerprint.Design{Package: "example.com/service/design/a", Roots: []eval.Root{a}})
	require.NoError(t, err)
	require.Empty(t, diagnosticKeys(diagnostics))
}

// TestGenerationProvenancePreservesVersionAndSuppression checks attribution
// narrows design skew while version checks retain their module-wide contract.
func TestGenerationProvenancePreservesVersionAndSuppression(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/service\n\nrequire github.com/CaliLuke/loom v1.2.3\n")
	core := designRootWithHTTPRoute("show", "GET", "/widgets/{id}")
	design := &designfingerprint.Design{Package: "example.com/service/design/a", Roots: []eval.Root{core}}
	writeTestFile(t, filepath.Join(root, "b", "gen", "loom.json"), `{"loom_version":"v1.2.2","design_digest":"foreign","design_package":"example.com/service/design/b"}`)
	diagnostics, err := analyzeGeneratedManifests(root, design)
	require.NoError(t, err)
	require.Equal(t, []string{RuleGeneratedVersionSkew + ":b/gen/loom.json"}, diagnosticKeys(diagnostics))
	writeTestFile(t, filepath.Join(root, "a", "gen", "loom.json"), `{"loom_version":"v1.2.3","design_digest":"stale","design_package":"example.com/service/design/a"}`)
	core.API.Meta = expr.MetaExpr{SuppressionMeta: {RuleGeneratedDesignSkew}}
	diagnostics, err = analyzeGeneratedManifests(root, design)
	require.NoError(t, err)
	require.Equal(t, []string{RuleGeneratedVersionSkew + ":b/gen/loom.json"}, diagnosticKeys(diagnostics))
}

// TestGenerationProvenanceRejectsIncompleteInput checks invalid public inputs
// fail before module loading instead of dereferencing a missing core root.
func TestGenerationProvenanceRejectsIncompleteInput(t *testing.T) {
	for _, roots := range [][]eval.Root{nil, {nil}, {&expr.RootExpr{}, &expr.RootExpr{}}} {
		_, err := Analyze(roots, t.TempDir(), "example.com/service/design")
		require.Error(t, err)
	}
	_, err := Analyze([]eval.Root{&expr.RootExpr{}}, t.TempDir(), "")
	require.ErrorContains(t, err, "invalid design package")
}
