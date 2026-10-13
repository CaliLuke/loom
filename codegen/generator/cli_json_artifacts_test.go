package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestCLIJSONArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_CLI_JSON_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_CLI_JSON_OUTPUT for isolated generated artifact comparison")
	}
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"body", cliJSONBodyDSL},
		{"unions", unionInUnionDSL},
		{"untagged", cliJSONUntaggedDSL},
		{"field-name", testdata.PayloadBodyCustomNameDSL},
		{"query-control", func() {
			mappingDefaultDSL(false, false)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderDesignModule(t, dir, "example.com/cliinput", tc.design)
		})
	}
}

func TestCLIJSONArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_CLI_JSON_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_CLI_JSON_OUTPUT to build existing comparison modules")
	}
	for _, name := range []string{"body", "unions", "untagged", "field-name", "query-control"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(output, name)
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}
