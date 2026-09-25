package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/loomsource"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestStructPkgPathEdgeCasesCompile generates the service, transport and
// example output of designs that use struct:pkg:path types in streams, in a
// package named like the service, or in a service whose name matches a root
// example file, then builds and vets the module.
func TestStructPkgPathEdgeCasesCompile(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
		// Files maps generated files to content they must contain.
		Files map[string][]string
	}{
		{
			Name: "streaming-http",
			DSL:  structPkgPathStreamingDSL("http"),
			Files: map[string][]string{
				"gen/svc/service.go": {"Recv() (*menu.Plain, error)", "SendAndClose(*menu.Plain) error"},
				"gen/menu/plain.go":  {"type Plain struct"},
			},
		},
		{
			Name: "streaming-grpc",
			DSL:  structPkgPathStreamingDSL("grpc"),
			Files: map[string][]string{
				"gen/svc/service.go": {"Recv() (*menu.Plain, error)"},
				"gen/menu/plain.go":  {"type Plain struct"},
			},
		},
		{
			Name: "streaming-jsonrpc",
			DSL:  structPkgPathStreamingDSL("jsonrpc"),
			Files: map[string][]string{
				"gen/svc/service.go": {"Relay(context.Context, *menu.Plain, RelayServerStream) (err error)", "SendNotification(context.Context, *menu.Plain) error"},
				"gen/menu/plain.go":  {"type Plain struct"},
			},
		},
		{
			Name: "service-named-like-package-http",
			DSL:  structPkgPathDesign{HTTP: true, Multipart: true, List: true, Maps: true, PkgPath: "types/catalog"}.DSL,
			Files: map[string][]string{
				"gen/catalog/service.go":            {"package catalogsvc", `catalog "example.com/probe/gen/types/catalog"`},
				"gen/http/catalog/server/server.go": {`catalogsvc "example.com/probe/gen/catalog"`},
				"gen/http/catalog/client/types.go":  {`catalog "example.com/probe/gen/types/catalog"`},
				"catalog.go":                        {`catalog "example.com/probe/gen/types/catalog"`, "p *catalog.Item) (res *catalog.Item, err error)"},
				"gen/http/catalog/server/types.go":  {"*catalog.Item"},
				"gen/http/catalog/client/cli.go":    {"*catalog.Item"},
			},
		},
		{
			Name: "service-named-like-package-grpc",
			DSL:  structPkgPathDesign{GRPC: true, List: true, PkgPath: "types/catalog"}.DSL,
			Files: map[string][]string{
				"gen/grpc/catalog/server/server.go": {`catalogsvc "example.com/probe/gen/catalog"`},
			},
		},
		{
			Name: "service-named-like-package-jsonrpc",
			DSL:  structPkgPathDesign{JSONRPC: true, List: true, PkgPath: "types/catalog"}.DSL,
			Files: map[string][]string{
				"gen/jsonrpc/catalog/server/server.go": {`catalogsvc "example.com/probe/gen/catalog"`},
			},
		},
		{
			Name: "service-common-package-types-common",
			DSL:  structPkgPathDesign{HTTP: true, GRPC: true, List: true, Service: "common", PkgPath: "types/common"}.DSL,
		},
		{
			Name: "service-named-multipart",
			DSL:  structPkgPathDesign{HTTP: true, Multipart: true, List: true, Service: "multipart"}.DSL,
			Files: map[string][]string{
				"multipart.go":        {"func (s *multipartsrvc) Put("},
				"multipart_codecs.go": {`"mime/multipart"`, "func MultipartPutEncoderFunc(mw *multipart.Writer, p *common.Item) error"},
			},
		},
	}
	source := structPkgPathLoomSource(t)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			codegen.RunDSL(t, c.DSL)
			dir := writeStructPkgPathModule(t, source)
			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", false)
			require.NoError(t, err)
			for path, wants := range c.Files {
				content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
				if !assert.NoError(t, err) {
					continue
				}
				for _, want := range wants {
					assert.Contains(t, string(content), want, path)
				}
			}
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}

