package scripts_test

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const proofFakeLake = `#!/bin/sh
if [ -n "${LOOM_PROOF_COMMAND_LOG:-}" ]; then
  printf '%s\n' "$*" >> "$LOOM_PROOF_COMMAND_LOG"
fi
case "$*" in
  'env lean --version')
    if [ "$LOOM_PROOF_TEST_MODE" = version ]; then
      echo 'Lean (version 4.0.0, test)'
    else
      echo 'Lean (version 4.34.1, test)'
    fi ;;
  build)
    case "$LOOM_PROOF_TEST_MODE" in
      build-error) exit 1 ;;
      warning) echo 'warning: ignored proof obligation' ;;
    esac ;;
  'build ValueContract.Issue456CollisionDiagnostics')
    case "$LOOM_PROOF_TEST_MODE" in
      issue456-build-error) exit 1 ;;
      issue456-warning) echo 'warning: ignored issue 456 proof obligation' ;;
    esac ;;
	'env lean -DwarningAsError=true NegativeNearestPattern.lean'|\
	'env lean -DwarningAsError=true NegativeNearestFormat.lean'|\
	'env lean -DwarningAsError=true NegativeNumericOverwrite.lean'|\
	'env lean -DwarningAsError=true NegativeNumericClosedTie.lean'|\
	'env lean -DwarningAsError=true NegativeEnumOverride.lean')
		if [ "${LOOM_PROOF_NEGATIVE_SUCCEEDS:-}" = "$4" ]; then exit 0; fi
		if [ "$LOOM_PROOF_TEST_MODE" = negative-fails ] && [ "$4" = NegativeNearestPattern.lean ]; then
			echo 'error: unknown identifier'; exit 1
		fi
		case "$4" in
			NegativeNumericOverwrite.lean)
				echo 'error: unsolved goals'; echo '⊢ False' ;;
			*)
				echo 'error: proved that the proposition'; echo 'is false' ;;
		esac
		exit 1 ;;
  'env lean -DwarningAsError=true ValueContract/AxiomAudit.lean')
    case "$LOOM_PROOF_TEST_MODE" in
      axiom) echo 'error: forbidden transitive axiom custom'; exit 1 ;;
      empty-audit) exit 0 ;;
    esac
    echo 'VALUE_CONTRACT_THEOREM ValueContract.Legacy.first axioms=[]'
    if [ "$LOOM_PROOF_TEST_MODE" != missing-theorem ]; then
      echo 'VALUE_CONTRACT_THEOREM ValueContract.Legacy.second axioms=[]'
    fi
    echo 'VALUE_CONTRACT_AUDIT_OK 2' ;;
  'env lean -DwarningAsError=true ValueContract/AxiomAudit456.lean')
    case "$LOOM_PROOF_TEST_MODE" in
      issue456-axiom) echo 'error: forbidden transitive axiom custom'; exit 1 ;;
      empty-issue456-audit) exit 0 ;;
    esac
    echo 'ISSUE456_THEOREM ValueContract.Candidate.Issue456CollisionDiagnostics.first axioms=[]'
    if [ "$LOOM_PROOF_TEST_MODE" != missing-issue456-theorem ]; then
      echo 'ISSUE456_THEOREM ValueContract.Candidate.Issue456CollisionDiagnostics.second axioms=[]'
    fi
    echo 'ISSUE456_AUDIT_OK 2' ;;
  'env leanchecker --fresh ValueContract.Proofs')
    case "$LOOM_PROOF_TEST_MODE" in
      kernel) exit 1 ;;
      missing-checker) echo 'leanchecker: command not found'; exit 127 ;;
    esac ;;
  'env leanchecker --fresh ValueContract.Issue456CollisionDiagnostics')
    case "$LOOM_PROOF_TEST_MODE" in
      issue456-kernel) exit 1 ;;
      missing-issue456-checker) echo 'leanchecker: command not found'; exit 127 ;;
    esac ;;
  *) echo "unexpected command: $*"; exit 2 ;;
esac
`

