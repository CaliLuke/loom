package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

const fileOnlyCLIHarness = `package files_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	cli "example.com/files/gen/http/cli/files"
)

func TestNoEndpoints(t *testing.T) {
	args := os.Args
	t.Cleanup(func() {
		os.Args = args
	})
	os.Args = []string{"files-cli"}
	endpoint, payload, err := cli.ParseEndpoint("http", "localhost", nil, nil, nil, false)
	require.ErrorContains(t, err, "no HTTP endpoints are available")
	require.Nil(t, endpoint)
	require.Nil(t, payload)
}
`

func TestCLIFileOnlyServiceCompile(t *testing.T) {
	codegen.RunDSL(t, func() {
		dsl.API("files", func() {})
		testdata.FileServiceDSL()
	})
	dir := t.TempDir()
	goMod := fmt.Sprintf("module example.com/files\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	_, err := Generate(dir, "gen", false)
	require.NoError(t, err)
	_, err = Generate(dir, "example", false)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cli_test.go"), []byte(fileOnlyCLIHarness), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-count=1", "."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}
