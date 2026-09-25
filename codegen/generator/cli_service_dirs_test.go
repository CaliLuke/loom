package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/naming"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestCLIServiceTransportDirs generates a service named cli, whose transport
// packages share gen/<transport>/cli with the client CLI package of each
// server, and checks that the directories there are the ones that expr
// validation reserves: naming.TransportServiceDirs and the server directory.
// The generated module must compile.
func TestCLIServiceTransportDirs(t *testing.T) {
	cases := []struct {
		Name       string
		Transports []string
	}{
		{Name: "http and grpc", Transports: []string{"http", "grpc"}},
		{Name: "jsonrpc", Transports: []string{"jsonrpc"}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			codegen.RunDSL(t, cliServiceDSL(c.Transports))
			dir := t.TempDir()
			goMod := fmt.Sprintf(
				"module example.com/clidirs\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n",
				testingx.RepoRoot(),
			)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)

			for _, transport := range c.Transports {
				want := append(naming.TransportServiceDirs(transport), naming.ServerDir("calc"))
				slices.Sort(want)
				entries, err := os.ReadDir(filepath.Join(dir, codegen.Gendir, transport, naming.CLIDir))
				require.NoError(t, err)
				got := make([]string, 0, len(entries))
				for _, entry := range entries {
					if entry.IsDir() {
						got = append(got, entry.Name())
					}
				}
				assert.Equal(t, want, got, transport)
			}

			_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err)
			output, err := testingx.RunCmd(dir, "go", "build", "./...")
			require.NoError(t, err, output)
		})
	}
}

// cliServiceDSL returns a design with a service named cli exposed over
// transports and hosted by the server calc.
func cliServiceDSL(transports []string) func() {
	return func() {
		dsl.API("clidirs", func() {
			dsl.Server("calc", func() {
				dsl.Services("cli")
			})
		})
		dsl.Service("cli", func() {
			if slices.Contains(transports, "jsonrpc") {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
			}
			dsl.Method("show", func() {
				dsl.Payload(func() {
					dsl.Attribute("id", dsl.Int, func() {
						dsl.Meta("rpc:tag", "1")
					})
				})
				dsl.Result(dsl.String)
				for _, transport := range transports {
					switch transport {
					case "http":
						dsl.HTTP(func() {
							dsl.POST("/show")
						})
					case "grpc":
						dsl.GRPC(func() {})
					case "jsonrpc":
						dsl.JSONRPC(func() {})
					}
				}
			})
		})
	}
}
