package valuecontract

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	loom571ShardRoot       = "/tmp/loom572-full934-sharded-results"
	loom571ShardSize       = 16
	loom571ShardCount      = 59
	loom571ShardPilotIndex = 0
	loom571PilotFloor      = uint64(14 << 30)
	loom571FullFloor       = uint64(16 << 30)
	loom571AttemptFloor    = uint64(12 << 30)
)

type (
	loom571ShardSelection struct {
		Index              int      `json:"index"`
		CatalogSHA         string   `json:"catalog_sha"`
		CandidateContentID string   `json:"candidate_content_id"`
		IDs                []string `json:"ids"`
	}
	loom571ShardShared struct {
		CandidateContentID string      `json:"candidate_content_id"`
		Sources            [2]revision `json:"sources"`
		BinaryPaths        [2]string   `json:"binary_paths"`
		BinarySHA256       [2]string   `json:"binary_sha256"`
	}
	loom571ShardCaseStatus struct {
		ID          string               `json:"id"`
		FreeSamples []uint64             `json:"free_samples"`
		Differences []artifactDifference `json:"differences"`
	}
	loom571ArchiveRecord struct {
		Path   string `json:"path"`
		Type   string `json:"type"`
		Mode   uint32 `json:"mode"`
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
	}
	loom571CaseMeasure struct {
		ID              string `json:"id"`
		ExpandedBytes   int64  `json:"expanded_bytes"`
		CompressedBytes int64  `json:"compressed_bytes"`
	}
	loom571ShardEvidence struct {
		Selection        loom571ShardSelection  `json:"selection"`
		ArchiveSHA256    string                 `json:"archive_sha256"`
		ArchiveBytes     int64                  `json:"archive_bytes"`
		ExpandedBytes    int64                  `json:"expanded_bytes"`
		MinimumFree      uint64                 `json:"minimum_free_bytes"`
		FreeSamples      []uint64               `json:"free_samples"`
		Records          []loom571ArchiveRecord `json:"records"`
		CaseMeasurements []loom571CaseMeasure   `json:"case_measurements"`
	}
	loom571CountWriter struct {
		size int64
	}
)

func (writer *loom571CountWriter) Write(data []byte) (int, error) {
	writer.size += int64(len(data))
	return len(data), nil
}

func loom571FreeBytes() (uint64, error) {
	var state syscall.Statfs_t
	if err := syscall.Statfs("/tmp", &state); err != nil {
		return 0, err
	}
	return uint64(state.Bavail) * uint64(state.Bsize), nil
}

func loom571RequireSpace(t *testing.T, minimum uint64) uint64 {
	t.Helper()
	available, err := loom571FreeBytes()
	require.NoError(t, err)
	require.GreaterOrEqual(t, available, minimum,
		"preserve evidence and stop below %d GiB", minimum>>30)
	return available
}

func loom571ShardCases(t *testing.T, manifest fullCatalogManifest, index int) []fullCatalogCase {
	t.Helper()
	require.GreaterOrEqual(t, index, 0)
	require.Less(t, index, loom571ShardCount)
	start := index * loom571ShardSize
	end := min(start+loom571ShardSize, len(manifest.Cases))
	require.Less(t, start, end)
	return append([]fullCatalogCase(nil), manifest.Cases[start:end]...)
}

func loom571ShardIndex(t *testing.T) int {
	t.Helper()
	raw := os.Getenv("LOOM571_SHARD_INDEX")
	require.NotEmpty(t, raw)
	index, err := strconv.Atoi(raw)
	require.NoError(t, err)
	require.GreaterOrEqual(t, index, 0)
	require.Less(t, index, loom571ShardCount)
	if os.Getenv("LOOM571_SHARD_PILOT") == "1" {
		require.Equal(t, loom571ShardPilotIndex, index)
	}
	return index
}

func loom571ShardMinimum(t *testing.T) uint64 {
	t.Helper()
	if os.Getenv("LOOM571_SHARD_PILOT") == "1" {
		return loom571PilotFloor
	}
	return loom571FullFloor
}

