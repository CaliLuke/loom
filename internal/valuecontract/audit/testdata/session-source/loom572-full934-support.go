package valuecontract

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

type fullCatalogCase struct {
	CatalogID string `json:"catalog_id"`
	Probe     probe  `json:"probe"`
}

type fullCatalogManifest struct {
	Baseline           string            `json:"baseline"`
	CandidateContentID string            `json:"candidate_content_id"`
	CommonContentID    string            `json:"common_content_id"`
	Cases              []fullCatalogCase `json:"cases"`
}

type fullCatalogExpectation struct {
	Phase    string `json:"phase"`
	Contains string `json:"contains"`
	Reason   string `json:"reason"`
	Issue    string `json:"issue,omitempty"`
}

const fullCatalogResults = "/tmp/loom572-full934-results"
const fullCatalogCapture = "/tmp/loom572-full934-parent95cf"
const fullCatalogCandidateCapture = "/tmp/loom572-full934-candidate-freeze4"
const fullCatalogManifestPath = "/tmp/loom572-full934-manifest.json"
const fullCatalogManifestSHA = "e2e92e9132c57ed060c712d45f9c7e8b2814a8e3cf876a399a1ab71bccdbae8b"
const fullCatalogPath = "/tmp/loom571-full-catalog.json"
const fullCatalogSHA = "033f6944f44b37e236c4bc6b453b6e8d3b209dbf715d4ffb4d402ec53c779152"
const fullCatalogCommit = "95cfbbff3bcaeebec3037b5ee996d9e89f084789"
const fullCatalogBaseID = "b613254468a747bb948893206b5eb0118c6a33cde963e590e56d4ff66e041851"

func fullCatalogPinnedRead(t *testing.T, path, sha string, value any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, sha, digest(raw), path)
	repairRead(t, path, value)
}

func fullCatalogProbe(catalogID string, expected expectation) probe {
	separator := strings.LastIndexByte(catalogID, '/')
	if separator <= 0 || separator == len(catalogID)-1 {
		return probe{}
	}
	domain, function := catalogID[:separator], catalogID[separator+1:]
	id := strings.NewReplacer("/", "-", "_", "-").Replace(strings.ToLower(domain + "-" + function))
	return probe{
		ID: id, Import: "github.com/CaliLuke/loom/" + domain + "/testdata", Function: function,
		Coverage: []string{"full five-domain exported design catalog"},
		Baseline: expected, Candidate: expected,
	}
}

func fullCatalogManifestRead(t *testing.T) fullCatalogManifest {
	t.Helper()
	var result fullCatalogManifest
	fullCatalogPinnedRead(t, fullCatalogManifestPath, fullCatalogManifestSHA, &result)
	require.Equal(t, fullCatalogCommit, result.Baseline)
	require.NotEqual(t, "", result.CandidateContentID)
	require.Equal(t, result.CandidateContentID, result.CommonContentID)
	var catalog []string
	fullCatalogPinnedRead(t, fullCatalogPath, fullCatalogSHA, &catalog)
	require.Len(t, catalog, 934)
	var authored map[string]fullCatalogExpectation
	repairRead(t, filepath.Join(fullCatalogCapture, "common-source", "internal", "testdatacompile", "expectations.json"), &authored)
	require.Len(t, result.Cases, len(catalog))
	seen := make(map[string]bool, len(result.Cases))
	for index, catalogID := range catalog {
		expected := expectation{}
		if raw, present := authored[catalogID]; present {
			expected = expectation{Phase: raw.Phase, Contains: raw.Contains, Issue: raw.Issue}
			require.Equal(t, "intentional validation failure", raw.Reason, catalogID)
		}
		want := fullCatalogProbe(catalogID, expected)
		require.Equal(t, catalogID, result.Cases[index].CatalogID)
		require.True(t, reflect.DeepEqual(want, result.Cases[index].Probe), catalogID)
		require.False(t, seen[want.ID], "duplicate probe ID %s", want.ID)
		seen[want.ID] = true
	}
	require.Len(t, authored, 77)
	return result
}

func fullCatalogTreeID(t *testing.T, root string) string {
	t.Helper()
	var paths []string
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, relative)
		return nil
	}))
	inventory, err := sourceInventory(root, paths)
	require.NoError(t, err)
	identity, err := snapshotIdentity(inventory)
	require.NoError(t, err)
	return identity
}

func fullCatalogSources(t *testing.T, manifest fullCatalogManifest) [2]revision {
	t.Helper()
	var base, candidate revision
	repairRead(t, filepath.Join(fullCatalogCapture, "base-source.json"), &base)
	repairRead(t, filepath.Join(fullCatalogCandidateCapture, "candidate-source.json"), &candidate)
	require.Equal(t, fullCatalogCommit, base.Input)
	require.Equal(t, fullCatalogCommit, base.Commit)
	require.Empty(t, base.ContentID)
	require.Equal(t, fullCatalogBaseID, fullCatalogTreeID(t, base.Source))
	require.Equal(t, fullCatalogCommit, candidate.Commit)
	require.Equal(t, manifest.CandidateContentID, candidate.ContentID)
	require.Equal(t, "/private/tmp/loom572-prep-95cf", candidate.Input)
	require.Equal(t, candidate.Input, candidate.OriginalSource)
	require.Equal(t, filepath.Join(fullCatalogCandidateCapture, "common-source"), candidate.Source)
	require.Equal(t, candidate.Source, filepath.Join(fullCatalogCandidateCapture, "common-source"))
	require.Equal(t, manifest.CandidateContentID, fullCatalogTreeID(t, candidate.Source))
	var common struct {
		ContentID      string   `json:"content_id"`
		Excluded       []string `json:"excluded"`
		OriginalSource string   `json:"original_source"`
	}
	repairRead(t, filepath.Join(fullCatalogCandidateCapture, "common-source.json"), &common)
	require.Equal(t, manifest.CommonContentID, common.ContentID)
	require.Equal(t, []string{".git metadata", ".claude agent-only pointers"}, common.Excluded)
	require.Equal(t, candidate.OriginalSource, common.OriginalSource)
	require.Equal(t, manifest.CommonContentID, fullCatalogTreeID(t, candidate.Source))
	return [2]revision{base, candidate}
}

