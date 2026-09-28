package valuecontract

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/loomsource"
	"github.com/CaliLuke/loom/internal/testprocess"
)

type (
	comparisonManifest struct {
		raw []byte
		// Baseline pins the source revision whose characterized outcomes are expected.
		Baseline string `json:"baseline"`
		// Probes lists the common designs generated under both revisions.
		Probes []probe `json:"probes"`
		// Differences lists the reviewed exact-byte changes between revisions.
		Differences []allowedDifference `json:"intended_differences"`
	}
	probe struct {
		// ID identifies the probe in retained evidence.
		ID string `json:"id"`
		// Design names a common standalone DSL source under testdata.
		Design string `json:"design,omitempty"`
		// Fixture names a checked-in fixture whose design is regenerated.
		Fixture string `json:"fixture,omitempty"`
		// Import names an existing exported specimen package.
		Import string `json:"import,omitempty"`
		// Function names the exported specimen DSL entry point.
		Function string `json:"function,omitempty"`
		// Coverage states the contract surfaces exercised by the probe.
		Coverage []string `json:"coverage"`
		// Decoder enables the generated advertised-example assertion harness.
		Decoder string `json:"decoder,omitempty"`
		// Transport selects the generated client package used by the decoder.
		Transport string `json:"transport,omitempty"`
		// ExpectedJSON lists authored service fields that decoding must preserve.
		ExpectedJSON string `json:"expected_json,omitempty"`
		// Baseline records the characterized outcome of the declared baseline revision.
		Baseline expectation `json:"baseline"`
		// Candidate records the reviewed expected outcome of the proposed source.
		Candidate expectation `json:"candidate"`
	}
	revision struct {
		// Input records the caller-supplied source reference.
		Input string `json:"input"`
		// Source is the validated local path used by temporary modules.
		Source string `json:"source"`
		// Commit records the exact source commit identity.
		Commit string `json:"commit"`
		// ContentID identifies the immutable local content snapshot, including dirty files.
		ContentID string `json:"content_id,omitempty"`
		// OriginalSource records the live path before snapshot capture.
		OriginalSource string `json:"original_source,omitempty"`
		// Binary is the temporary revision-specific CLI executable.
		Binary string `json:"-"`
	}
	probeResult struct {
		// ID identifies the probe in retained evidence.
		ID string `json:"id"`
		// Phases retains each attempted command outcome separately.
		Phases []phaseResult `json:"phases"`
		// Inputs records the common design and handwritten fixture bytes.
		Inputs map[string]string `json:"inputs"`
		// Artifacts maps retained relative paths to exact-byte SHA-256 hashes.
		Artifacts map[string]string `json:"artifacts"`
		// Observation retains advertised JSON and decoded service values.
		Observation jsontext.Value `json:"observation,omitempty"`
	}
)