func loom571FileSHA(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	size, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return "", 0, copyErr
	}
	if closeErr != nil {
		return "", 0, closeErr
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func loom571StrictSources(t *testing.T, manifest fullCatalogManifest) [2]revision {
	t.Helper()
	sources := fullCatalogSources(t, manifest)
	require.Equal(t, "/tmp/loom572-full934-parent95cf/common-source", sources[0].Source)
	require.Equal(t, "/private/tmp/loom572-prep-95cf", sources[1].Input)
	require.Equal(t, sources[1].Input, sources[1].OriginalSource)
	return sources
}

func loom571ValidateGeneratorReceipts(t *testing.T) {
	t.Helper()
	for _, label := range []string{"base", "candidate"} {
		var built phaseResult
		repairRead(t, filepath.Join(loom571ShardRoot, label+"-generator.json"), &built)
		require.Equal(t, "build-generator", built.Phase)
		require.Zero(t, built.Exit)
		require.Empty(t, built.Output)
		require.False(t, built.Infrastructure)
	}
}

func loom571EnsureShared(t *testing.T, manifest fullCatalogManifest) loom571ShardShared {
	t.Helper()
	sources := loom571StrictSources(t, manifest)
	path := filepath.Join(loom571ShardRoot, "shared.json")
	_, statErr := os.Stat(path)
	if statErr == nil {
		var shared loom571ShardShared
		repairRead(t, path, &shared)
		require.Equal(t, manifest.CandidateContentID, shared.CandidateContentID)
		for index := range sources {
			require.Equal(t, sources[index], shared.Sources[index])
			require.Equal(t, filepath.Join(loom571ShardRoot, "bin", []string{"base", "candidate"}[index], "loom"), shared.BinaryPaths[index])
			got, _, hashErr := loom571FileSHA(shared.BinaryPaths[index])
			require.NoError(t, hashErr)
			require.Equal(t, shared.BinarySHA256[index], got)
			sources[index].Binary = shared.BinaryPaths[index]
		}
		loom571ValidateGeneratorReceipts(t)
		return shared
	}
	require.ErrorIs(t, statErr, fs.ErrNotExist)
	require.NoError(t, os.Mkdir(loom571ShardRoot, 0700))
	shared := loom571ShardShared{CandidateContentID: manifest.CandidateContentID, Sources: sources}
	for index, label := range []string{"base", "candidate"} {
		binary := filepath.Join(loom571ShardRoot, "bin", label, "loom")
		require.NoError(t, os.MkdirAll(filepath.Dir(binary), 0700))
		built := runPhase(t.Context(), sources[index].Source, nil,
			"build-generator", "go", "build", "-o", binary, "./cmd/loom")
		writeJSON(t, filepath.Join(loom571ShardRoot, label+"-generator.json"), built)
		require.Zero(t, built.Exit, "%s generator: %s", label, built.Output)
		require.False(t, built.Infrastructure)
		hash, _, hashErr := loom571FileSHA(binary)
		require.NoError(t, hashErr)
		shared.BinaryPaths[index], shared.BinarySHA256[index] = binary, hash
	}
	writeJSON(t, path, shared)
	loom571ValidateGeneratorReceipts(t)
	return shared
}

func loom571SharedSources(t *testing.T, manifest fullCatalogManifest) [2]revision {
	t.Helper()
	shared := loom571EnsureShared(t, manifest)
	sources := loom571StrictSources(t, manifest)
	for index := range sources {
		require.Equal(t, shared.Sources[index], sources[index])
		sources[index].Binary = shared.BinaryPaths[index]
	}
	return sources
}

func loom571CollectRecords(root string) ([]loom571ArchiveRecord, error) {
	var records []loom571ArchiveRecord
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
			return fmt.Errorf("unsupported special mode bits for %s: %s", relative, info.Mode())
		}
		record := loom571ArchiveRecord{Path: relative, Mode: uint32(info.Mode().Perm())}
		switch {
		case info.IsDir():
			record.Type = "directory"
			record.SHA256 = digest(nil)
		case info.Mode().IsRegular():
			record.Type = "file"
			record.SHA256, record.Size, err = loom571FileSHA(path)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported retained entry %s with mode %s", relative, info.Mode())
		}
		records = append(records, record)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(records, func(left, right loom571ArchiveRecord) int {
		return strings.Compare(left.Path, right.Path)
	})
	for index := 1; index < len(records); index++ {
		if records[index-1].Path == records[index].Path {
			return nil, fmt.Errorf("duplicate normalized path %s", records[index].Path)
		}
	}
	return records, nil
}

