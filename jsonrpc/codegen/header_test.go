package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// httpNamedServices are service names whose package import paths start with
// the name of the generated HTTP tree.
var httpNamedServices = []string{"http", "httpbin"}

// TestJSONRPCImportPath checks that only the packages of the generated HTTP
// tree move to the JSON-RPC tree, matching whole path segments.
func TestJSONRPCImportPath(t *testing.T) {
	const genpkg = "example.com/gen/httpapi/gen"
	cases := []struct {
		Name, Path, Want string
	}{
		{"http-server", genpkg + "/http/calc/server", genpkg + "/jsonrpc/calc/server"},
		{"http-client", genpkg + "/http/http_/client", genpkg + "/jsonrpc/http_/client"},
		{"service-named-http", genpkg + "/http_", genpkg + "/http_"},
		{"service-named-httpbin", genpkg + "/httpbin", genpkg + "/httpbin"},
		{"service-views", genpkg + "/httpbin/views", genpkg + "/httpbin/views"},
		{"http-root", genpkg + "/http", genpkg + "/http"},
		{"module-with-gen-http", "example.com/gen/httpapi/types", "example.com/gen/httpapi/types"},
		{"other-module-http-tree", "example.com/other/gen/http/calc/server", "example.com/other/gen/http/calc/server"},
		{"loom-http", "github.com/CaliLuke/loom/http", "github.com/CaliLuke/loom/http"},
		{"stdlib", "net/http", "net/http"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			assert.Equal(t, c.Want, jsonrpcImportPath(genpkg, c.Path))
		})
	}
}

// TestHTTPNamedServiceImports checks that the JSON-RPC encoder, decoder and
// type files, which the HTTP code generator builds, keep the import paths
// of service packages whose paths start with "http" in a module whose path
// contains "gen/http".
func TestHTTPNamedServiceImports(t *testing.T) {
	const genpkg = "example.com/gen/httpapi/gen"
	root := RunJSONRPCDSL(t, httpNamedServiceDSL)
	services := CreateJSONRPCServices(root)
	files := append(ServerFiles(genpkg, services), ClientFiles(genpkg, services)...)
	files = append(files, ServerTypeFiles(genpkg, services)...)
	want := map[string]string{
		"http":    genpkg + "/http_",
		"httpbin": genpkg + "/httpbin",
	}
	for _, name := range httpNamedServices {
		t.Run(name, func(t *testing.T) {
			dir := services.Get(name).Service.PathName
			checked := 0
			for _, f := range files {
				if f == nil || filepath.Dir(filepath.Dir(f.Path)) != filepath.Join(codegen.Gendir, "jsonrpc", dir) {
					continue
				}
				if base := filepath.Base(f.Path); base != "encode_decode.go" && base != "types.go" {
					continue
				}
				header := codegen.HeaderDataForSection(f.HeaderSection())
				require.NotNil(t, header, f.Path)
				found := false
				for _, spec := range header.Imports {
					assert.NotContains(t, spec.Path, "jsonrpcapi", f.Path)
					assert.NotContains(t, spec.Path, "gen/jsonrpc_", f.Path)
					assert.NotContains(t, spec.Path, "gen/jsonrpcbin", f.Path)
					if spec.Path == want[name] {
						found = true
					}
				}
				assert.Truef(t, found, "%s does not import %s", f.Path, want[name])
				checked++
			}
			assert.NotZero(t, checked)
		})
	}
}

// TestHTTPNamedServiceGeneratedModuleBuilds checks that the generated
// packages of JSON-RPC services named http and httpbin, in a module whose
// path contains "gen/http", compile.
func TestHTTPNamedServiceGeneratedModuleBuilds(t *testing.T) {
	root := RunJSONRPCDSL(t, httpNamedServiceDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/gen/httpapi", root)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}

// httpNamedServiceDSL declares JSON-RPC services named http and httpbin with
// a unary method and a server-sent events method.
func httpNamedServiceDSL() {
	API("httpnamed", func() {
		JSONRPC(func() {})
	})
	for _, name := range httpNamedServices {
		Service(name, func() {
			JSONRPC(func() {
				POST("/rpc/" + name)
			})
			Method("add", func() {
				Payload(func() {
					ID("id", String)
					Attribute("a", Int)
				})
				Result(func() {
					ID("id", String)
					Attribute("sum", Int)
				})
				JSONRPC(func() {})
			})
			Method("watch", func() {
				Payload(func() {
					ID("id", String)
				})
				StreamingResult(func() {
					ID("id", String)
					Attribute("v", String)
				})
				JSONRPC(func() {
					ServerSentEvents()
				})
			})
		})
	}
}