// TestCompareRevisions is opt-in because each revision generates and compiles
// the full common probe corpus in two independent processes.
func TestCompareRevisions(t *testing.T) {
	base, candidate, results := os.Getenv("LOOM_VALUE_BASE"), os.Getenv("LOOM_VALUE_CANDIDATE"), os.Getenv("LOOM_VALUE_RESULTS")
	if base == "" && candidate == "" && results == "" {
		t.Skip("set LOOM_VALUE_BASE, LOOM_VALUE_CANDIDATE and LOOM_VALUE_RESULTS")
	}
	require.NotEmpty(t, base)
	require.NotEmpty(t, candidate)
	require.NotEmpty(t, results)
	require.True(t, filepath.IsAbs(results), "results path must be absolute")
	for _, tool := range []string{"go", "protoc", "protoc-gen-go", "protoc-gen-go-grpc"} {
		_, err := exec.LookPath(tool)
		require.NoError(t, err, "install prerequisite %s with make depend", tool)
	}
	// Refuse an existing result directory: evidence from distinct runs must not mix.
	require.NoError(t, os.MkdirAll(filepath.Dir(results), 0700))
	require.NoError(t, os.Mkdir(results, 0700))
	root, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	manifest := readManifest(t)
	require.NoError(t, validateBaseline(manifest.Baseline, base))
	resolved := []revision{resolveRevision(t, root, base), resolveRevision(t, root, candidate)}
	commonRoot := filepath.Join(results, "common-source")
	commonFiles := captureGitSource(t, root, commonRoot)
	commonID, err := snapshotIdentity(commonFiles)
	require.NoError(t, err)
	writeJSON(t, filepath.Join(results, "common-source.json"), map[string]any{"original_source": root, "content_id": commonID, "excluded": []string{".git metadata", ".claude agent-only pointers"}})
	t.Cleanup(func() {
		require.NoError(t, verifySourceSnapshot(commonRoot, commonFiles))
	})
	if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
		original := resolved[1].Source
		resolved[1].OriginalSource = original
		if filepath.Clean(original) == filepath.Clean(root) {
			resolved[1].Source = commonRoot
			resolved[1].ContentID = commonID
		} else {
			snapshot := filepath.Join(results, "candidate-source")
			files := captureGitSource(t, original, snapshot)
			identity, err := snapshotIdentity(files)
			require.NoError(t, err)
			resolved[1].Source, resolved[1].ContentID = snapshot, identity
			t.Cleanup(func() {
				require.NoError(t, verifySourceSnapshot(snapshot, files))
			})
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(results, "selected-manifest.json"), manifest.raw, 0600))
	labels := []string{"base", "candidate"}
	for i := range resolved {
		resolved[i].Binary = filepath.Join(t.TempDir(), "loom")
		built := runPhase(t.Context(), resolved[i].Source, nil, "build-generator", "go", "build", "-o", resolved[i].Binary, "./cmd/loom")
		writeJSON(t, filepath.Join(results, labels[i]+"-source.json"), resolved[i])
		writeJSON(t, filepath.Join(results, labels[i]+"-generator.json"), built)
		require.Zero(t, built.Exit, "generator infrastructure: %s", built.Output)
	}
	for _, p := range manifest.Probes {
		t.Run(p.ID, func(t *testing.T) {
			var outputs [2]probeResult
			for i, source := range resolved {
				var repeated [2]probeResult
				for attempt := range 2 {
					location := filepath.Join(results, labels[i], p.ID, fmt.Sprintf("run-%d", attempt+1))
					repeated[attempt] = runProbe(t, source, p, location, commonRoot)
					expected := p.Baseline
					if i == 1 {
						expected = p.Candidate
					}
					if failure := firstFailure(repeated[attempt]); checkOutcome(expected, failure) != nil {
						t.Errorf("%s run %d: %v", labels[i], attempt+1, checkOutcome(expected, failure))
					}
				}
				require.Equal(t, repeated[0].Inputs, repeated[1].Inputs, "common inputs changed during comparison")
				require.Empty(t, compareArtifacts(repeated[0].Artifacts, repeated[1].Artifacts), "%s is nondeterministic", labels[i])
				require.Equal(t, string(repeated[0].Observation), string(repeated[1].Observation), "%s decoder observation is nondeterministic", labels[i])
				outputs[i] = repeated[0]
			}
			require.Equal(t, outputs[0].Inputs, outputs[1].Inputs, "baseline and candidate must use identical common inputs")
			differences := compareArtifacts(outputs[0].Artifacts, outputs[1].Artifacts)
			writeJSON(t, filepath.Join(results, p.ID+"-differences.json"), differences)
			require.NoError(t, checkDifferences(p.ID, differences, manifest.Differences))
		})
	}
}