func loom571SafeArchivePath(path string) bool {
	return path != "" && !filepath.IsAbs(path) && filepath.ToSlash(filepath.Clean(path)) == path &&
		path != ".." && !strings.HasPrefix(path, "../")
}

func loom571WriteArchiveTo(writer io.Writer, root string, records []loom571ArchiveRecord) error {
	gzipWriter, err := gzip.NewWriterLevel(writer, gzip.BestCompression)
	if err != nil {
		return err
	}
	gzipWriter.Header.ModTime = time.Unix(0, 0)
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	closeWriters := func() error {
		if err := tarWriter.Close(); err != nil {
			return err
		}
		return gzipWriter.Close()
	}
	for _, record := range records {
		if !loom571SafeArchivePath(record.Path) {
			return fmt.Errorf("unsafe archive path %q", record.Path)
		}
		header := &tar.Header{
			Name: record.Path, Mode: int64(record.Mode), ModTime: time.Unix(0, 0),
			AccessTime: time.Time{}, ChangeTime: time.Time{}, Uid: 0, Gid: 0,
			Uname: "", Gname: "", Format: tar.FormatPAX,
		}
		switch record.Type {
		case "directory":
			header.Typeflag = tar.TypeDir
		case "file":
			header.Typeflag = tar.TypeReg
			header.Size = record.Size
		default:
			return fmt.Errorf("unsupported archive record type %q", record.Type)
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if record.Type != "file" {
			continue
		}
		file, err := os.Open(filepath.Join(root, filepath.FromSlash(record.Path)))
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tarWriter, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return closeWriters()
}

func loom571WriteArchive(path, root string, records []loom571ArchiveRecord) (string, int64, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", 0, err
	}
	writeErr := loom571WriteArchiveTo(file, root, records)
	closeErr := file.Close()
	if writeErr != nil {
		return "", 0, writeErr
	}
	if closeErr != nil {
		return "", 0, closeErr
	}
	return loom571FileSHA(path)
}

func loom571ExtractAndVerify(archive, destination string, expected []loom571ArchiveRecord) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}
	tarReader := tar.NewReader(gzipReader)
	seen := make(map[string]bool, len(expected))
	index := 0
	for {
		header, nextErr := tarReader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nextErr
		}
		if !loom571SafeArchivePath(header.Name) {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		if seen[header.Name] {
			return fmt.Errorf("duplicate archive path %s", header.Name)
		}
		seen[header.Name] = true
		if index >= len(expected) || expected[index].Path != header.Name {
			return fmt.Errorf("unexpected archive record %s at %d", header.Name, index)
		}
		record := expected[index]
		index++
		if uint32(header.Mode)&0777 != record.Mode {
			return fmt.Errorf("mode mismatch for %s", record.Path)
		}
		target := filepath.Join(destination, filepath.FromSlash(record.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		switch record.Type {
		case "directory":
			if header.Typeflag != tar.TypeDir {
				return fmt.Errorf("type mismatch for %s", record.Path)
			}
			if err := os.Mkdir(target, fs.FileMode(record.Mode)); err != nil && !errors.Is(err, fs.ErrExist) {
				return err
			}
		case "file":
			if header.Typeflag != tar.TypeReg || header.Size != record.Size {
				return fmt.Errorf("type or size mismatch for %s", record.Path)
			}
			output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fs.FileMode(record.Mode))
			if err != nil {
				return err
			}
			hash := sha256.New()
			_, copyErr := io.Copy(io.MultiWriter(output, hash), tarReader)
			closeErr := output.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			if hex.EncodeToString(hash.Sum(nil)) != record.SHA256 {
				return fmt.Errorf("content mismatch for %s", record.Path)
			}
		default:
			return fmt.Errorf("unsupported expected type %s", record.Type)
		}
	}
	if index != len(expected) {
		return fmt.Errorf("archive missing records: got %d want %d", index, len(expected))
	}
	if err := gzipReader.Close(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	actual, err := loom571CollectRecords(destination)
	if err != nil {
		return err
	}
	if !slices.Equal(actual, expected) {
		return fmt.Errorf("extracted inventory mismatch")
	}
	return nil
}

func loom571Selection(manifest fullCatalogManifest, index int, cases []fullCatalogCase) loom571ShardSelection {
	selection := loom571ShardSelection{
		Index:              index,
		CatalogSHA:         fullCatalogSHA,
		CandidateContentID: manifest.CandidateContentID,
		IDs:                make([]string, 0, len(cases)),
	}
	for _, current := range cases {
		selection.IDs = append(selection.IDs, current.CatalogID)
	}
	return selection
}

func loom571TreeSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func loom571CompressedTreeSize(root string) (int64, error) {
	records, err := loom571CollectRecords(root)
	if err != nil {
		return 0, err
	}
	counter := &loom571CountWriter{}
	if err := loom571WriteArchiveTo(counter, root, records); err != nil {
		return 0, err
	}
	return counter.size, nil
}

func loom571MeasureCases(t *testing.T, cases []fullCatalogCase, expanded string) ([]loom571CaseMeasure, []uint64) {
	t.Helper()
	measurements := make([]loom571CaseMeasure, 0, len(cases))
	var samples []uint64
	for _, current := range cases {
		var status loom571ShardCaseStatus
		repairRead(t, filepath.Join(expanded, "status", current.CatalogID+".json"), &status)
		require.Equal(t, current.CatalogID, status.ID)
		samples = append(samples, status.FreeSamples...)
		var expandedBytes, compressedBytes int64
		for _, label := range []string{"base", "candidate"} {
			root := filepath.Join(expanded, label, current.Probe.ID)
			size, err := loom571TreeSize(root)
			require.NoError(t, err)
			compressed, err := loom571CompressedTreeSize(root)
			require.NoError(t, err)
			expandedBytes += size
			compressedBytes += compressed
		}
		measurements = append(measurements, loom571CaseMeasure{
			ID: current.CatalogID, ExpandedBytes: expandedBytes, CompressedBytes: compressedBytes,
		})
	}
	return measurements, samples
}

func loom571MinimumFree(t *testing.T, samples []uint64) uint64 {
	t.Helper()
	require.NotEmpty(t, samples)
	minimum := samples[0]
	for _, sample := range samples[1:] {
		minimum = min(minimum, sample)
	}
	return minimum
}

func loom571ArchiveShard(t *testing.T, selection loom571ShardSelection, cases []fullCatalogCase, expanded string) {
	t.Helper()
	measurements, freeSamples := loom571MeasureCases(t, cases, expanded)
	freeSamples = append(freeSamples, loom571RequireSpace(t, loom571AttemptFloor))
	records, err := loom571CollectRecords(expanded)
	require.NoError(t, err)
	expandedBytes, err := loom571TreeSize(expanded)
	require.NoError(t, err)
	archive := filepath.Join(loom571ShardRoot, "shards", fmt.Sprintf("%02d.tar.gz", selection.Index))
	require.NoError(t, os.MkdirAll(filepath.Dir(archive), 0700))
	freeSamples = append(freeSamples, loom571RequireSpace(t, loom571AttemptFloor))
	archiveSHA, archiveBytes, err := loom571WriteArchive(archive, expanded, records)
	require.NoError(t, err)
	freeSamples = append(freeSamples, loom571RequireSpace(t, loom571AttemptFloor))
	verifyRoot := filepath.Join(loom571ShardRoot, "verify", fmt.Sprintf("%02d", selection.Index))
	require.NoError(t, os.MkdirAll(verifyRoot, 0700))
	require.NoError(t, loom571ExtractAndVerify(archive, verifyRoot, records))
	freeSamples = append(freeSamples, loom571RequireSpace(t, loom571AttemptFloor))
	evidence := loom571ShardEvidence{
		Selection: selection, ArchiveSHA256: archiveSHA, ArchiveBytes: archiveBytes,
		ExpandedBytes: expandedBytes, MinimumFree: loom571MinimumFree(t, freeSamples), FreeSamples: freeSamples,
		Records: records, CaseMeasurements: measurements,
	}
	writeJSON(t, filepath.Join(loom571ShardRoot, "shards", fmt.Sprintf("%02d.json", selection.Index)), evidence)
	require.NoError(t, os.RemoveAll(verifyRoot))
	require.NoError(t, os.RemoveAll(expanded))
}

func TestLoom571ShardPartition(t *testing.T) {
	manifest := fullCatalogManifestRead(t)
	require.Len(t, manifest.Cases, 934)
	var got []string
	seen := make(map[string]bool, len(manifest.Cases))
	for index := range loom571ShardCount {
		cases := loom571ShardCases(t, manifest, index)
		if index < loom571ShardCount-1 {
			require.Len(t, cases, loom571ShardSize)
		} else {
			require.Len(t, cases, 6)
		}
		for _, current := range cases {
			require.False(t, seen[current.CatalogID], current.CatalogID)
			seen[current.CatalogID] = true
			got = append(got, current.CatalogID)
		}
	}
	want := make([]string, 0, len(manifest.Cases))
	for _, current := range manifest.Cases {
		want = append(want, current.CatalogID)
	}
	require.Equal(t, want, got)
	require.Len(t, seen, 934)
	pilot := loom571ShardCases(t, manifest, loom571ShardPilotIndex)
	var ids []string
	for _, current := range pilot {
		ids = append(ids, current.CatalogID)
	}
	writeJSON(t, "/tmp/loom572-full934-pilot-membership.json", ids)
}

func TestLoom571ShardGeneration(t *testing.T) {
	require.Equal(t, "2", flag.Lookup("test.parallel").Value.String(), "exact two-worker resource policy")
	manifest := fullCatalogManifestRead(t)
	index := loom571ShardIndex(t)
	cases := loom571ShardCases(t, manifest, index)
	loom571RequireSpace(t, loom571ShardMinimum(t))
	sources := loom571SharedSources(t, manifest)
	expanded := filepath.Join(loom571ShardRoot, "expanded", fmt.Sprintf("%02d", index))
	require.NoError(t, os.MkdirAll(filepath.Dir(expanded), 0700))
	require.NoError(t, os.Mkdir(expanded, 0700), "never mix distinct shard runs")
	selection := loom571Selection(manifest, index, cases)
	writeJSON(t, filepath.Join(expanded, "selection.json"), selection)
	t.Cleanup(func() {
		loom571StrictSources(t, manifest)
		loom571ArchiveShard(t, selection, cases, expanded)
	})
	for _, current := range cases {
		current := current
		t.Run(current.Probe.ID, func(t *testing.T) {
			t.Parallel()
			status := loom571ShardCaseStatus{ID: current.CatalogID}
			var runs [2][2]probeResult
			for sourceIndex, label := range []string{"base", "candidate"} {
				for attempt := range 2 {
					status.FreeSamples = append(status.FreeSamples, loom571RequireSpace(t, loom571AttemptFloor))
					location := filepath.Join(expanded, label, current.Probe.ID, fmt.Sprintf("run-%d", attempt+1))
					runs[sourceIndex][attempt] = runProbe(t, sources[sourceIndex], current.Probe, location, sources[1].Source)
				}
			}
			status.Differences = compareArtifacts(runs[0][0].Artifacts, runs[1][0].Artifacts)
			writeJSON(t, filepath.Join(expanded, current.Probe.ID+"-differences.json"), status.Differences)
			writeJSON(t, filepath.Join(expanded, "status", current.CatalogID+".json"), status)
			base := repairPair(t, expanded, "base", current.Probe, current.Probe.Baseline)
			candidate := repairPair(t, expanded, "candidate", current.Probe, current.Probe.Candidate)
			require.Equal(t, base.Inputs, candidate.Inputs, "same captured specimen bytes")
			require.Equal(t, status.Differences, compareArtifacts(base.Artifacts, candidate.Artifacts))
		})
	}
}

func TestLoom571ShardRetainedPilot(t *testing.T) {
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
	rehydrated := filepath.Join(loom571ShardRoot, "rehydrated", fmt.Sprintf("%02d", index))
	require.NoError(t, os.MkdirAll(rehydrated, 0700))
	require.NoError(t, loom571ExtractAndVerify(archive, rehydrated, evidence.Records))
	expandedBytes, err := loom571TreeSize(rehydrated)
	require.NoError(t, err)
	require.Equal(t, evidence.ExpandedBytes, expandedBytes)
	measurements, attemptSamples := loom571MeasureCases(t, cases, rehydrated)
	require.Equal(t, evidence.CaseMeasurements, measurements)
	require.Len(t, evidence.FreeSamples, len(attemptSamples)+4)
	require.Equal(t, attemptSamples, evidence.FreeSamples[:len(attemptSamples)])
	for _, sample := range evidence.FreeSamples {
		require.GreaterOrEqual(t, sample, loom571AttemptFloor)
	}
	require.Equal(t, evidence.MinimumFree, loom571MinimumFree(t, evidence.FreeSamples))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(rehydrated))
	})
	for _, current := range cases {
		current := current
		t.Run(current.Probe.ID, func(t *testing.T) {
			base := repairPair(t, rehydrated, "base", current.Probe, current.Probe.Baseline)
			candidate := repairPair(t, rehydrated, "candidate", current.Probe, current.Probe.Candidate)
			require.Equal(t, base.Inputs, candidate.Inputs)
			differences := compareArtifacts(base.Artifacts, candidate.Artifacts)
			var recorded []artifactDifference
			repairRead(t, filepath.Join(rehydrated, current.Probe.ID+"-differences.json"), &recorded)
			require.Len(t, recorded, len(differences))
			for diffIndex := range differences {
				require.Equal(t, differences[diffIndex], recorded[diffIndex])
			}
		})
	}
}

