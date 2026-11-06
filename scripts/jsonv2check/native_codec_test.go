package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNativeCodecExceptionIsExact(t *testing.T) {
	script, err := filepath.Abs("../lint_json_v2.sh")
	require.NoError(t, err)
	checker, err := os.ReadFile("main.go")
	require.NoError(t, err)
	nativeImport := "\"encoding/" + "json\""
	nativeSource := "package specimen\nimport native " + nativeImport + "\nvar _ = native.Marshal\n"
	for _, tc := range []struct {
		name, path, source string
		accepted           bool
	}{
		{"native owner", "execution/selvedge/workflow_encoding.go", nativeSource, true},
		{"sibling", "execution/selvedge/other.go", nativeSource, false},
		{"same leaf elsewhere", "execution/other/workflow_encoding.go", nativeSource, false},
		{"generator", "http/codegen/example.go", nativeSource, false},
		{"determinism still checked", "execution/selvedge/workflow_encoding.go", "package specimen\nimport json \"encoding/json/v2\"\ntype V struct{}\nfunc(V) MarshalJSON()([]byte,error){return json.Marshal(map[string]any{})}\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for name, data := range map[string][]byte{
				tc.path:                       []byte(tc.source),
				"scripts/jsonv2check/main.go": checker,
				"go.mod":                      []byte("module example.com/lintprobe\n\ngo 1.27.3\n"),
			} {
				path := filepath.Join(root, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
				require.NoError(t, os.WriteFile(path, data, 0o600))
			}
			for _, args := range [][]string{{"init", "-q"}, {"add", "."}} {
				cmd := exec.Command("git", args...)
				cmd.Dir = root
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "%s", output)
			}
			cmd := exec.Command("bash", script)
			cmd.Dir = root
			output, err := cmd.CombinedOutput()
			if tc.accepted {
				require.NoError(t, err, "%s", output)
			} else {
				require.Error(t, err, "%s", output)
			}
		})
	}
}
