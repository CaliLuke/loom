package valuecontract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArtifactComparison rejects every unlisted byte or path change.
func TestArtifactComparison(t *testing.T) {
	for _, tc := range []struct {
		name, path, old, next, kind string
	}{
		{"changed byte", "gen/a.go", "package a\n", "package b\n", "changed"},
		{"added", "gen/a.go", "", "package a\n", "added"},
		{"removed", "gen/a.go", "package a\n", "", "removed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, after := map[string]string{}, map[string]string{}
			if tc.old != "" {
				before[tc.path] = digest([]byte(tc.old))
			}
			if tc.next != "" {
				after[tc.path] = digest([]byte(tc.next))
			}
			differences := compareArtifacts(before, after)
			require.Len(t, differences, 1)
			require.Equal(t, tc.kind, differences[0].Kind)
			require.Equal(t, tc.path, differences[0].Path)
			require.Error(t, checkDifferences("probe", differences, nil))
			allowed := []allowedDifference{{Probe: "probe", Path: tc.path, Before: before[tc.path], After: after[tc.path], Reason: "negative control"}}
			require.NoError(t, checkDifferences("probe", differences, allowed))
			allowed[0].After = "wrong digest"
			require.Error(t, checkDifferences("probe", differences, allowed))
		})
	}
	require.Empty(t, compareArtifacts(map[string]string{"a": "b"}, map[string]string{"a": "b"}))
	require.Error(t, checkDifferences("probe", nil, []allowedDifference{{Probe: "probe", Path: "stale", Reason: "stale"}}))
}

// TestSnapshotPreservesExactBytes retains bytes without module-path normalization.
func TestSnapshotPreservesExactBytes(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(source, "gen"), 0700))
	raw := []byte("package generated\n\x00\xff\n")
	require.NoError(t, os.WriteFile(filepath.Join(source, "gen", "probe.go"), raw, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(source, "go.mod"), []byte("local replacement"), 0600))
	got, err := snapshotArtifacts(source, dest, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"gen/probe.go": digest(raw)}, got)
	saved, err := os.ReadFile(filepath.Join(dest, "gen", "probe.go"))
	require.NoError(t, err)
	require.Equal(t, raw, saved)
}

// TestExpectedFailureRequiresExactPhase keeps setup failures separate from decoder defects.
func TestExpectedFailureRequiresExactPhase(t *testing.T) {
	want := expectation{Phase: "decoder", Contains: "authored value replaced"}
	require.NoError(t, checkOutcome(want, phaseResult{Phase: "decoder", Exit: 1, Output: "authored value replaced"}))
	require.Error(t, checkOutcome(want, phaseResult{Phase: "build", Exit: 1, Output: "authored value replaced"}))
	require.Error(t, checkOutcome(want, phaseResult{Phase: "decoder", Exit: 1, Output: "no space left on device"}))
	require.Error(t, checkOutcome(want, phaseResult{Phase: "decoder", Exit: 1, Output: "found packages valueprobe and svcapi [setup failed]"}))
	require.Error(t, checkOutcome(want, phaseResult{Phase: "decoder", Exit: 1, Output: "authored value replaced", Infrastructure: true}))
	require.Error(t, checkOutcome(want, phaseResult{}))
	require.Error(t, checkOutcome(expectation{}, phaseResult{Phase: "gen", Exit: 1, Output: "failure"}))
}

// TestBaselineIdentity rejects expectations for a different source revision.
func TestBaselineIdentity(t *testing.T) {
	const baseline = "f5b786b39675e2b5c1466f04f3301b7b337779b6"
	require.NoError(t, validateBaseline(baseline, baseline))
	require.ErrorContains(t, validateBaseline(baseline, "ff7873dfff1ad3803c421e1e64978906ee1139ce"), "does not match")
	require.ErrorContains(t, validateBaseline(baseline, "/local/dirty/checkout"), "full published commit")
	require.ErrorContains(t, validateBaseline(baseline, "main"), "full published commit")
}

// TestHandwrittenInputMutation rejects unexpected generator writes to fixture inputs.
func TestHandwrittenInputMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "service.go")
	original := []byte("package fixture\n")
	require.NoError(t, os.WriteFile(path, original, 0600))
	inputs := map[string]string{"service.go": digest(original)}
	require.NoError(t, checkInputsUnchanged(dir, inputs))
	require.NoError(t, os.WriteFile(path, []byte("package changed\n"), 0600))
	require.ErrorContains(t, checkInputsUnchanged(dir, inputs), "changed handwritten input")
}
