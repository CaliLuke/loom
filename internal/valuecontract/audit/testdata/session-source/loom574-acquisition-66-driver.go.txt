package valuecontract

import (
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const repairOriginal = "/tmp/loom574-comparison-discovery"
const repairResults = "/tmp/loom574-acquisition-66"
const repairAcceptedFinalContentID = "43893ba9771f5447b9b3e5332703e3f80e6b27aa4d1812d03be1691028837a92"
const repairKnownCompilerSHA = "13482562680f9942f5524499b3952fe4f9c7e357de2a57b7543322839b1f8476"

func repairKnownFailure() expectation {
	return expectation{Phase: "build", Contains: "cannot use &body (value of type *StaticJSONResponseBody) as StaticJSONResponseBody value in argument to ValidateStaticJSONResponseBody", Issue: "https://github.com/CaliLuke/loom/issues/565"}
}

func repairManifestCompatible(original, current []probe) error {
	if len(original) != 66 || len(current) != 66 {
		return fmt.Errorf("expected exactly66 probes")
	}
	expected := append([]probe(nil), original...)
	count := 0
	for index, p := range expected {
		if p.ID != "byte-representation-ownership" {
			continue
		}
		count++
		if p.Baseline != (expectation{}) || p.Candidate != (expectation{}) {
			return fmt.Errorf("original known-failure expectation changed")
		}
		expected[index].Baseline = repairKnownFailure()
		expected[index].Candidate = repairKnownFailure()
	}
	if count != 1 || !reflect.DeepEqual(expected, current) {
		return fmt.Errorf("unreviewed probe or expectation change")
	}
	return nil
}

func repairRead(t *testing.T, path string, result any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, result, json.RejectUnknownMembers(true)))
}

func repairCommon(t *testing.T) string {
	t.Helper()
	common := filepath.Join(repairOriginal, "common-source")
	var files map[string]sourceFile
	repairRead(t, common+"-files.json", &files)
	require.NoError(t, verifySourceSnapshot(common, files))
	identity, err := snapshotIdentity(files)
	require.NoError(t, err)
	var metadata struct {
		OriginalSource string   `json:"original_source"`
		ContentID      string   `json:"content_id"`
		Excluded       []string `json:"excluded"`
	}
	repairRead(t, filepath.Join(repairOriginal, "common-source.json"), &metadata)
	require.Equal(t, "02cdce4df7893ccb0c34ef8c7a5b9f4ee6df16c020e310ea8ed52ecd0c2e0cca", identity)
	require.Equal(t, identity, metadata.ContentID)
	var candidate revision
	repairRead(t, filepath.Join(repairOriginal, "candidate-source.json"), &candidate)
	require.Equal(t, identity, candidate.ContentID)
	require.Equal(t, "2f6bfbf7758f4de56eb14569a49a645355ca228f", candidate.Commit)
	require.Equal(t, "/Users/luca/.codex/worktrees/body-type-names/loom", candidate.Input)
	require.Equal(t, common, candidate.Source)
	return common
}

