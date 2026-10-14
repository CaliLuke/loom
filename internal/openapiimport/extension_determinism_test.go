package openapiimport

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderedExtensionsProcessDeterminism(t *testing.T) {
	if path := os.Getenv("LOOM_EXTENSION_TEST_OUTPUT"); path != "" {
		metadata, err := renderedExtensions("schema", map[string]any{
			"x-z": map[string]any{"z": 3, "a": []any{map[string]any{"z": 2, "a": 1}}, "m": map[string]any{"z": true, "a": false}},
			"x-a": nil,
		})
		require.NoError(t, err)
		var output string
		for _, entry := range metadata {
			output += entry.name + "=" + entry.value + "\n"
		}
		require.NoError(t, os.WriteFile(path, []byte(output), 0600))
		return
	}
	executable, err := os.Executable()
	require.NoError(t, err)
	expected := "openapi:schema:extension:x-a=null\n" + `openapi:schema:extension:x-z={"a":[{"a":1,"z":2}],"m":{"a":false,"z":true},"z":3}` + "\n"
	for range 4 {
		path := filepath.Join(t.TempDir(), "extensions.txt")
		command := exec.Command(executable, "-test.run=^TestRenderedExtensionsProcessDeterminism$")
		command.Env = append(os.Environ(), "LOOM_EXTENSION_TEST_OUTPUT="+path)
		output, err := command.CombinedOutput()
		require.NoError(t, err, string(output))
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, expected, string(got))
	}
}

func TestRenderedExtensionsValueAndFailureControls(t *testing.T) {
	for _, scope := range []string{"", "operation", "parameter", "response"} {
		t.Run(scope, func(t *testing.T) {
			got, err := renderedExtensions(scope, map[string]any{"x-value": []any{false, 0, "", nil}})
			require.NoError(t, err)
			require.Len(t, got, 1)
			prefix := "openapi:"
			if scope != "" {
				prefix += scope + ":"
			}
			require.Equal(t, prefix+"extension:x-value", got[0].name)
			require.Equal(t, `[false,0,"",null]`, got[0].value)
		})
	}
	_, err := renderedExtensions("", map[string]any{"x-invalid": make(chan int)})
	require.ErrorContains(t, err, `vendor extension "x-invalid"`)
}
