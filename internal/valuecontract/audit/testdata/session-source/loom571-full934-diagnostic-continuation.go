package valuecontract

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	loom571DiagnosticAttempt struct {
		Revision       string `json:"revision"`
		Attempt        int    `json:"attempt"`
		Phase          string `json:"phase"`
		Exit           int    `json:"exit"`
		OutputSHA256   string `json:"output_sha256"`
		ExpectationErr string `json:"expectation_error,omitempty"`
	}
	loom571DiagnosticCase struct {
		CatalogID string                     `json:"catalog_id"`
		ProbeID   string                     `json:"probe_id"`
		Attempts  []loom571DiagnosticAttempt `json:"attempts"`
	}
	loom571DiagnosticPacket struct {
		Index              int                     `json:"index"`
		CatalogSHA         string                  `json:"catalog_sha"`
		CandidateContentID string                  `json:"candidate_content_id"`
		Disposition        string                  `json:"disposition"`
		Cases              []loom571DiagnosticCase `json:"cases"`
	}
)

func loom571DiagnosticPhases(current probe) []string {
	phases := []string{"gen"}
	if current.Fixture == "" {
		phases = append(phases, "example")
	}
	phases = append(phases, "tidy", "build", "vet")
	if current.Decoder != "" {
		phases = append(phases, "decoder")
	}
	return phases
}

func loom571DiagnosticAttemptRead(
	t *testing.T,
	root string,
	label string,
	current probe,
	attempt int,
	expected expectation,
) (probeResult, loom571DiagnosticAttempt) {
	t.Helper()
	location := filepath.Join(root, label, current.ID, fmt.Sprintf("run-%d", attempt))
	var result probeResult
	repairRead(t, filepath.Join(location, "result.json"), &result)
	require.Equal(t, current.ID, result.ID)

	logs := make(map[string]string)
	entries, err := os.ReadDir(location)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".log" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(location, entry.Name()))
		require.NoError(t, err)
		logs[entry.Name()[:len(entry.Name())-len(".log")]] = string(raw)
	}

	phases := loom571DiagnosticPhases(current)
	require.NotEmpty(t, result.Phases)
	require.LessOrEqual(t, len(result.Phases), len(phases))
	require.Len(t, logs, len(result.Phases))
	for index, phase := range result.Phases {
		require.Equal(t, phases[index], phase.Phase)
		require.False(t, phase.Infrastructure, "infrastructure failure during %s", phase.Phase)
		require.Equal(t, phase.Output, logs[phase.Phase], "retained phase log mismatch")
		if index < len(result.Phases)-1 {
			require.Zero(t, phase.Exit, "non-final phase failed")
		}
	}
	last := result.Phases[len(result.Phases)-1]
	if len(result.Phases) < len(phases) {
		require.NotZero(t, last.Exit, "truncated successful phase trace")
	} else if last.Exit != 0 {
		require.Equal(t, phases[len(phases)-1], last.Phase, "non-final phase failed")
	}
	require.Equal(t, result.Inputs, repairTreeHashes(t, filepath.Join(location, "inputs")))
	require.Equal(t, result.Artifacts, repairTreeHashes(t, filepath.Join(location, "artifacts")))
	if last.Exit == 0 || last.Phase != "gen" {
		require.NotEmpty(t, result.Artifacts)
		require.NotEmpty(t, result.Artifacts["gen/loom.json"])
	}

	validationErr := repairValidateRecord(result, current, expected, logs)
	diagnostic := loom571DiagnosticAttempt{
		Revision: label,
		Attempt:  attempt,
		Phase:    last.Phase,
		Exit:     last.Exit,
	}
	if last.Output != "" {
		diagnostic.OutputSHA256 = digest([]byte(last.Output))
	}
	if validationErr != nil {
		diagnostic.ExpectationErr = validationErr.Error()
	}
	return result, diagnostic
}

