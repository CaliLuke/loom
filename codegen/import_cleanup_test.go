package codegen

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFinalizeGoSourceKeepsUnnamedImportWithUnknownPackageName(t *testing.T) {
	source := []byte(`package sample

import "example.com/acme/go-uuid"

type ID struct {
	Value uuid.UUID
}
`)

	formatted, err := finalizeGoSource("sample.go", source)
	require.NoError(t, err)
	require.Contains(t, string(formatted), `"example.com/acme/go-uuid"`)
	require.Contains(t, string(formatted), "uuid.UUID")
}

func TestFinalizeGoSourceRemovesOrdinaryUnusedImport(t *testing.T) {
	source := []byte(`package sample

import (
	"fmt"
	"strings"
)

func Value() string {
	return strings.TrimSpace(" value ")
}
`)

	formatted, err := finalizeGoSource("sample.go", source)
	require.NoError(t, err)
	require.NotContains(t, string(formatted), `"fmt"`)
	require.True(t, strings.Contains(string(formatted), `"strings"`))
}

func TestBuildImportMapKeepsSameLocalNameDifferentPaths(t *testing.T) {
	source := []byte(`package sample

import (
	"example.com/one/client"
	"example.com/two/client"
)
`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", source, parser.ParseComments)
	require.NoError(t, err)

	imports := buildImportMap(file)

	require.Len(t, imports, 2)
	require.Contains(t, imports, importKey{localName: "client", path: "example.com/one/client"})
	require.Contains(t, imports, importKey{localName: "client", path: "example.com/two/client"})
}
