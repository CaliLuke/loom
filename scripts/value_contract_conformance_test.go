package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

const conformanceFakeLake = `#!/bin/sh
case "$*" in
  'env lean --version')
    if [ "$CONFORMANCE_MODE" = version ]; then echo 'Lean (version 4.0.0, test)'; else echo 'Lean (version 4.34.1, test)'; fi ;;
  'build value_contract_reference')
    case "$CONFORMANCE_MODE" in
      build) exit 1 ;;
      warning) echo 'warning: incomplete reference' ;;
      executable) exit 0 ;;
    esac
    mkdir -p .lake/build/bin
    printf '#!/bin/sh\nexit 0\n' > .lake/build/bin/value_contract_reference
    chmod +x .lake/build/bin/value_contract_reference ;;
  *) echo "unexpected lake command: $*"; exit 2 ;;
esac
`

const conformanceFakeGo = `#!/bin/sh
case "$*" in
  'test -json ./internal/valuecontract -run ^TestLeanConformance$ -count=1 -timeout=10m')
    [ "$LOOM_VALUE_CONFORMANCE" = 1 ] || exit 81
    [ -x "$LOOM_LEAN_REFERENCE" ] || exit 82
    [ -n "$LOOM_VALUE_CONFORMANCE_REPORT" ] || exit 83
    report='{"version":1,"cases":[{"name":"selection","owner":"source-selection","passed":true},{"name":"resolve","owner":"resolution","passed":true},{"name":"project","owner":"projection","passed":true},{"name":"codec","owner":"codec-boundary","passed":true},{"name":"negative","owner":"negative-control","passed":true}]}'
    case "$CONFORMANCE_MODE" in
      zero) report='{"version":1,"cases":[]}' ;;
      coverage) report='{"version":1,"cases":[{"name":"resolve","owner":"resolution","passed":true}]}' ;;
    esac
    if [ "$CONFORMANCE_MODE" != report ]; then printf '%s\n' "$report" > "$LOOM_VALUE_CONFORMANCE_REPORT"; fi
    printf '%s\n' '{"Action":"run","Package":"github.com/CaliLuke/loom/internal/valuecontract","Test":"TestLeanConformance"}'
    case "$CONFORMANCE_MODE" in
      skip) printf '%s\n' '{"Action":"skip","Package":"github.com/CaliLuke/loom/internal/valuecontract","Test":"TestLeanConformance/child"}' ;;
      fail) printf '%s\n' '{"Action":"fail","Package":"github.com/CaliLuke/loom/internal/valuecontract","Test":"TestLeanConformance/child"}'; exit 1 ;;
    esac
    printf '%s\n' '{"Action":"pass","Package":"github.com/CaliLuke/loom/internal/valuecontract","Test":"TestLeanConformance"}' '{"Action":"pass","Package":"github.com/CaliLuke/loom/internal/valuecontract"}'
    if [ "$CONFORMANCE_MODE" = exit ]; then exit 7; fi ;;
  'run ./scripts/valuecontractcheck '*) shift; shift; exec "$CONFORMANCE_VERIFIER" "$@" ;;
  *) echo "unexpected Go command: $*"; exit 2 ;;
esac
`

func TestValueContractConformanceGate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the conformance gate is a Unix contributor command")
	}
	verifier := filepath.Join(t.TempDir(), "verifier")
	output, err := exec.CommandContext(t.Context(), "go", "build", "-o", verifier, "./valuecontractcheck").CombinedOutput()
	require.NoError(t, err, string(output))
	for _, mode := range []string{"success", "version", "build", "warning", "executable", "skip", "fail", "exit", "report", "zero", "coverage", "missing-lake", "missing-go"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			script, err := os.ReadFile("check_value_contract_conformance.sh")
			require.NoError(t, err)
			require.NoError(t, os.Mkdir(filepath.Join(root, "scripts"), 0o700))
			path := filepath.Join(root, "scripts", "check_value_contract_conformance.sh")
			require.NoError(t, os.WriteFile(path, script, 0o700))
			project := filepath.Join(root, "expr", "lean", "value_projection")
			require.NoError(t, os.MkdirAll(project, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(project, "lean-toolchain"), []byte("leanprover/lean4:v4.34.1\n"), 0o600))
			bin := filepath.Join(root, "bin")
			require.NoError(t, os.Mkdir(bin, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "lake"), []byte(conformanceFakeLake), 0o700))
			if mode != "missing-go" {
				require.NoError(t, os.WriteFile(filepath.Join(bin, "go"), []byte(conformanceFakeGo), 0o700))
			}
			env := append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "CONFORMANCE_MODE="+mode, "CONFORMANCE_VERIFIER="+verifier)
			switch mode {
			case "missing-lake":
				env = append(env, "PATH=/usr/bin:/bin")
			case "missing-go":
				env = append(env, "PATH="+bin+":/usr/bin:/bin")
			}
			output, err := proofRun(t, root, env, "bash", path)
			if mode == "success" {
				require.NoError(t, err, output)
				require.Contains(t, output, "value contract conformance passed")
			} else {
				require.Error(t, err, output)
			}
		})
	}
}
