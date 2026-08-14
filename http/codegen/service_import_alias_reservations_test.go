package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransportImportAliasReservationsCoverGeneratedImports(t *testing.T) {
	reserved := make(map[string]struct{}, len(transportGeneratedImportNames))
	for _, name := range transportGeneratedImportNames {
		reserved[name] = struct{}{}
	}

	for name := range literalTransportImportNames(t) {
		_, ok := reserved[name]
		require.Truef(t, ok, "generated import name %q is missing from transportGeneratedImportNames", name)
	}
}

// literalTransportImportNames collects names emitted by literal import
// constructors. Dynamic service, view, and user-type import paths are
// deliberately excluded: their aliases are allocated from the design scope.
func literalTransportImportNames(t *testing.T) map[string]struct{} {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	directories := []string{
		filepath.Dir(thisFile),
		filepath.Join(filepath.Dir(thisFile), "..", "..", "jsonrpc", "codegen"),
	}
	names := make(map[string]struct{})
	for _, directory := range directories {
		entries, err := filepath.Glob(filepath.Join(directory, "*.go"))
		require.NoError(t, err)
		for _, filename := range entries {
			if strings.HasSuffix(filename, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
			require.NoError(t, err)
			if !fileEmitsDynamicTransportImport(file) {
				continue
			}
			ast.Inspect(file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.CompositeLit:
					if name, ok := literalImportSpecName(node); ok {
						names[name] = struct{}{}
					}
				case *ast.CallExpr:
					if name, ok := literalLoomImportName(node); ok {
						names[name] = struct{}{}
					}
				}
				return true
			})
		}
	}
	return names
}

func fileEmitsDynamicTransportImport(file *ast.File) bool {
	emitsDynamicImport := false
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		selector, ok := literal.Type.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "ImportSpec" {
			return true
		}
		for _, element := range literal.Elts {
			field, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := field.Key.(*ast.Ident)
			if !ok || key.Name != "Path" {
				continue
			}
			if _, ok := stringLiteral(field.Value); !ok {
				emitsDynamicImport = true
				return false
			}
		}
		return true
	})
	return emitsDynamicImport
}

func literalImportSpecName(literal *ast.CompositeLit) (string, bool) {
	selector, ok := literal.Type.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "ImportSpec" {
		return "", false
	}

	var importPath, explicitName string
	for _, element := range literal.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := field.Key.(*ast.Ident)
		if !ok {
			continue
		}
		value, ok := stringLiteral(field.Value)
		if !ok {
			continue
		}
		switch key.Name {
		case "Path":
			importPath = value
		case "Name":
			explicitName = value
		}
	}
	if importPath == "" {
		return "", false
	}
	if explicitName != "" {
		return explicitName, true
	}
	return path.Base(importPath), true
}

func literalLoomImportName(call *ast.CallExpr) (string, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	switch selector.Sel.Name {
	case "LoomNamedImport":
		if len(call.Args) != 2 {
			return "", false
		}
		return stringLiteral(call.Args[1])
	case "LoomImport":
		if len(call.Args) != 1 {
			return "", false
		}
		rel, ok := stringLiteral(call.Args[0])
		if !ok {
			return "", false
		}
		if rel == "" {
			return "loom", true
		}
		return path.Base(rel), true
	default:
		return "", false
	}
}

func stringLiteral(expression ast.Expr) (string, bool) {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	return strings.Trim(literal.Value, `"`), true
}
