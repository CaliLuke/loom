package service

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// TestExampleServiceStubSignatures checks that every example stub has the
// parameter and result types of the matching method of the generated service
// interface, so the starter implementation satisfies the interface for each
// method shape: unary, server streaming, results next to a server stream, and
// the per-message methods of a JSON-RPC WebSocket service.
func TestExampleServiceStubSignatures(t *testing.T) {
	note := func() any {
		return dsl.Type("Note", func() {
			dsl.Attribute("text", dsl.String)
			dsl.Required("text")
		})
	}
	cases := []struct {
		Name    string
		Service string
		DSL     func()
		Want    map[string]string
	}{
		{
			Name:    "http-sse-mixed-results",
			Service: "mixed",
			DSL: func() {
				dsl.API("mixedapi", func() {})
				n := note()
				dsl.Service("mixed", func() {
					dsl.Method("watch", func() {
						dsl.Payload(func() {
							dsl.Attribute("id", dsl.String)
						})
						dsl.Result(n)
						dsl.StreamingResult(dsl.String)
						dsl.HTTP(func() {
							dsl.GET("/watch")
							dsl.Param("id")
							dsl.ServerSentEvents()
						})
					})
					dsl.Method("tail", func() {
						dsl.Result(dsl.Int)
						dsl.StreamingResult(n)
						dsl.HTTP(func() {
							dsl.GET("/tail")
							dsl.ServerSentEvents()
						})
					})
					dsl.Method("ticks", func() {
						dsl.StreamingResult(dsl.String)
						dsl.HTTP(func() {
							dsl.GET("/ticks")
							dsl.ServerSentEvents()
						})
					})
					dsl.Method("show", func() {
						dsl.Payload(dsl.String)
						dsl.Result(n)
						dsl.HTTP(func() {
							dsl.GET("/show/{p}")
						})
					})
				})
			},
			Want: map[string]string{
				"Watch": "(context.Context, *WatchPayload, WatchServerStream) (*Note, error)",
				"Tail":  "(context.Context, TailServerStream) (int, error)",
				"Ticks": "(context.Context, TicksServerStream) (error)",
				"Show":  "(context.Context, string) (*Note, error)",
			},
		},
		{
			Name:    "jsonrpc-sse-mixed-results",
			Service: "mixed",
			DSL: func() {
				dsl.API("mixedapi", func() {
					dsl.JSONRPC(func() {})
				})
				n := note()
				dsl.Service("mixed", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("watch", func() {
						dsl.Payload(func() {
							dsl.ID("id", dsl.String)
						})
						dsl.Result(n)
						dsl.StreamingResult(dsl.String)
						dsl.JSONRPC(func() {
							dsl.ServerSentEvents()
						})
					})
				})
			},
			Want: map[string]string{
				"Watch": "(context.Context, *WatchPayload, WatchServerStream) (*Note, error)",
			},
		},
		{
			Name:    "jsonrpc-websocket",
			Service: "files",
			DSL: func() {
				dsl.API("filesapi", func() {
					dsl.JSONRPC(func() {})
				})
				n := note()
				dsl.Service("files", func() {
					dsl.JSONRPC(func() {
						dsl.GET("/ws")
					})
					dsl.Method("upload", func() {
						dsl.StreamingPayload(n)
						dsl.Result(n)
						dsl.JSONRPC(func() {})
					})
					dsl.Method("count", func() {
						dsl.StreamingPayload(dsl.String)
						dsl.Result(func() {
							dsl.Attribute("count", dsl.Int)
						})
						dsl.JSONRPC(func() {})
					})
					dsl.Method("push", func() {
						dsl.StreamingPayload(dsl.String)
						dsl.JSONRPC(func() {})
					})
					dsl.Method("echo", func() {
						dsl.StreamingPayload(n)
						dsl.StreamingResult(n)
						dsl.JSONRPC(func() {})
					})
					dsl.Method("watch", func() {
						dsl.Payload(dsl.String)
						dsl.StreamingResult(n)
						dsl.JSONRPC(func() {})
					})
				})
			},
			Want: map[string]string{
				"HandleStream": "(context.Context, Stream) (error)",
				"Upload":       "(context.Context, *Note) (*Note, error)",
				"Count":        "(context.Context, string) (*CountResult, error)",
				"Push":         "(context.Context, string) (error)",
				"Echo":         "(context.Context, *Note, EchoServerStream) (error)",
				"Watch":        "(context.Context, string, WatchServerStream) (error)",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			svc := root.Service(c.Service)
			require.NotNil(t, svc)
			data := services.Get(c.Service)
			require.NoError(t, SetUserTypeImports("example.com/app/gen", data))
			dir := t.TempDir()
			files := Files("example.com/app/gen", svc, services, map[string][]string{})
			require.NotEmpty(t, files)
			_, err := files[0].Render(dir)
			require.NoError(t, err)
			examples := ExampleServiceFiles("example.com/app/gen", root, services)
			require.Len(t, examples, 1)
			_, err = examples[0].Render(dir)
			require.NoError(t, err)

			iface := interfaceSignatures(t, filepath.Join(dir, files[0].Path))
			stubs := stubSignatures(t, filepath.Join(dir, examples[0].Path), data.PkgName)
			assert.Equal(t, c.Want, iface)
			assert.Equal(t, iface, stubs)
		})
	}
}

// interfaceSignatures returns the signatures of the Service interface methods
// declared in the file at path.
func interfaceSignatures(t *testing.T, path string) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)
	sigs := make(map[string]string)
	ast.Inspect(f, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "Service" {
			return true
		}
		iface, ok := spec.Type.(*ast.InterfaceType)
		require.True(t, ok)
		for _, m := range iface.Methods.List {
			sigs[m.Names[0].Name] = signature(t, fset, m.Type.(*ast.FuncType), "")
		}
		return false
	})
	return sigs
}

// stubSignatures returns the signatures of the methods declared in the
// example file at path, with the service package qualifier pkg removed.
func stubSignatures(t *testing.T, path, pkg string) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)
	sigs := make(map[string]string)
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			continue
		}
		sigs[fn.Name.Name] = signature(t, fset, fn.Type, pkg+".")
	}
	return sigs
}

// signature renders the parameter and result types of fn without names,
// removing qualifier from the type expressions.
func signature(t *testing.T, fset *token.FileSet, fn *ast.FuncType, qualifier string) string {
	t.Helper()
	list := func(fields *ast.FieldList) string {
		var types []string
		if fields != nil {
			for _, field := range fields.List {
				var b bytes.Buffer
				require.NoError(t, printer.Fprint(&b, fset, field.Type))
				typ := b.String()
				if qualifier != "" {
					typ = strings.ReplaceAll(typ, qualifier, "")
				}
				n := len(field.Names)
				if n == 0 {
					n = 1
				}
				for range n {
					types = append(types, typ)
				}
			}
		}
		return "(" + strings.Join(types, ", ") + ")"
	}
	return list(fn.Params) + " " + list(fn.Results)
}
