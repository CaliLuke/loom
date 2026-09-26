package generator

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	d "github.com/CaliLuke/loom/dsl"
)

// TestNonASCIINamesGeneratedModuleBuilds runs the gen and example commands
// over designs whose API, server, services, methods, types and struct
// metadata use non-ASCII names on HTTP, gRPC and JSON-RPC. It runs protoc,
// then builds and vets the resulting module, including the example
// commands, for each transport alone and for one server that mixes them.
func TestNonASCIINamesGeneratedModuleBuilds(t *testing.T) {
	const server = "valu30b5u30fcu30d0u30fc"
	cases := []struct {
		Name  string
		DSL   func()
		Files []string
		// APIFile is the example service file in the API package.
		APIFile string
		// Contains maps generated files to text they must contain.
		Contains map[string]string
	}{
		{
			Name:    "http and grpc",
			DSL:     nonASCIIHTTPGRPCDSL,
			APIFile: "cru00e8me.go",
			Contains: map[string]string{
				filepath.Join(cg.Gendir, "grpc", "cru00e8me", "pb", "loomgen_nonascii_cru00e8me.proto"): "message EntrEProto {",
				filepath.Join(cg.Gendir, "tipos", "menu00fc", "menü.go"):                                "package menu00fc",
			},
			Files: []string{
				filepath.Join("cmd", server, "main.go"),
				filepath.Join("cmd", server, "grpc.go"),
				filepath.Join("cmd", server+"-cli", "main.go"),
				filepath.Join(cg.Gendir, "http", "cli", server, "cli.go"),
				filepath.Join(cg.Gendir, "grpc", "cli", server, "cli.go"),
				filepath.Join(cg.Gendir, "tipos", "menu00fc", "menü.go"),
				filepath.Join(cg.Gendir, "http", "openapi.json"),
			},
		},
		{
			Name:    "jsonrpc",
			DSL:     nonASCIIJSONRPCDSL,
			APIFile: "u65e5u672c.go",
			Files: []string{
				filepath.Join("cmd", server, "main.go"),
				filepath.Join("cmd", server+"-cli", "main.go"),
				filepath.Join(cg.Gendir, "jsonrpc", "cli", server, "cli.go"),
			},
		},
		{
			Name:    "mixed transports",
			DSL:     nonASCIIMixedDSL,
			APIFile: "u65e5u672c.go",
			Files: []string{
				filepath.Join("cmd", server, "main.go"),
				filepath.Join("cmd", server, "grpc.go"),
				filepath.Join("cmd", server, "jsonrpc.go"),
				filepath.Join(cg.Gendir, "http", "cli", server, "cli.go"),
				filepath.Join(cg.Gendir, "grpc", "cli", server, "cli.go"),
				filepath.Join(cg.Gendir, "jsonrpc", "cli", server, "cli.go"),
			},
		},
	}
	source := loomModuleSource(t)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			dir := t.TempDir()
			goMod := fmt.Sprintf("module example.com/nonascii\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

			cg.RunDSL(t, c.DSL)
			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", false)
			require.NoError(t, err)
			for _, path := range c.Files {
				assert.FileExists(t, filepath.Join(dir, path))
			}
			for path, text := range c.Contains {
				content, err := os.ReadFile(filepath.Join(dir, path))
				require.NoError(t, err)
				assert.Contains(t, string(content), text, path)
			}
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, c.APIFile), nil, parser.PackageClauseOnly)
			require.NoError(t, err)
			assert.Equal(t, "cafu00e9", f.Name.Name, "API package name")

			runGeneratedModuleGo(t, dir, "mod", "tidy")
			runGeneratedModuleGo(t, dir, "build", "./...")
			runGeneratedModuleGo(t, dir, "vet", "./...")
		})
	}
}

func nonASCIIHTTPGRPCDSL() {
	d.API("Café", func() {
		d.Server("サーバー", func() {
			d.Services("Crème")
			d.Host("local", func() {
				d.URI("http://localhost:8000")
				d.URI("grpc://localhost:8080")
			})
		})
	})
	nonASCIIHTTPGRPCService()
}

func nonASCIIJSONRPCDSL() {
	d.API("Café", func() {
		d.JSONRPC(func() {})
		d.Server("サーバー", func() {
			d.Services("日本")
			d.Host("local", func() {
				d.URI("http://localhost:8000")
			})
		})
	})
	nonASCIIJSONRPCService()
}

func nonASCIIMixedDSL() {
	d.API("Café", func() {
		d.JSONRPC(func() {})
		d.Server("サーバー", func() {
			d.Services("Crème", "日本")
			d.Host("local", func() {
				d.URI("http://localhost:8000")
				d.URI("grpc://localhost:8080")
			})
		})
	})
	nonASCIIHTTPGRPCService()
	nonASCIIJSONRPCService()
}

func nonASCIIHTTPGRPCService() {
	var Menu = d.Type("Menü", func() {
		d.Meta("struct:pkg:path", "tipos/menü")
		d.Field(1, "plat", d.String)
		d.Field(2, "prix", d.Int)
	})
	var Order = d.Type("Entrée", func() {
		d.Meta("struct:name:proto", "EntréeProto")
		d.Field(1, "menü", Menu)
		d.Field(2, "quantité", d.Int)
	})
	d.Service("Crème", func() {
		d.Method("añadir", func() {
			d.Payload(Order)
			d.Result(Order)
			d.HTTP(func() {
				d.POST("/añadir")
			})
			d.GRPC(func() {})
		})
	})
}

func nonASCIIJSONRPCService() {
	d.Service("日本", func() {
		d.JSONRPC(func() {
			d.POST("/rpc")
		})
		d.Method("取得", func() {
			d.Payload(func() {
				d.ID("id", d.String)
				d.Attribute("名前", d.String)
			})
			d.Result(func() {
				d.ID("id", d.String)
				d.Attribute("total", d.Int)
			})
			d.JSONRPC(func() {})
		})
	})
}
