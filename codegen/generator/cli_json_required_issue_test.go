package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/testingx"
)

const cliJSONRequiredHarness = `package clipresence_test

import (
 "testing"

 pc "example.com/clipresence/gen/http/plain/client"
 rc "example.com/clipresence/gen/jsonrpc/rpc/client"
 "github.com/stretchr/testify/require"
)

func TestRequiredJSONMember(t *testing.T) {
 t.Run("http missing required", func(t *testing.T) {
  _, err := pc.BuildReadPayload("{}")
  require.Error(t, err)
 })
 t.Run("jsonrpc missing required", func(t *testing.T) {
  _, err := rc.BuildReadPayload("{}")
  require.Error(t, err)
 })
 t.Run("http supplied zero", func(t *testing.T) {
  payload, err := pc.BuildReadPayload("{\"flag\":false}")
  require.NoError(t, err)
  require.False(t, payload.Flag)
 })
 t.Run("jsonrpc supplied zero", func(t *testing.T) {
  payload, err := rc.BuildReadPayload("{\"flag\":false}")
  require.NoError(t, err)
  require.False(t, payload.Flag)
 })
}
`

func TestCLIJSONRequiredPresence(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("cli_presence", func() {
			dsl.JSONRPC(func() {
			})
		})
		for _, name := range []string{"plain", "rpc"} {
			dsl.Service(name, func() {
				if name == "rpc" {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
				}
				dsl.Method("read", func() {
					dsl.Payload(func() {
						dsl.Attribute("flag", dsl.Boolean, func() {
							dsl.Default(true)
						})
						dsl.Required("flag")
					})
					if name == "rpc" {
						dsl.JSONRPC(func() {
						})
					} else {
						dsl.HTTP(func() {
							dsl.POST("/read")
						})
					}
				})
			})
		}
	})
	dir := t.TempDir()
	roots := []eval.Root{root}
	serviceFiles, err := Service("example.com/clipresence/gen", roots)
	require.NoError(t, err)
	transportFiles, err := Transport("example.com/clipresence/gen", roots)
	require.NoError(t, err)
	for _, file := range mergeFilesByPath(append(serviceFiles, transportFiles...)) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	module := fmt.Sprintf("module example.com/clipresence\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427\n\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cli_test.go"), []byte(cliJSONRequiredHarness), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-v", "-count=1", "."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}
