package codegen

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInlineBodyShapeMismatchControls(t *testing.T) {
	const source = "package specimen\ntype Body struct { U *struct { Label *string `json:\"label\"` } }\nfunc BuildBody() { v := &struct { Label *string `json:\"label\"` }{}; _ = v }"
	for _, tc := range []struct {
		name, literal string
		equal         bool
	}{
		{"unchanged", "&struct { Label *string `json:\"label\"` }{}", true},
		{"equivalent quoted tag", "&struct { Label *string \"json:\\\"label\\\"\" }{}", true},
		{"tag mismatch", "&struct { Label *string `json:\"label,omitempty\"` }{}", false},
		{"type mismatch", "&struct { Label string `json:\"label\"` }{}", false},
		{"missing literal", "0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Replace(source, "v := &struct { Label *string `json:\"label\"` }{}", "v := "+tc.literal, 1)
			file, err := parser.ParseFile(token.NewFileSet(), "specimen.go", text, 0)
			require.NoError(t, err)
			declared, emitted := inlineBodyShapes(t, []*ast.File{file}, "Body", "BuildBody")
			require.Len(t, declared, 1)
			if tc.equal {
				require.Equal(t, declared, emitted)
			} else {
				require.NotEqual(t, declared, emitted)
			}
		})
	}
}

func parseInlineBodyPackage(t *testing.T, dir, side string) []*ast.File {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "gen", "http", "inline_transform", side, "*.go"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	files := make([]*ast.File, 0, len(paths))
	for _, path := range paths {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		require.NoError(t, err)
		files = append(files, file)
	}
	return files
}

func inlineBodyShapes(t *testing.T, files []*ast.File, body, transform string) (map[string]bool, map[string]bool) {
	t.Helper()
	var declaration *ast.StructType
	var constructor *ast.FuncDecl
	for _, file := range files {
		for _, node := range file.Decls {
			switch node := node.(type) {
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					if spec, ok := spec.(*ast.TypeSpec); ok && spec.Name.Name == body {
						require.Nil(t, declaration, "ambiguous body declaration %s", body)
						declaration, ok = spec.Type.(*ast.StructType)
						require.True(t, ok, "body %s must remain a struct", body)
					}
				}
			case *ast.FuncDecl:
				if node.Recv == nil && node.Name.Name == transform {
					require.Nil(t, constructor, "ambiguous body transform %s", transform)
					constructor = node
				}
			}
		}
	}
	require.NotNil(t, declaration, "missing body declaration %s", body)
	require.NotNil(t, constructor, "missing body transform %s", transform)
	declared := make(map[string]bool)
	ast.Inspect(declaration.Fields, func(node ast.Node) bool {
		if shape, ok := node.(*ast.StructType); ok {
			declared[inlineStructShape(t, shape)] = true
		}
		return true
	})
	emitted := make(map[string]bool)
	ast.Inspect(constructor.Body, func(node ast.Node) bool {
		if literal, ok := node.(*ast.CompositeLit); ok {
			if shape, ok := literal.Type.(*ast.StructType); ok {
				emitted[inlineStructShape(t, shape)] = true
			}
		}
		return true
	})
	return declared, emitted
}

func inlineStructShape(t *testing.T, shape *ast.StructType) string {
	t.Helper()
	var out bytes.Buffer
	// An empty file set removes source-position formatting differences while
	// retaining complete field/type syntax. Compare semantic tag strings even
	// when a literal is quoted differently in otherwise identical syntax.
	require.NoError(t, format.Node(&out, token.NewFileSet(), shape))
	var tags []string
	ast.Inspect(shape, func(node ast.Node) bool {
		if field, ok := node.(*ast.Field); ok && field.Tag != nil {
			value, err := strconv.Unquote(field.Tag.Value)
			require.NoError(t, err)
			tags = append(tags, field.Tag.Value, strconv.Quote(value))
		}
		return true
	})
	return strings.NewReplacer(tags...).Replace(out.String())
}