// TestRepairCandidateGeneration is generation-only. Final acceptance requires
// TestRepairRetainedComparison with reviewed exact differences after discovery.
func TestRepairCandidateGeneration(t *testing.T) {
	common := repairCommon(t)
	manifest := readManifest(t)
	var original comparisonManifest
	repairRead(t, filepath.Join(repairOriginal, "selected-manifest.json"), &original)
	require.Equal(t, original.Baseline, manifest.Baseline)
	require.NoError(t, repairManifestCompatible(original.Probes, manifest.Probes))
	require.NoError(t, os.Mkdir(repairResults, 0700), "never merge distinct candidate runs")
	// Reuse the independently approved immutable capture; never reread live source.
	var candidate revision
	repairRead(t, "/tmp/loom574-acquisition-targeted/candidate-source.json", &candidate)
	require.Equal(t, repairAcceptedFinalContentID, candidate.ContentID)
	require.Equal(t, "/tmp/loom574-acquisition-targeted/common-source", candidate.Source)
	require.Equal(t, "2f6bfbf7758f4de56eb14569a49a645355ca228f", candidate.Commit)
	var files map[string]sourceFile
	repairRead(t, candidate.Source+"-files.json", &files)
	require.NoError(t, verifySourceSnapshot(candidate.Source, files))
	identity, err := snapshotIdentity(files)
	require.NoError(t, err)
	require.Equal(t, repairAcceptedFinalContentID, identity)
	approvedSource := candidate.Source
	candidate.Source = filepath.Join(repairResults, "candidate-source")
	require.NoError(t, materializeSource(approvedSource, candidate.Source, files))
	writeJSON(t, candidate.Source+"-files.json", files)
	t.Cleanup(func() {
		require.NoError(t, verifySourceSnapshot(approvedSource, files))
		require.NoError(t, verifySourceSnapshot(candidate.Source, files))
		repairCommon(t)
	})
	writeJSON(t, filepath.Join(repairResults, "candidate-source.json"), candidate)
	require.NoError(t, os.WriteFile(filepath.Join(repairResults, "selected-manifest.json"), manifest.raw, 0600))
	candidate.Binary = filepath.Join(t.TempDir(), "loom")
	built := runPhase(t.Context(), candidate.Source, nil, "build-generator", "go", "build", "-o", candidate.Binary, "./cmd/loom")
	writeJSON(t, filepath.Join(repairResults, "candidate-generator.json"), built)
	require.Zero(t, built.Exit, "generator infrastructure: %s", built.Output)
	count := 0
	for _, p := range manifest.Probes {
		count++
		t.Run(p.ID, func(t *testing.T) {
			var repeated [2]probeResult
			for attempt := range 2 {
				location := filepath.Join(repairResults, "candidate", p.ID, fmt.Sprintf("run-%d", attempt+1))
				repeated[attempt] = runProbe(t, candidate, p, location, common)
				require.NoError(t, checkOutcome(p.Candidate, firstFailure(repeated[attempt])))
			}
			repairPair(t, repairResults, "candidate", p, p.Candidate)
			require.Equal(t, repeated[0].Inputs, repeated[1].Inputs)
			require.Empty(t, compareArtifacts(repeated[0].Artifacts, repeated[1].Artifacts))
			require.Equal(t, string(repeated[0].Observation), string(repeated[1].Observation))
		})
	}
	require.Equal(t, 66, count)
}

func repairTreeHashes(t *testing.T, root string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return result
	}
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("nonregular retained artifact %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = digest(raw)
		return nil
	}))
	return result
}

func repairPair(t *testing.T, resultRoot, label string, p probe, expected expectation) probeResult {
	t.Helper()
	var pair [2]probeResult
	for attempt := range 2 {
		location := filepath.Join(resultRoot, label, p.ID, fmt.Sprintf("run-%d", attempt+1))
		repairRead(t, filepath.Join(location, "result.json"), &pair[attempt])
		require.Equal(t, p.ID, pair[attempt].ID)
		logs := make(map[string]string)
		entries, err := os.ReadDir(location)
		require.NoError(t, err)
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(location, entry.Name()))
			require.NoError(t, err)
			logs[strings.TrimSuffix(entry.Name(), ".log")] = string(raw)
		}
		require.NoError(t, repairValidateRecord(pair[attempt], p, expected, logs))
		require.Equal(t, pair[attempt].Inputs, repairTreeHashes(t, filepath.Join(location, "inputs")))
		require.Equal(t, pair[attempt].Artifacts, repairTreeHashes(t, filepath.Join(location, "artifacts")))
	}
	require.Equal(t, pair[0].Inputs, pair[1].Inputs)
	require.Empty(t, compareArtifacts(pair[0].Artifacts, pair[1].Artifacts))
	require.Equal(t, string(pair[0].Observation), string(pair[1].Observation))
	return pair[0]
}

