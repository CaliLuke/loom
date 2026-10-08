package testdatacompile

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/loomsource"
	"github.com/CaliLuke/loom/internal/testprocess"
)

type (
	expectation struct {
		Phase    string `json:"phase"`
		Contains string `json:"contains"`
		Reason   string `json:"reason"`
		Issue    string `json:"issue,omitempty"`
	}
	observation struct {
		ID     string `json:"id"`
		Phase  string `json:"phase"`
		Output string `json:"output,omitempty"`
	}
	command struct {
		phase string
		args  []string
	}
)

// TestDesigns runs the expensive compile tier explicitly through Make/CI.
// Each subtest owns one temporary module; -parallel bounds both disk use and
// child process concurrency. All cases share the caller's Go caches.
func TestDesigns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the full compile tier runs on macOS and Linux")
	}
	if os.Getenv("LOOM_TESTDATA_COMPILE") != "1" {
		t.Skip("run make test-testdata-compile to compile every exported testdata design")
	}
	for _, tool := range []string{"go", "protoc", "protoc-gen-go", "protoc-gen-go-grpc"} {
		_, err := exec.LookPath(tool)
		require.NoError(t, err, "install prerequisite %s with make depend", tool)
	}
	root, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	suite := t.TempDir()
	source, err := loomsource.Resolve(root, filepath.Join(suite, "source"))
	require.NoError(t, err)
	designs := discover(t, source)
	seen := make(map[string]bool, len(designs))
	for _, d := range designs {
		seen[d.id] = true
	}
	expected := validateExpectations(t, filepath.Join(root, "internal", "testdatacompile", "expectations.json"), seen)
	loom := filepath.Join(suite, "loom")
	out, err := runCommand(t.Context(), source, nil, "go", "build", "-o", loom, "./cmd/loom")
	require.NoError(t, err, "%s", out)
	suiteCtx, abort := context.WithCancelCause(t.Context())
	t.Cleanup(func() {
		abort(nil)
	})
	sum, err := os.ReadFile(filepath.Join(source, "go.sum"))
	require.NoError(t, err)
	for _, d := range designs {
		t.Run(d.id, func(t *testing.T) {
			t.Parallel()
			if cause := context.Cause(suiteCtx); cause != nil {
				t.Skipf("corpus aborted after infrastructure failure: %v", cause)
			}
			dir := t.TempDir()
			module := "example.com/specimen"
			require.NoError(t, os.Mkdir(filepath.Join(dir, "design"), 0o700))
			gomod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %q\n", module, source)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o600))
			// Imported fixture packages can register shared schemes during init.
			// Match the fresh evaluation state used by the direct DSL tests while
			// retaining the fixture's references to those shared expressions and
			// the default runtime root pointers that the CLI registers afterward.
			wrapper := fmt.Sprintf(`package design

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	specimen %q
)

func init() {
	*expr.Root = expr.RootExpr{}
	*expr.GeneratedResultTypes = expr.ResultTypesRoot{}
	eval.Reset()
	specimen.%s()
}
`, d.importPath, d.name)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte(wrapper), 0o600))
			// loom gen invokes protoc and both Go plugins for every emitted
			// protobuf file. Any protoc failure therefore fails the gen phase.
			commands := []command{
				{phase: "gen", args: []string{loom, "gen", module + "/design"}},
				{phase: "example", args: []string{loom, "example", module + "/design"}},
				{phase: "tidy", args: []string{"go", "mod", "tidy"}},
				{phase: "build", args: []string{"go", "build", "./..."}},
				{phase: "vet", args: []string{"go", "vet", "./..."}},
			}
			got := observation{ID: d.id}
			for _, cmd := range commands {
				ctx, cancel := context.WithTimeout(suiteCtx, 2*time.Minute)
				out, err := runCommand(ctx, dir, []string{"LOOM_DIR=" + source}, cmd.args...)
				timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
				cancel()
				if err != nil {
					got.Phase = cmd.phase
					got.Output = fmt.Sprintf("%v\n%s", err, out)
					if timedOut || strings.Contains(got.Output, "no space left on device") {
						abort(fmt.Errorf("%s: %s infrastructure failure: %s", d.id, cmd.phase, got.Output))
						recordObservation(t, got)
						t.Fatalf("%v", context.Cause(suiteCtx))
					}
					break
				}
			}
			recordObservation(t, got)
			want, known := expected[d.id]
			if !known {
				require.Empty(t, got.Phase, "%s failed:\n%s", got.Phase, got.Output)
				return
			}
			require.NotEmpty(t, got.Phase, "unexpected pass: remove the stale expectation for %s (%s)", d.id, want.Issue)
			require.Equal(t, want.Phase, got.Phase, "%s", got.Output)
			require.Contains(t, got.Output, want.Contains)
			if want.Issue == "" {
				require.Regexp(t, `stage eval\.(Context\.Errors|RunDSL)`, got.Output)
			}
			t.Logf("verified expected failure: %s %s", want.Reason, want.Issue)
		})
	}
}

func validateExpectations(t *testing.T, path string, designs map[string]bool) map[string]expectation {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var expected map[string]expectation
	require.NoError(t, json.Unmarshal(data, &expected, json.RejectUnknownMembers(true)))
	for id, want := range expected {
		require.True(t, designs[id], "stale expectation for missing design %s", id)
		require.Contains(t, []string{"gen", "example", "tidy", "build", "vet"}, want.Phase, "%s", id)
		require.NotEmpty(t, want.Contains, "%s", id)
		require.NotEmpty(t, want.Reason, "%s", id)
		if want.Issue == "" {
			require.Equal(t, "intentional validation failure", want.Reason, "%s requires a tracking issue", id)
			require.Equal(t, "gen", want.Phase)
		} else {
			require.Regexp(t, `^https://github.com/CaliLuke/loom/issues/[0-9]+$`, want.Issue)
		}
	}
	return expected
}

func recordObservation(t *testing.T, got observation) {
	t.Helper()
	if dir := os.Getenv("LOOM_TESTDATA_RESULTS"); dir != "" {
		require.NoError(t, os.MkdirAll(dir, 0o700))
		data, err := json.Marshal(got)
		require.NoError(t, err)
		name := strings.ReplaceAll(got.ID, "/", "_") + ".json"
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0o600))
	}
}

func runCommand(ctx context.Context, dir string, extra []string, args ...string) ([]byte, error) {
	cmd := testprocess.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOMAXPROCS=2")
	cmd.Env = append(cmd.Env, extra...)
	return cmd.CombinedOutput()
}