func TestValueContractProofGate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the proof gate is a Unix contributor command")
	}
	for _, tc := range []struct {
		name           string
		mode           string
		negativeSource string
		want           string
	}{
		{name: "success"},
		{name: "build error", mode: "build-error", want: "proof build failed"},
		{name: "warning", mode: "warning", want: "proof build emitted a warning"},
		{name: "negative control unexpected failure", mode: "negative-fails", want: "negative proof control failed for an unexpected reason"},
		{name: "nearest pattern control succeeds", negativeSource: "NegativeNearestPattern.lean", want: "negative proof control unexpectedly succeeded: NegativeNearestPattern.lean"},
		{name: "nearest format control succeeds", negativeSource: "NegativeNearestFormat.lean", want: "negative proof control unexpectedly succeeded: NegativeNearestFormat.lean"},
		{name: "numeric overwrite control succeeds", negativeSource: "NegativeNumericOverwrite.lean", want: "negative proof control unexpectedly succeeded: NegativeNumericOverwrite.lean"},
		{name: "numeric closed tie control succeeds", negativeSource: "NegativeNumericClosedTie.lean", want: "negative proof control unexpectedly succeeded: NegativeNumericClosedTie.lean"},
		{name: "enum override control succeeds", negativeSource: "NegativeEnumOverride.lean", want: "negative proof control unexpectedly succeeded: NegativeEnumOverride.lean"},
		{name: "wrong toolchain", mode: "version", want: "expected Lean 4.34.1"},
		{name: "axiom rejected", mode: "axiom", want: "forbidden transitive axiom"},
		{name: "empty audit", mode: "empty-audit", want: "missing audit completion"},
		{name: "missing theorem", mode: "missing-theorem", want: "missing theorem report"},
		{name: "kernel failure", mode: "kernel", want: "fresh kernel replay failed"},
		{name: "missing checker", mode: "missing-checker", want: "leanchecker: command not found"},
		{name: "issue 456 build error", mode: "issue456-build-error", want: "issue 456 proof build failed"},
		{name: "issue 456 build warning", mode: "issue456-warning", want: "issue 456 proof build emitted a warning"},
		{name: "issue 456 axiom rejected", mode: "issue456-axiom", want: "forbidden transitive axiom"},
		{name: "empty issue 456 audit", mode: "empty-issue456-audit", want: "missing audit completion for the issue 456 theorem manifest"},
		{name: "missing issue 456 theorem", mode: "missing-issue456-theorem", want: "missing issue 456 theorem report"},
		{name: "issue 456 kernel failure", mode: "issue456-kernel", want: "issue 456 fresh kernel replay failed"},
		{name: "missing issue 456 checker", mode: "missing-issue456-checker", want: "leanchecker: command not found"},
		{name: "missing lake", mode: "missing-lake", want: "missing lake"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := proofTestTree(t, false)
			bin := filepath.Join(root, "bin")
			commandLog := filepath.Join(root, "commands.log")
			require.NoError(t, os.Mkdir(bin, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "lake"), []byte(proofFakeLake), 0o700))
			env := append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "LOOM_PROOF_TEST_MODE="+tc.mode, "LOOM_PROOF_NEGATIVE_SUCCEEDS="+tc.negativeSource, "LOOM_PROOF_COMMAND_LOG="+commandLog)
			if tc.mode == "missing-lake" {
				// Use an allowlist, not host system paths: distro elan installs
				// lake in /usr/bin. dirname is the only external command the
				// script needs before checking whether lake is available.
				require.NoError(t, os.Remove(filepath.Join(bin, "lake")))
				dirname, err := exec.LookPath("dirname")
				require.NoError(t, err)
				require.NoError(t, os.Symlink(dirname, filepath.Join(bin, "dirname")))
				env = append(os.Environ(), "PATH="+bin)
			}
			output, err := proofRun(t, root, env, "bash", filepath.Join(root, "scripts", "check_value_contract_proof.sh"))
			if tc.want != "" {
				require.Error(t, err, output)
				require.Contains(t, output, tc.want)
				return
			}
			require.NoError(t, err, output)
			require.Contains(t, output, "value contract proof gate passed")
			commands, err := os.ReadFile(commandLog)
			require.NoError(t, err)
			for _, source := range []string{
				"NegativeNearestPattern.lean",
				"NegativeNearestFormat.lean",
				"NegativeNumericOverwrite.lean",
				"NegativeNumericClosedTie.lean",
				"NegativeEnumOverride.lean",
			} {
				require.Contains(t, string(commands), "env lean -DwarningAsError=true "+source+"\n")
			}
			require.Contains(t, string(commands), "env lean -DwarningAsError=true ValueContract/AxiomAudit456.lean\n")
			require.Contains(t, string(commands), "build ValueContract.Issue456CollisionDiagnostics\n")
			require.Contains(t, string(commands), "env leanchecker --fresh ValueContract.Issue456CollisionDiagnostics\n")
		})
	}
}