func TestRepairRetainedComparison(t *testing.T) {
	repairCommon(t)
	manifest := readManifest(t)
	var original, regenerated comparisonManifest
	repairRead(t, filepath.Join(repairOriginal, "selected-manifest.json"), &original)
	repairRead(t, filepath.Join(repairResults, "selected-manifest.json"), &regenerated)
	require.Equal(t, original.Baseline, manifest.Baseline)
	require.NoError(t, repairManifestCompatible(original.Probes, manifest.Probes))
	require.Equal(t, manifest.Probes, regenerated.Probes)
	var baseline, candidate revision
	repairRead(t, filepath.Join(repairOriginal, "base-source.json"), &baseline)
	repairRead(t, filepath.Join(repairResults, "candidate-source.json"), &candidate)
	require.Equal(t, manifest.Baseline, baseline.Commit)
	require.Equal(t, manifest.Baseline, baseline.Input)
	require.Equal(t, manifest.Baseline, candidate.Commit)
	var files map[string]sourceFile
	repairRead(t, candidate.Source+"-files.json", &files)
	require.NoError(t, verifySourceSnapshot(candidate.Source, files))
	identity, err := snapshotIdentity(files)
	require.NoError(t, err)
	require.Equal(t, identity, candidate.ContentID)
	require.Equal(t, repairAcceptedFinalContentID, identity, "final capture must be explicitly approved before retained acceptance")
	require.Equal(t, filepath.Join(repairResults, "candidate-source"), candidate.Source)
	require.Equal(t, "/Users/luca/.codex/worktrees/body-type-names/loom", candidate.Input)
	require.Equal(t, "/Users/luca/go/pkg/mod/github.com/!cali!luke/loom@v1.10.0-alpha.2.0.20260929034221-2f6bfbf7758f", baseline.Source)
	for _, where := range []string{filepath.Join(repairOriginal, "base-generator.json"), filepath.Join(repairOriginal, "candidate-generator.json"), filepath.Join(repairResults, "candidate-generator.json")} {
		var built phaseResult
		repairRead(t, where, &built)
		require.Zero(t, built.Exit)
		require.False(t, built.Infrastructure)
	}
	require.Len(t, manifest.Probes, 66)
	for _, p := range manifest.Probes {
		t.Run(p.ID, func(t *testing.T) {
			// The recorded discovery expected success, but both retained parent
			// attempts demonstrate the independently reviewed known build failure.
			// The exact manifest delta changes interpretation, never input bytes.
			base := repairPair(t, repairOriginal, "base", p, p.Baseline)
			next := repairPair(t, repairResults, "candidate", p, p.Candidate)
			require.Equal(t, base.Inputs, next.Inputs, "same common source bytes")
			differences := compareArtifacts(base.Artifacts, next.Artifacts)
			for _, difference := range differences {
				require.Contains(t, []string{"gen/http/openapi.json", "gen/http/openapi.yaml"}, difference.Path, "non-schema output must stay exact")
			}
			switch p.ID {
			case "type-identity", "http-s-s-e-variant-projection", "codegen-user-type-package-transports":
				require.Empty(t, differences, "unrelated async representation differences must be gone")
			}
			writeJSON(t, filepath.Join(repairResults, p.ID+"-differences.json"), differences)
			require.NoError(t, checkDifferences(p.ID, differences, manifest.Differences))
		})
	}
}

