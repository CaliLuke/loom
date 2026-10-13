package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/stretchr/testify/require"
)

func TestJSONTagArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_JSON_TAG_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_JSON_TAG_OUTPUT for isolated artifact comparison")
	}
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"ordinary", func() {
			jsonTagNormalizationDSL(false)
		}},
		{"full", func() {
			jsonTagNormalizationDSL(true)
		}},
		{"skip", jsonTagSkipDSL},
		{"spec", testdata.OpenAPIAuthoredJSONTagsDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderDesignModule(t, dir, "example.com/tags", tc.design)
		})
	}
}

func TestJSONTagArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_JSON_TAG_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_JSON_TAG_OUTPUT for isolated artifact compilation")
	}
	for _, name := range []string{"ordinary", "full", "skip", "spec"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(output, name)
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				logs, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, logs)
			}
		})
	}
}
