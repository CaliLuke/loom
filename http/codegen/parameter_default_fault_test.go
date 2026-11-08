package codegen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestParameterDefaultFault renders every consumer of a fallible parameter
// default and requires one server-fault classification that reaches the
// decoder result unchanged, with returns matching each enclosing function.
func TestParameterDefaultFault(t *testing.T) {
	root := RunHTTPDSL(t, parameterDefaultFaultDSL)
	dir := t.TempDir()
	renderHTTPModuleFiles(t, dir, "example.com/fault/gen", root)
	serverPath := filepath.Join(dir, "gen", "http", "fault", "server", "encode_decode.go")
	data, err := os.ReadFile(serverPath)
	require.NoError(t, err)
	server := string(data)

	failures := regexp.MustCompile(`.*"parameter [a-z]+: invalid default".*`).FindAllString(server, -1)
	require.Len(t, failures, 5, "styled, deepObject member, JSON query and multipart JSON query defaults, and the multipart JSON query member default applied by conversion")
	for _, failure := range failures {
		require.Regexp(t, `return (payload, )?loom\.Fault\("%s", "parameter`, failure, "a failed authored default is a server fault")
	}
	closure := regexp.MustCompile(`\}\(\); parameterError != nil \{\s*return (\S+)\(`).FindAllStringSubmatch(server, -1)
	require.NotEmpty(t, closure)
	for _, wrapper := range closure {
		require.NotEqual(t, "loom.DecodePayloadError", wrapper[1], "parameter assignment must not reclassify a server fault")
	}

	for _, path := range []string{serverPath, filepath.Join(dir, "gen", "http", "fault", "client", "cli.go")} {
		require.Empty(t, returnArityMismatches(t, path))
	}
}

func parameterDefaultFaultDSL() {
	label := func() {
		Meta("struct:field:type", "label.Label", "example.com/fault/label")
	}
	filter := Type("Filter", func() {
		Attribute("name", String)
		Attribute("tag", String, func() {
			label()
			Default("label:member")
		})
	})
	pet := Type("Pet", func() {
		Attribute("name", String)
		Attribute("tag", String, func() {
			label()
			Default("label:pet")
		})
		Required("name")
	})
	tokens := func() {
		Attribute("tokens", MapOf(String, String), func() {
			Elem(label)
			Default(map[string]any{"a": "label:query"})
		})
	}
	Service("fault", func() {
		Method("styled", func() {
			Payload(func() {
				Attribute("tags", ArrayOf(String), func() {
					Elem(label)
					Default([]any{"label:styled"})
				})
			})
			HTTP(func() {
				GET("/styled")
				Param("tags", func() {
					Meta("openapi:style", "pipeDelimited")
				})
				Response(StatusNoContent)
			})
		})
		Method("member", func() {
			Payload(func() {
				Attribute("filter", filter)
			})
			HTTP(func() {
				GET("/member")
				Param("filter", func() {
					Meta("openapi:style", "deepObject")
				})
				Response(StatusNoContent)
			})
		})
		Method("query", func() {
			Payload(func() {
				Attribute("page", Int)
				tokens()
				Required("page")
			})
			HTTP(func() {
				GET("/query")
				Param("page")
				Param("tokens")
				Response(StatusNoContent)
			})
		})
		Method("upload", func() {
			Payload(func() {
				tokens()
				Attribute("pets", MapOf(String, pet))
				Attribute("name", String)
			})
			HTTP(func() {
				POST("/upload")
				Param("tokens")
				Param("pets")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
	})
}

// returnArityMismatches reports return statements whose result count differs
// from their enclosing function's declared results. A single call result and
// a bare return with named results are accepted.
func returnArityMismatches(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err)
	var mismatches []string
	var visit func(node ast.Node, results *ast.FieldList)
	visit = func(node ast.Node, results *ast.FieldList) {
		ast.Inspect(node, func(n ast.Node) bool {
			switch actual := n.(type) {
			case *ast.FuncLit:
				if actual != node {
					visit(actual.Body, actual.Type.Results)
					return false
				}
			case *ast.ReturnStmt:
				want, named := resultCount(results)
				got := len(actual.Results)
				_, call := singleCall(actual.Results)
				if got != want && !(got == 0 && named) && !(got == 1 && call) {
					mismatches = append(mismatches, fmt.Sprintf("%s: %d results, want %d", fset.Position(actual.Pos()), got, want))
				}
			}
			return true
		})
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
			visit(fn.Body, fn.Type.Results)
		}
	}
	return mismatches
}

func resultCount(results *ast.FieldList) (int, bool) {
	if results == nil {
		return 0, false
	}
	count, named := 0, false
	for _, field := range results.List {
		if len(field.Names) == 0 {
			count++
			continue
		}
		named = true
		count += len(field.Names)
	}
	return count, named
}

func singleCall(results []ast.Expr) (*ast.CallExpr, bool) {
	if len(results) != 1 {
		return nil, false
	}
	call, ok := results[0].(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	// Error constructors return one value; other calls may forward several.
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		if pkg, ok := selector.X.(*ast.Ident); ok && slices.Contains([]string{"loom", "fmt", "errors"}, pkg.Name) {
			return nil, false
		}
	}
	return call, true
}
