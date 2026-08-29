package openapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type exportClass string

const (
	// externalSchemaContract covers the types and constants referenced by the
	// exported v3 document model. Removing these would break active consumers.
	externalSchemaContract exportClass = "external schema contract"
	// activeFrameworkHelper covers exports still called by Loom's renderer.
	// Repository and release-history searches found no external Loom consumers
	// for the removed JSON Hyper-Schema generator, so those APIs are intentionally
	// absent instead of retained as compatibility aliases.
	activeFrameworkHelper exportClass = "active framework helper"
)

var supportedExports = map[string]exportClass{
	"AdditionalPropertiesFromExpr": activeFrameworkHelper,
	"Array":                        externalSchemaContract,
	"Boolean":                      externalSchemaContract,
	"ClosedObjectModeFromExpr":     activeFrameworkHelper,
	"Discriminator":                externalSchemaContract,
	"DocsFromExpr":                 activeFrameworkHelper,
	"ExtensionsFromExpr":           activeFrameworkHelper,
	"ExternalDocs":                 externalSchemaContract,
	"Integer":                      externalSchemaContract,
	"MarshalJSON":                  activeFrameworkHelper,
	"MarshalYAML":                  activeFrameworkHelper,
	"MergeExtensions":              activeFrameworkHelper,
	"MustGenerate":                 activeFrameworkHelper,
	"NewSchema":                    activeFrameworkHelper,
	"Null":                         externalSchemaContract,
	"Number":                       externalSchemaContract,
	"Object":                       externalSchemaContract,
	"Schema":                       externalSchemaContract,
	"Schema.MarshalJSON":           externalSchemaContract,
	"Schema.MarshalYAML":           externalSchemaContract,
	"ScopedExtensionsFromExpr":     activeFrameworkHelper,
	"String":                       externalSchemaContract,
	"Tag":                          externalSchemaContract,
	"Tag.MarshalJSON":              externalSchemaContract,
	"Tag.MarshalYAML":              externalSchemaContract,
	"TagNamesFromExpr":             activeFrameworkHelper,
	"TagsFromExpr":                 activeFrameworkHelper,
	"Type":                         externalSchemaContract,
	"XML":                          externalSchemaContract,
}

func TestSupportedExportSurface(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	files := make([]*ast.File, 0, len(entries))
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fileSet, name, nil, 0)
		require.NoError(t, parseErr)
		files = append(files, file)
	}

	actual := make([]string, 0, len(supportedExports))
	for _, file := range files {
		for _, decl := range file.Decls {
			switch declaration := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					for _, name := range exportedSpecNames(spec) {
						actual = append(actual, name)
					}
				}
			case *ast.FuncDecl:
				if !declaration.Name.IsExported() {
					continue
				}
				name := declaration.Name.Name
				if receiver := exportedReceiverName(declaration); receiver != "" {
					name = receiver + "." + name
				}
				actual = append(actual, name)
			}
		}
	}

	expected := make([]string, 0, len(supportedExports))
	for name := range supportedExports {
		expected = append(expected, name)
	}
	sort.Strings(actual)
	sort.Strings(expected)
	require.Equal(t, expected, actual)
}

func exportedSpecNames(spec ast.Spec) []string {
	switch declaration := spec.(type) {
	case *ast.TypeSpec:
		if declaration.Name.IsExported() {
			return []string{declaration.Name.Name}
		}
	case *ast.ValueSpec:
		var names []string
		for _, name := range declaration.Names {
			if name.IsExported() {
				names = append(names, name.Name)
			}
		}
		return names
	}
	return nil
}

func exportedReceiverName(declaration *ast.FuncDecl) string {
	if declaration.Recv == nil || len(declaration.Recv.List) != 1 {
		return ""
	}
	receiver := declaration.Recv.List[0].Type
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		receiver = pointer.X
	}
	name, ok := receiver.(*ast.Ident)
	if !ok || !name.IsExported() {
		return ""
	}
	return name.Name
}