// TestValueContractProofAuditNegativeControls checks actual imported proof bodies.
// Ordinary Go tests exercise orchestration above; this external-tool tier is
// mandatory when changing the proof gate and fails if explicitly enabled without Lean.
func TestValueContractProofAuditNegativeControls(t *testing.T) {
	if os.Getenv("LOOM_VALUE_PROOF_TEST") != "1" {
		t.Skip("set LOOM_VALUE_PROOF_TEST=1 to run actual Lean axiom rejection controls")
	}
	_, err := exec.LookPath("lake")
	require.NoError(t, err, "enabled proof controls require the pinned Lean toolchain")
	for _, tc := range []struct {
		name    string
		body    string
		missing bool
		want    string
	}{
		{
			name: "admitted dependency",
			body: "set_option warningAsError false in\ntheorem gateDependency : False := by sorry\n",
			want: "forbidden transitive axiom sorryAx",
		},
		{
			name: "custom axiom dependency",
			body: "axiom gateCorrectness : False\ntheorem gateDependency : False := gateCorrectness\n",
			want: "forbidden transitive axiom ValueContract.Legacy.gateCorrectness",
		},
		{
			name:    "missing required theorem",
			missing: true,
			want:    "ValueContract.Legacy.gateRequired",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := proofTestTree(t, true)
			project := filepath.Join(root, "expr", "lean", "value_projection")
			legacy := filepath.Join(project, "ValueContract", "Legacy.lean")
			original, err := os.ReadFile(legacy)
			require.NoError(t, err)
			injected := tc.body + "theorem gateRequired : False := gateDependency\n\nend ValueContract.Legacy"
			if tc.missing {
				injected = "end ValueContract.Legacy"
			}
			require.NoError(t, os.WriteFile(legacy, []byte(strings.Replace(string(original), "end ValueContract.Legacy", injected, 1)), 0o600))
			manifest := filepath.Join(project, "required-theorems.txt")
			names, err := os.ReadFile(manifest)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(manifest, append(names, []byte("ValueContract.Legacy.gateRequired\n")...), 0o600))
			output, err := proofRun(t, project, os.Environ(), "lake", "build")
			require.NoError(t, err, output)
			// The dependency is now imported: the audit must traverse its body,
			// rather than relying on a source warning or a direct axiom declaration.
			output, err = proofRun(t, project, os.Environ(), "lake", "env", "lean", "-DwarningAsError=true", "ValueContract/AxiomAudit.lean")
			require.Error(t, err, output)
			require.Contains(t, output, tc.want)
			output, err = proofRun(t, root, os.Environ(), "bash", filepath.Join(root, "scripts", "check_value_contract_proof.sh"))
			require.Error(t, err, output)
		})
	}
}

