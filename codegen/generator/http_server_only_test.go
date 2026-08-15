package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestGenerateHTTPServerOnlyRemovesClientsAndCompiles(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	root := codegen.RunDSL(t, httpServerOnlyDSL)
	dir := t.TempDir()
	goMod := fmt.Sprintf(
		"module example.com/server-only\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n",
		testingx.RepoRoot(),
	)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

	_, err := Generate(dir, "gen", false)
	require.NoError(t, err)
	clientDir := filepath.Join(dir, codegen.Gendir, "http", "widgets", "client")
	cliDir := filepath.Join(dir, codegen.Gendir, "http", "cli")
	require.DirExists(t, clientDir)
	require.DirExists(t, cliDir)

	if root.API.Meta == nil {
		root.API.Meta = make(map[string][]string)
	}
	root.API.Meta["http:generate"] = []string{"server"}
	_, err = Generate(dir, "gen", false)
	require.NoError(t, err)

	require.DirExists(t, filepath.Join(dir, codegen.Gendir, "http", "widgets", "server"))
	require.FileExists(t, filepath.Join(dir, codegen.Gendir, "widgets", "service.go"))
	require.NoDirExists(t, clientDir)
	require.NoDirExists(t, cliDir)

	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}

func TestTransportRejectsUnsupportedHTTPGenerationMode(t *testing.T) {
	root := codegen.RunDSL(t, httpServerOnlyDSL)
	if root.API.Meta == nil {
		root.API.Meta = make(map[string][]string)
	}
	root.API.Meta["http:generate"] = []string{"client"}

	_, err := Transport("example.com/server-only/gen", []eval.Root{root})

	require.ErrorContains(t, err, `HTTP generation mode "client" must be one of all or server`)
}

func httpServerOnlyDSL() {
	dsl.API("server-only", func() {})
	dsl.Service("widgets", func() {
		dsl.Method("show", func() {
			dsl.Payload(func() {
				dsl.Attribute("id", dsl.String)
				dsl.Required("id")
			})
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.GET("/widgets/{id}")
			})
		})
	})
}