func readManifest(t *testing.T) comparisonManifest {
	t.Helper()
	path := os.Getenv("LOOM_VALUE_MANIFEST")
	if path == "" {
		path = "manifest.json"
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var manifest comparisonManifest
	require.NoError(t, json.Unmarshal(data, &manifest, json.RejectUnknownMembers(true)))
	manifest.raw = data
	require.Regexp(t, `^[a-f0-9]{40}$`, manifest.Baseline)
	require.NotEmpty(t, manifest.Probes)
	seen := make(map[string]bool)
	for _, p := range manifest.Probes {
		require.Regexp(t, `^[a-z][a-z0-9-]+$`, p.ID)
		require.False(t, seen[p.ID], "duplicate probe %s", p.ID)
		seen[p.ID] = true
		sources := 0
		for _, source := range []string{p.Design, p.Fixture, p.Import} {
			if source != "" {
				sources++
			}
		}
		require.Equal(t, 1, sources, "probe %s must have one source", p.ID)
		require.NotEmpty(t, p.Coverage)
		if p.Design != "" {
			require.FileExists(t, filepath.Join("testdata", p.Design))
		}
		if p.Import != "" {
			require.NotEmpty(t, p.Function)
		}
		for _, expected := range []expectation{p.Baseline, p.Candidate} {
			if expected.Phase != "" {
				require.Contains(t, []string{"gen", "example", "build", "vet", "decoder"}, expected.Phase)
				require.NotEmpty(t, expected.Contains)
				require.Regexp(t, `^https://github.com/CaliLuke/loom/issues/[0-9]+$`, expected.Issue)
			}
		}
	}
	for _, difference := range manifest.Differences {
		require.True(t, seen[difference.Probe])
		require.NotEmpty(t, difference.Reason)
	}
	return manifest
}

// TestComparisonManifest validates probe sources and explicit failure expectations.
func TestComparisonManifest(t *testing.T) {
	readManifest(t)
}

func resolveRevision(t *testing.T, root, input string) revision {
	t.Helper()
	var resolved revision
	resolved.Input = input
	if info, err := os.Stat(input); err == nil && info.IsDir() {
		path, err := filepath.Abs(input)
		require.NoError(t, err)
		// Use the repository's shared source validation with an explicit local override.
		t.Setenv("LOOM_DIR", path)
		resolved.Source, err = loomsource.Resolve(root, filepath.Join(t.TempDir(), "unused"))
		require.NoError(t, err)
		result := runPhase(t.Context(), path, nil, "revision", "git", "rev-parse", "HEAD")
		require.Zero(t, result.Exit, result.Output)
		resolved.Commit = strings.TrimSpace(result.Output)
		return resolved
	}
	require.Regexp(t, `^[a-f0-9]{40}$`, input, "remote sources must be full pushed commit IDs")
	result := runPhase(t.Context(), root, nil, "download", "go", "mod", "download", "-json", "github.com/CaliLuke/loom@"+input)
	require.Zero(t, result.Exit, "source infrastructure: %s", result.Output)
	var module struct {
		// Dir is the downloaded module cache directory.
		Dir string
		// Origin records the source commit reported by the Go module downloader.
		Origin struct {
			// Hash is the exact downloaded commit identity.
			Hash string
		}
	}
	require.NoError(t, json.Unmarshal([]byte(result.Output), &module))
	require.Equal(t, input, module.Origin.Hash, "download must resolve the exact requested commit")
	t.Setenv("LOOM_DIR", module.Dir)
	var err error
	resolved.Source, err = loomsource.Resolve(root, filepath.Join(t.TempDir(), "unused"))
	require.NoError(t, err)
	resolved.Commit = input
	return resolved
}

func runPhase(parent context.Context, dir string, extra []string, phase string, args ...string) phaseResult {
	ctx, cancel := context.WithTimeout(parent, 3*time.Minute)
	defer cancel()
	cmd := testprocess.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOMAXPROCS=2")
	cmd.Env = append(cmd.Env, extra...)
	output, err := cmd.CombinedOutput()
	result := phaseResult{Phase: phase, Output: string(output)}
	if err != nil {
		result.Exit = 1
		result.Output = err.Error() + "\n" + result.Output
		result.Infrastructure = ctx.Err() != nil || strings.Contains(result.Output, "no space left on device") || strings.Contains(result.Output, "executable file not found")
	}
	return result
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value, json.Deterministic(true))
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, append(data, '\n'), 0600))
}

func firstFailure(result probeResult) phaseResult {
	for _, phase := range result.Phases {
		if phase.Exit != 0 {
			return phase
		}
	}
	return phaseResult{}
}

func validateBaseline(declared, requested string) error {
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(requested) {
		return fmt.Errorf("LOOM_VALUE_BASE must name a full published commit ID, got %q", requested)
	}
	if declared != requested {
		return fmt.Errorf("manifest baseline %s does not match LOOM_VALUE_BASE %s; select an explicitly reviewed manifest", declared, requested)
	}
	return nil
}
