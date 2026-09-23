package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestMetadataVarNamesAreUniquePerEndpoint(t *testing.T) {
	root := RunGRPCDSL(t, testdata.MetadataVarNameCollisionDSL)
	services := CreateGRPCServices(root)
	files := append(ServerFiles("", services), ClientFiles("", services)...)

	funcs := map[string]string{}
	for _, file := range files {
		renderedPath, err := file.Render(t.TempDir())
		require.NoError(t, err)
		src, err := os.ReadFile(renderedPath)
		require.NoError(t, err)
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, renderedPath, src, 0)
		require.NoError(t, err)
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			funcs[fn.Name.Name] = string(src[fset.Position(fn.Pos()).Offset:fset.Position(fn.End()).Offset])
			for _, dup := range duplicateBlockVars(fn.Body) {
				t.Errorf("%s: %s declares %q more than once", file.Path, fn.Name.Name, dup)
			}
		}
	}

	cases := []struct {
		Name     string
		Func     string
		Contains []string
		Excludes []string
	}{
		{
			Name:     "server request decoder",
			Func:     "DecodeCollidingRequest",
			Contains: []string{`md.Get("foo_bar")`, `md.Get("x-foo-bar")`, "fooBar2"},
		},
		{
			Name:     "client request encoder",
			Func:     "EncodeCollidingRequest",
			Contains: []string{`"foo_bar"`, `"x-foo-bar"`},
		},
		{
			Name:     "client response decoder",
			Func:     "DecodeCollidingResponse",
			Contains: []string{`hdr.Get("res_id")`, `trlr.Get("x-res-id")`, "resID2"},
		},
		{
			Name:     "isolated endpoint keeps base name",
			Func:     "DecodeIsolatedRequest",
			Contains: []string{"fooBar"},
			Excludes: []string{"fooBar2"},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			code, ok := funcs[c.Func]
			require.True(t, ok, "function %s not generated", c.Func)
			for _, s := range c.Contains {
				require.Contains(t, code, s)
			}
			for _, s := range c.Excludes {
				require.NotContains(t, code, s)
			}
		})
	}
	require.Contains(t, funcs["DecodeCollidingRequest"], "NewCollidingPayload(message, fooBar, fooBar2)")
}

// duplicateBlockVars returns the identifiers declared more than once by var
// declarations directly inside the same block, for every block under body.
func duplicateBlockVars(body *ast.BlockStmt) []string {
	var dups []string
	ast.Inspect(body, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		seen := map[string]bool{}
		for _, stmt := range block.List {
			decl, ok := stmt.(*ast.DeclStmt)
			if !ok {
				continue
			}
			gen, ok := decl.Decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				for _, id := range spec.(*ast.ValueSpec).Names {
					if seen[id.Name] {
						dups = append(dups, id.Name)
					}
					seen[id.Name] = true
				}
			}
		}
		return true
	})
	return dups
}