func TestLoom571ShardDiagnosticAcquisition(t *testing.T) {
	manifest := fullCatalogManifestRead(t)
	index := loom571ShardIndex(t)
	cases := loom571ShardCases(t, manifest, index)
	loom571SharedSources(t, manifest)
	var evidence loom571ShardEvidence
	repairRead(t, filepath.Join(loom571ShardRoot, "shards", fmt.Sprintf("%02d.json", index)), &evidence)
	require.Equal(t, loom571Selection(manifest, index, cases), evidence.Selection)
	archive := filepath.Join(loom571ShardRoot, "shards", fmt.Sprintf("%02d.tar.gz", index))
	archiveSHA, archiveBytes, err := loom571FileSHA(archive)
	require.NoError(t, err)
	require.Equal(t, evidence.ArchiveSHA256, archiveSHA)
	require.Equal(t, evidence.ArchiveBytes, archiveBytes)

	loom571RequireSpace(t, loom571AttemptFloor)
	rehydrated := filepath.Join(t.TempDir(), "rehydrated")
	require.NoError(t, os.Mkdir(rehydrated, 0700))
	require.NoError(t, loom571ExtractAndVerify(archive, rehydrated, evidence.Records))
	loom571RequireSpace(t, loom571AttemptFloor)
	expandedBytes, err := loom571TreeSize(rehydrated)
	require.NoError(t, err)
	require.Equal(t, evidence.ExpandedBytes, expandedBytes)
	measurements, samples := loom571MeasureCases(t, cases, rehydrated)
	require.Equal(t, evidence.CaseMeasurements, measurements)
	require.GreaterOrEqual(t, len(evidence.FreeSamples), len(samples))
	require.Equal(t, samples, evidence.FreeSamples[:len(samples)])
	for _, sample := range evidence.FreeSamples {
		require.GreaterOrEqual(t, sample, loom571AttemptFloor)
	}
	require.Equal(t, evidence.MinimumFree, loom571MinimumFree(t, evidence.FreeSamples))

	packet := loom571DiagnosticPacket{
		Index:              index,
		CatalogSHA:         fullCatalogSHA,
		CandidateContentID: manifest.CandidateContentID,
		Disposition:        "quarantined-unaccepted",
	}
	mismatchCount := 0
	for _, current := range cases {
		currentPacket := loom571DiagnosticCase{CatalogID: current.CatalogID, ProbeID: current.Probe.ID}
		var pairs [2][2]probeResult
		for revisionIndex, label := range []string{"base", "candidate"} {
			expected := []expectation{current.Probe.Baseline, current.Probe.Candidate}[revisionIndex]
			for attempt := range 2 {
				result, diagnostic := loom571DiagnosticAttemptRead(
					t, rehydrated, label, current.Probe, attempt+1, expected,
				)
				pairs[revisionIndex][attempt] = result
				currentPacket.Attempts = append(currentPacket.Attempts, diagnostic)
				if diagnostic.ExpectationErr != "" {
					mismatchCount++
				}
			}
			require.Equal(t, pairs[revisionIndex][0].Inputs, pairs[revisionIndex][1].Inputs)
			require.Empty(t, compareArtifacts(
				pairs[revisionIndex][0].Artifacts, pairs[revisionIndex][1].Artifacts,
			))
			require.Equal(t, string(pairs[revisionIndex][0].Observation), string(pairs[revisionIndex][1].Observation))
		}
		require.Equal(t, pairs[0][0].Inputs, pairs[1][0].Inputs)
		differences := compareArtifacts(pairs[0][0].Artifacts, pairs[1][0].Artifacts)
		var recorded []artifactDifference
		repairRead(t, filepath.Join(rehydrated, current.Probe.ID+"-differences.json"), &recorded)
		if len(differences) == 0 {
			require.Empty(t, recorded)
		} else {
			require.Equal(t, differences, recorded)
		}
		packet.Cases = append(packet.Cases, currentPacket)
	}
	require.Positive(t, mismatchCount, "diagnostic continuation requires a quarantined expectation mismatch")
	require.Len(t, packet.Cases, len(cases))
	packetPath := filepath.Join(loom571ShardRoot, "logs", fmt.Sprintf("shard-%02d-diagnostic.json", index))
	require.NoFileExists(t, packetPath)
	writeJSON(t, packetPath, packet)
}