// repairValidateRecord checks the complete execution trace before interpreting
// its outcome. Retained records cannot claim success by omitting later phases.
func repairValidateRecord(result probeResult, p probe, expected expectation, logs map[string]string) error {
	phases := []string{"gen"}
	if p.Fixture == "" {
		phases = append(phases, "example")
	}
	phases = append(phases, "tidy", "build", "vet")
	if p.Decoder != "" {
		phases = append(phases, "decoder")
	}
	if expected.Phase != "" {
		found := false
		for index, phase := range phases {
			if phase == expected.Phase {
				phases = phases[:index+1]
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected failure phase not in probe execution: %s", expected.Phase)
		}
	}
	if len(result.Phases) != len(phases) {
		return fmt.Errorf("incomplete or extra phase trace: got %d want %d", len(result.Phases), len(phases))
	}
	if len(logs) != len(phases) {
		return fmt.Errorf("missing or extra phase logs: got %d want %d", len(logs), len(phases))
	}
	for index, phase := range result.Phases {
		if phase.Phase != phases[index] {
			return fmt.Errorf("phase %d: got %s want %s", index, phase.Phase, phases[index])
		}
		if phase.Infrastructure {
			return fmt.Errorf("infrastructure failure during %s", phase.Phase)
		}
		if log, ok := logs[phase.Phase]; !ok || log != phase.Output {
			return fmt.Errorf("retained phase log mismatch: %s", phase.Phase)
		}
		shouldFail := expected.Phase != "" && index == len(phases)-1
		if (phase.Exit != 0) != shouldFail {
			return fmt.Errorf("unexpected phase exit at %s: %d", phase.Phase, phase.Exit)
		}
	}
	if err := checkOutcome(expected, firstFailure(result)); err != nil {
		return err
	}
	if p.ID == "byte-representation-ownership" {
		if expected != repairKnownFailure() {
			return fmt.Errorf("unreviewed root-validator disposition")
		}
		if got := digest([]byte(firstFailure(result).Output)); got != repairKnownCompilerSHA {
			return fmt.Errorf("known compiler diagnostic changed: %s", got)
		}
	}
	if expected.Phase != "gen" {
		if len(result.Artifacts) == 0 || result.Artifacts["gen/loom.json"] == "" {
			return fmt.Errorf("successful generation requires retained artifacts and gen/loom.json")
		}
	}
	return nil
}

func TestRepairVerifierNegativeControls(t *testing.T) {
	p := probe{ID: "control", Decoder: "actual"}
	good := probeResult{Artifacts: map[string]string{"gen/loom.json": "hash"}}
	logs := make(map[string]string)
	for _, name := range []string{"gen", "example", "tidy", "build", "vet", "decoder"} {
		good.Phases = append(good.Phases, phaseResult{Phase: name, Output: name})
		logs[name] = name
	}
	require.NoError(t, repairValidateRecord(good, p, expectation{}, logs))
	for _, count := range []int{0, 1, 4, 5} {
		truncated := good
		truncated.Phases = good.Phases[:count]
		require.Error(t, repairValidateRecord(truncated, p, expectation{}, logs))
	}
	missing := good
	missing.Artifacts = nil
	require.Error(t, repairValidateRecord(missing, p, expectation{}, logs))
	missing.Artifacts = map[string]string{"other": "hash"}
	require.Error(t, repairValidateRecord(missing, p, expectation{}, logs))
	bad := good
	bad.Phases = append([]phaseResult(nil), good.Phases...)
	bad.Phases[2], bad.Phases[3] = bad.Phases[3], bad.Phases[2]
	require.Error(t, repairValidateRecord(bad, p, expectation{}, logs))
	bad.Phases = append([]phaseResult(nil), good.Phases...)
	bad.Phases[4].Infrastructure = true
	require.Error(t, repairValidateRecord(bad, p, expectation{}, logs))
	logs["vet"] = "modified"
	require.Error(t, repairValidateRecord(good, p, expectation{}, logs))
	logs["vet"] = "vet"
	logs["unexecuted"] = "extra"
	require.Error(t, repairValidateRecord(good, p, expectation{}, logs))
	delete(logs, "unexecuted")
	failed := probeResult{Phases: []phaseResult{{Phase: "gen", Exit: 1, Output: "known contract rejection"}}, Artifacts: map[string]string{}}
	require.NoError(t, repairValidateRecord(failed, p, expectation{Phase: "gen", Contains: "known contract rejection"}, map[string]string{"gen": "known contract rejection"}))
	fixture := probe{ID: "fixture", Fixture: "checked-in"}
	fixtureResult := good
	fixtureResult.Phases = []phaseResult{good.Phases[0], good.Phases[2], good.Phases[3], good.Phases[4]}
	require.NoError(t, repairValidateRecord(fixtureResult, fixture, expectation{}, map[string]string{"gen": "gen", "tidy": "tidy", "build": "build", "vet": "vet"}))
}

func TestRepairFinalDriverApprovalControls(t *testing.T) {
	var original comparisonManifest
	repairRead(t, filepath.Join(repairOriginal, "selected-manifest.json"), &original)
	var current comparisonManifest
	repairRead(t, "byte_schema_manifest.json", &current)
	require.NoError(t, repairManifestCompatible(original.Probes, current.Probes))
	changed := append([]probe(nil), current.Probes...)
	changed[0].Baseline = repairKnownFailure()
	require.Error(t, repairManifestCompatible(original.Probes, changed))
	changed = append([]probe(nil), current.Probes...)
	changed[len(changed)-1].Candidate.Contains = "generic build error"
	require.Error(t, repairManifestCompatible(original.Probes, changed))
	changed = append([]probe(nil), current.Probes...)
	changed[0], changed[1] = changed[1], changed[0]
	require.Error(t, repairManifestCompatible(original.Probes, changed))
	require.Error(t, repairManifestCompatible(original.Probes, current.Probes[:65]))
	var p probe
	for _, currentProbe := range current.Probes {
		if currentProbe.ID == "byte-representation-ownership" {
			p = currentProbe
		}
	}
	var result probeResult
	location := filepath.Join(repairOriginal, "base", p.ID, "run-1")
	repairRead(t, filepath.Join(location, "result.json"), &result)
	logs := make(map[string]string)
	for _, phase := range result.Phases {
		raw, err := os.ReadFile(filepath.Join(location, phase.Phase+".log"))
		require.NoError(t, err)
		logs[phase.Phase] = string(raw)
	}
	require.NoError(t, repairValidateRecord(result, p, p.Baseline, logs))
	result.Phases = append([]phaseResult(nil), result.Phases...)
	result.Phases[len(result.Phases)-1].Output += " changed"
	logs["build"] += " changed"
	require.ErrorContains(t, repairValidateRecord(result, p, p.Baseline, logs), "known compiler diagnostic changed")
	badExpected := p.Baseline
	badExpected.Issue = "other"
	require.ErrorContains(t, repairValidateRecord(result, p, badExpected, logs), "unreviewed root-validator disposition")
}
