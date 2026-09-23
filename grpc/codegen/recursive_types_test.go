package codegen

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestGRPCFilesRenderRecursiveTypes(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"self-recursive-result", testdata.MessageUserTypeWithNestedUserTypesDSL},
		{"mutually-recursive-payload-and-result", mutuallyRecursiveGRPCDSL},
		{"collection-recursive-payload-and-result", collectionRecursiveGRPCDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			services := CreateGRPCServices(root)
			groups := map[string][]*codegen.File{
				"proto":        ProtoFiles("", services),
				"server":       ServerFiles("", services),
				"client":       ClientFiles("", services),
				"server-types": ServerTypeFiles("", services),
				"client-types": ClientTypeFiles("", services),
				"client-cli":   ClientCLIFiles("", services),
			}
			dir := t.TempDir()
			declared := make(map[string]map[string]struct{})
			called := make(map[string]map[string]struct{})
			for name, files := range groups {
				require.NotEmpty(t, files, name)
				for _, f := range files {
					path, err := f.Render(dir)
					require.NoError(t, err, "%s: %s", name, f.Path)
					if filepath.Ext(path) == ".go" {
						collectTransformHelperNames(t, path, declared, called)
					}
				}
			}
			for pkg, names := range called {
				for name := range names {
					_, ok := declared[pkg][name]
					require.True(t, ok, "%s calls undeclared transform helper %s", pkg, name)
				}
			}
		})
	}
}

func TestHasAnyTypeRecursiveTypes(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Method   string
		Expected bool
	}{
		{"self-recursive", testdata.MessageUserTypeWithNestedUserTypesDSL, "MethodMessageUserTypeWithNestedUserTypes", false},
		{"mutually-recursive", mutuallyRecursiveGRPCDSL, "Walk", false},
		{"mutually-recursive-with-any", mutuallyRecursiveAnyGRPCDSL, "Walk", true},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			require.Len(t, root.Services, 1)
			m := root.Services[0].Method(c.Method)
			require.NotNil(t, m)
			require.Equal(t, c.Expected, hasAnyType(m.Payload) || hasAnyType(m.Result))
		})
	}
}

func TestIsInlineRecursive(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Type     string
		Expected bool
	}{
		{"direct-field-recursion", testdata.MessageUserTypeWithNestedUserTypesDSL, "RecursiveT", false},
		{"non-recursive", testdata.MessageUserTypeWithNestedUserTypesDSL, "UT", false},
		{"mutual-recursion-through-field", mutuallyRecursiveGRPCDSL, "Edge", false},
		{"mutual-recursion-through-collections", mutuallyRecursiveGRPCDSL, "Node", false},
		{"array-and-map-recursion", collectionRecursiveGRPCDSL, "Tree", true},
		{"inline-object-recursion", inlineObjectRecursiveGRPCDSL, "Branch", true},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			ut := root.UserType(c.Type)
			require.NotNil(t, ut)
			require.Equal(t, c.Expected, isInlineRecursive(&expr.AttributeExpr{Type: ut}))

			services := CreateGRPCServices(root)
			for _, f := range append(ServerTypeFiles("", services), ClientTypeFiles("", services)...) {
				var buf bytes.Buffer
				for _, s := range f.AllSections() {
					require.NoError(t, s.Write(&buf), f.Path)
				}
				_, err := parser.ParseFile(token.NewFileSet(), f.Path, buf.Bytes(), parser.SkipObjectResolution)
				require.NoError(t, err, f.Path)
			}
		})
	}
}

func mutuallyRecursiveGRPCDSL() {
	var Node = Type("Node", func() {
		Field(1, "name", String)
		Field(2, "edges", ArrayOf("Edge"))
		Field(3, "index", MapOf(String, "Edge"))
	})
	var Edge = Type("Edge", func() {
		Field(1, "target", Node)
	})
	Service("Graph", func() {
		Method("Walk", func() {
			Payload(Node)
			Result(Edge)
			GRPC(func() {})
		})
	})
}

func collectionRecursiveGRPCDSL() {
	var Tree = Type("Tree", func() {
		Field(1, "children", ArrayOf("Tree"))
		Field(2, "lookup", MapOf(String, "Tree"))
	})
	Service("Forest", func() {
		Method("Grow", func() {
			Payload(Tree)
			Result(Tree)
			GRPC(func() {})
		})
	})
}

func inlineObjectRecursiveGRPCDSL() {
	var Branch = Type("Branch", func() {
		Field(1, "leaves", func() {
			Field(1, "branches", ArrayOf("Branch"))
		})
	})
	Service("Orchard", func() {
		Method("Prune", func() {
			Payload(Branch)
			Result(Branch)
			GRPC(func() {})
		})
	})
}

func mutuallyRecursiveAnyGRPCDSL() {
	var Node = Type("Node", func() {
		Field(1, "edges", ArrayOf("Edge"))
	})
	var Edge = Type("Edge", func() {
		Field(1, "target", Node)
		Field(2, "extra", Any)
	})
	Service("Graph", func() {
		Method("Walk", func() {
			Payload(Node)
			Result(Edge)
			GRPC(func() {})
		})
	})
}

// collectTransformHelperNames records the protobuf transform helpers that the
// Go file at path declares and calls, keyed by the file directory.
func collectTransformHelperNames(t *testing.T, path string, declared, called map[string]map[string]struct{}) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	require.NoError(t, err)
	pkg := filepath.Dir(path)
	if declared[pkg] == nil {
		declared[pkg] = make(map[string]struct{})
		called[pkg] = make(map[string]struct{})
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Recv == nil && transformHelperPattern.MatchString(node.Name.Name) {
				declared[pkg][node.Name.Name] = struct{}{}
			}
		case *ast.CallExpr:
			if id, ok := node.Fun.(*ast.Ident); ok && transformHelperPattern.MatchString(id.Name) {
				called[pkg][id.Name] = struct{}{}
			}
		}
		return true
	})
}

var transformHelperPattern = regexp.MustCompile(`^(svc|protobuf)[A-Z].*To[A-Z]`)