// TestStructPkgPathServicePackageRejected checks that generation fails with
// an actionable error when a service uses a type whose struct:pkg:path is the
// package generated for that service.
func TestStructPkgPathServicePackageRejected(t *testing.T) {
	for _, transport := range []string{"http", "grpc", "jsonrpc"} {
		t.Run(transport, func(t *testing.T) {
			d := structPkgPathDesign{Service: "common", PkgPath: "common", List: true}
			switch transport {
			case "http":
				d.HTTP = true
			case "grpc":
				d.GRPC = true
			case "jsonrpc":
				d.JSONRPC = true
			}
			codegen.RunDSL(t, d.DSL)
			dir := writeStructPkgPathModule(t, structPkgPathLoomSource(t))
			for _, cmd := range []string{"gen", "example"} {
				_, err := Generate(dir, cmd, false)
				require.Error(t, err, cmd)
				assert.Contains(t, err.Error(), `service "common": type "Item" has struct:pkg:path "common", the package generated for the service; the service package cannot import itself.`, cmd)
			}
		})
	}
}

// structPkgPathLoomSource returns the Loom module directory that generated
// test modules replace github.com/CaliLuke/loom with.
func structPkgPathLoomSource(t *testing.T) string {
	t.Helper()
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	source, err := loomsource.Resolve(repoRoot, filepath.Join(t.TempDir(), "loom-pinned"))
	require.NoError(t, err)
	return source
}

// writeStructPkgPathModule returns a new directory holding the go.mod file
// of the example.com/probe module, which uses the Loom module at source.
func writeStructPkgPathModule(t *testing.T, source string) string {
	t.Helper()
	dir := t.TempDir()
	goMod := fmt.Sprintf("module example.com/probe\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	return dir
}

// structPkgPathStreamingDSL returns a design whose streaming methods exposed
// on transport send and receive the "Plain" type generated in the "menu"
// struct:pkg:path package, directly and nested in a service type.
func structPkgPathStreamingDSL(transport string) func() {
	return func() {
		dsl.API("probe", func() {
			if transport == "jsonrpc" {
				dsl.JSONRPC(func() {})
			}
		})
		plain := dsl.Type("Plain", func() {
			dsl.Meta("struct:pkg:path", "menu")
			dsl.Field(1, "name", dsl.String)
			dsl.Field(2, "count", dsl.Int)
			dsl.Required("name")
		})
		wrap := dsl.Type("Wrap", func() {
			dsl.Field(1, "plain", plain)
			dsl.Field(2, "plains", dsl.ArrayOf(plain))
		})
		endpoint := func(route func()) {
			switch transport {
			case "http":
				dsl.HTTP(route)
			case "grpc":
				dsl.GRPC(func() {})
			case "jsonrpc":
				dsl.JSONRPC(func() {})
			}
		}
		dsl.Service("svc", func() {
			if transport == "jsonrpc" {
				dsl.JSONRPC(func() {
					dsl.GET("/rpc")
				})
			}
			if transport != "jsonrpc" {
				dsl.Method("upload", func() {
					dsl.StreamingPayload(plain)
					dsl.Result(plain)
					endpoint(func() {
						dsl.GET("/upload")
					})
				})
			}
			dsl.Method("relay", func() {
				dsl.StreamingPayload(plain)
				dsl.StreamingResult(plain)
				endpoint(func() {
					dsl.GET("/relay")
				})
			})
			dsl.Method("wrapped", func() {
				dsl.StreamingPayload(wrap)
				dsl.StreamingResult(wrap)
				endpoint(func() {
					dsl.GET("/wrapped")
				})
			})
			if transport != "jsonrpc" {
				dsl.Method("watch", func() {
					dsl.Payload(func() {
						dsl.Field(1, "name", dsl.String)
					})
					dsl.StreamingResult(plain)
					endpoint(func() {
						dsl.GET("/watch")
						dsl.Param("name")
					})
				})
			}
			if transport == "http" {
				dsl.Method("events", func() {
					dsl.StreamingResult(plain)
					dsl.HTTP(func() {
						dsl.GET("/events")
						dsl.ServerSentEvents()
					})
				})
			}
		})
		if transport == "jsonrpc" {
			dsl.Service("feed", func() {
				dsl.JSONRPC(func() {
					dsl.POST("/feed")
				})
				dsl.Method("events", func() {
					dsl.Payload(func() {
						dsl.ID("id", dsl.String)
					})
					dsl.StreamingResult(plain)
					dsl.JSONRPC(func() {
						dsl.ServerSentEvents()
					})
				})
			})
		}
	}
}