func fullCatalogSpace(minimum uint64) error {
	var state syscall.Statfs_t
	if err := syscall.Statfs("/tmp", &state); err != nil {
		return err
	}
	if available := uint64(state.Bavail) * uint64(state.Bsize); available < minimum {
		return fmt.Errorf("less than %d GiB free; preserve evidence and stop", minimum>>30)
	}
	return nil
}

func TestFullCatalogControls(t *testing.T) {
	manifest := fullCatalogManifestRead(t)
	sources := fullCatalogSources(t, manifest)
	require.Len(t, manifest.Cases, 934)
	require.Equal(t, filepath.Join(fullCatalogCandidateCapture, "common-source"), sources[1].Source)
	require.NoError(t, fullCatalogSpace(30<<30))
}

func TestFullCatalogGeneration(t *testing.T) {
	require.Equal(t, "2", flag.Lookup("test.parallel").Value.String(), "exact two-worker resource policy")
	manifest := fullCatalogManifestRead(t)
	sources := fullCatalogSources(t, manifest)
	require.NoError(t, fullCatalogSpace(30<<30))
	require.NoError(t, os.Mkdir(fullCatalogResults, 0700), "never mix distinct runs")
	writeJSON(t, filepath.Join(fullCatalogResults, "selected-manifest.json"), manifest)
	t.Cleanup(func() {
		fullCatalogSources(t, manifest)
	})
	for index, label := range []string{"base", "candidate"} {
		sources[index].Binary = filepath.Join(fullCatalogResults, "bin", label, "loom")
		require.NoError(t, os.MkdirAll(filepath.Dir(sources[index].Binary), 0700))
		built := runPhase(t.Context(), sources[index].Source, nil, "build-generator", "go", "build", "-o", sources[index].Binary, "./cmd/loom")
		writeJSON(t, filepath.Join(fullCatalogResults, label+"-source.json"), sources[index])
		writeJSON(t, filepath.Join(fullCatalogResults, label+"-generator.json"), built)
		require.Zero(t, built.Exit, "%s generator: %s", label, built.Output)
		require.False(t, built.Infrastructure)
	}
	require.Equal(t, filepath.Base(sources[0].Binary), filepath.Base(sources[1].Binary))
	for _, c := range manifest.Cases {
		t.Run(c.Probe.ID, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, fullCatalogSpace(12<<30))
			var runs [2][2]probeResult
			for index, label := range []string{"base", "candidate"} {
				for attempt := range 2 {
					require.NoError(t, fullCatalogSpace(12<<30))
					location := filepath.Join(fullCatalogResults, label, c.Probe.ID, fmt.Sprintf("run-%d", attempt+1))
					runs[index][attempt] = runProbe(t, sources[index], c.Probe, location, sources[1].Source)
				}
			}
			differences := compareArtifacts(runs[0][0].Artifacts, runs[1][0].Artifacts)
			writeJSON(t, filepath.Join(fullCatalogResults, c.Probe.ID+"-differences.json"), differences)
			base := repairPair(t, fullCatalogResults, "base", c.Probe, c.Probe.Baseline)
			candidate := repairPair(t, fullCatalogResults, "candidate", c.Probe, c.Probe.Candidate)
			require.Equal(t, base.Inputs, candidate.Inputs, "same captured specimen bytes")
		})
	}
}

func TestFullCatalogRetainedComparison(t *testing.T) {
	manifest := fullCatalogManifestRead(t)
	sources := fullCatalogSources(t, manifest)
	var retained fullCatalogManifest
	repairRead(t, filepath.Join(fullCatalogResults, "selected-manifest.json"), &retained)
	require.True(t, reflect.DeepEqual(manifest, retained))
	for index, label := range []string{"base", "candidate"} {
		sources[index].Binary = filepath.Join(fullCatalogResults, "bin", label, "loom")
		var source revision
		repairRead(t, filepath.Join(fullCatalogResults, label+"-source.json"), &source)
		require.Equal(t, sources[index], source)
		var built phaseResult
		repairRead(t, filepath.Join(fullCatalogResults, label+"-generator.json"), &built)
		require.Equal(t, "build-generator", built.Phase)
		require.Zero(t, built.Exit)
		require.False(t, built.Infrastructure)
	}
	for _, c := range manifest.Cases {
		t.Run(c.Probe.ID, func(t *testing.T) {
			base := repairPair(t, fullCatalogResults, "base", c.Probe, c.Probe.Baseline)
			candidate := repairPair(t, fullCatalogResults, "candidate", c.Probe, c.Probe.Candidate)
			require.Equal(t, base.Inputs, candidate.Inputs)
			differences := compareArtifacts(base.Artifacts, candidate.Artifacts)
			var recorded []artifactDifference
			repairRead(t, filepath.Join(fullCatalogResults, c.Probe.ID+"-differences.json"), &recorded)
			require.True(t, slices.Equal(differences, recorded), "recorded difference entries/order must match actual artifacts")
			require.Empty(t, differences, "no observed difference is implicitly allowed")
		})
	}
}