func TestValueContractProofIssue456AuditNegativeControls(t *testing.T) {
	if os.Getenv("LOOM_VALUE_PROOF_TEST") != "1" {
		t.Skip("set LOOM_VALUE_PROOF_TEST=1 to run actual Lean axiom rejection controls")
	}
	_, err := exec.LookPath("lake")
	require.NoError(t, err, "enabled proof controls require the pinned Lean toolchain")
	for _, tc := range []struct {
		name    string
		body    string
		missing bool
		want    string
	}{
		{
			name: "admitted dependency",
			body: "set_option warningAsError false in\ntheorem gateIssue456Dependency : False := by sorry\n",
			want: "forbidden transitive axiom sorryAx",
		},
		{
			name: "custom axiom dependency",
			body: "axiom gateIssue456Correctness : False\ntheorem gateIssue456Dependency : False := gateIssue456Correctness\n",
			want: "forbidden transitive axiom ValueContract.Candidate.Issue456CollisionDiagnostics.gateIssue456Correctness",
		},
		{
			name:    "missing required theorem",
			missing: true,
			want:    "ValueContract.Candidate.Issue456CollisionDiagnostics.gateIssue456Required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := proofTestTree(t, true)
			project := filepath.Join(root, "expr", "lean", "value_projection")
			source := filepath.Join(project, "ValueContract", "Issue456CollisionDiagnostics.lean")
			original, err := os.ReadFile(source)
			require.NoError(t, err)
			injected := tc.body + "theorem gateIssue456Required : False := gateIssue456Dependency\n\nend ValueContract.Candidate.Issue456CollisionDiagnostics"
			if tc.missing {
				injected = "end ValueContract.Candidate.Issue456CollisionDiagnostics"
			}
			require.NoError(t, os.WriteFile(source, []byte(strings.Replace(string(original), "end ValueContract.Candidate.Issue456CollisionDiagnostics", injected, 1)), 0o600))
			manifest := filepath.Join(project, "required-theorems-456.txt")
			names, err := os.ReadFile(manifest)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(manifest, append(names, []byte("ValueContract.Candidate.Issue456CollisionDiagnostics.gateIssue456Required\n")...), 0o600))
			output, err := proofRun(t, project, os.Environ(), "lake", "build", "ValueContract.Issue456CollisionDiagnostics")
			require.NoError(t, err, output)
			output, err = proofRun(t, project, os.Environ(), "lake", "env", "lean", "-DwarningAsError=true", "ValueContract/AxiomAudit456.lean")
			require.Error(t, err, output)
			require.Contains(t, output, tc.want)
			output, err = proofRun(t, root, os.Environ(), "bash", filepath.Join(root, "scripts", "check_value_contract_proof.sh"))
			require.Error(t, err, output)
		})
	}
}

func proofTestTree(t *testing.T, full bool) string {
	t.Helper()
	root := t.TempDir()
	script, err := os.ReadFile("check_value_contract_proof.sh")
	require.NoError(t, err)
	require.NoError(t, os.Mkdir(filepath.Join(root, "scripts"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scripts", "check_value_contract_proof.sh"), script, 0o700))
	project := filepath.Join(root, "expr", "lean", "value_projection")
	require.NoError(t, os.MkdirAll(project, 0o700))
	if !full {
		require.NoError(t, os.WriteFile(filepath.Join(project, "lean-toolchain"), []byte("leanprover/lean4:v4.34.1\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(project, "required-theorems.txt"), []byte("ValueContract.Legacy.first\nValueContract.Legacy.second\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(project, "required-theorems-456.txt"), []byte("ValueContract.Candidate.Issue456CollisionDiagnostics.first\nValueContract.Candidate.Issue456CollisionDiagnostics.second\n"), 0o600))
		return root
	}
	source := filepath.Join("..", "expr", "lean", "value_projection")
	require.NoError(t, filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == ".lake" {
			return filepath.SkipDir
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(project, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	}))
	return root
}

func proofRun(t *testing.T, dir string, env []string, name string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
}