func TestLoom571ArchiveVerifierControls(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.MkdirAll(filepath.Join(source, "dir"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(source, "a"), []byte("alpha"), 0640))
	require.NoError(t, os.WriteFile(filepath.Join(source, "dir", "b"), []byte("beta"), 0600))
	records, err := loom571CollectRecords(source)
	require.NoError(t, err)
	good := filepath.Join(t.TempDir(), "good.tar.gz")
	_, _, err = loom571WriteArchive(good, source, records)
	require.NoError(t, err)
	require.NoError(t, loom571ExtractAndVerify(good, filepath.Join(t.TempDir(), "good"), records))

	raw, err := os.ReadFile(good)
	require.NoError(t, err)
	raw[len(raw)/2] ^= 0xff
	corrupt := filepath.Join(t.TempDir(), "corrupt.tar.gz")
	require.NoError(t, os.WriteFile(corrupt, raw, 0600))
	require.Error(t, loom571ExtractAndVerify(corrupt, filepath.Join(t.TempDir(), "corrupt"), records))

	missing := filepath.Join(t.TempDir(), "missing.tar.gz")
	_, _, err = loom571WriteArchive(missing, source, records[:len(records)-1])
	require.NoError(t, err)
	require.Error(t, loom571ExtractAndVerify(missing, filepath.Join(t.TempDir(), "missing"), records))

	duplicate := filepath.Join(t.TempDir(), "duplicate.tar.gz")
	file, err := os.OpenFile(duplicate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	require.NoError(t, err)
	require.NoError(t, loom571WriteArchiveTo(file, source, append(records, records[0])))
	require.NoError(t, file.Close())
	require.Error(t, loom571ExtractAndVerify(duplicate, filepath.Join(t.TempDir(), "duplicate"), records))

	traversal := filepath.Join(t.TempDir(), "traversal.tar.gz")
	out, err := os.OpenFile(traversal, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	require.NoError(t, err)
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "../escape", Mode: 0600, Size: 1, Typeflag: tar.TypeReg}))
	_, err = tw.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, out.Close())
	require.Error(t, loom571ExtractAndVerify(traversal, filepath.Join(t.TempDir(), "traversal"), records))

	nonnormal := filepath.Join(t.TempDir(), "nonnormal.tar.gz")
	out, err = os.OpenFile(nonnormal, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	require.NoError(t, err)
	gz = gzip.NewWriter(out)
	tw = tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "dir/../a", Mode: 0600, Size: 1, Typeflag: tar.TypeReg}))
	_, err = tw.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, out.Close())
	require.Error(t, loom571ExtractAndVerify(nonnormal, filepath.Join(t.TempDir(), "nonnormal"), records))

	linkRoot := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Mkdir(linkRoot, 0700))
	require.NoError(t, os.Symlink("../escape", filepath.Join(linkRoot, "escape")))
	_, err = loom571CollectRecords(linkRoot)
	require.Error(t, err)

	specialRoot := filepath.Join(t.TempDir(), "special")
	require.NoError(t, os.Mkdir(specialRoot, 0700))
	specialEntry := filepath.Join(specialRoot, "sticky")
	require.NoError(t, os.Mkdir(specialEntry, 0700))
	require.NoError(t, os.Chmod(specialEntry, 0700|os.ModeSticky))
	_, err = loom571CollectRecords(specialRoot)
	require.Error(t, err)
}
