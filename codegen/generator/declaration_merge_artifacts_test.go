package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestSharedDeclarationArtifactAdmission(t *testing.T) {
	for _, tc := range []struct {
		name        string
		left, right []codegen.Section
		error       string
	}{
		{name: "different header packages", left: []codegen.Section{codegen.Header("", "alpha", nil)}, right: []codegen.Section{codegen.Header("", "beta", nil)}, error: "conflicting package declarations in gen/shared.go: alpha, beta"},
		{name: "different raw package declarations", left: []codegen.Section{codegen.NewRawSection("source-header", "package alpha\n")}, right: []codegen.Section{codegen.NewRawSection("source-header", "package beta\n")}, error: "conflicting package declarations in gen/shared.go: alpha, beta"},
		{name: "explicit import binding conflict", left: []codegen.Section{codegen.Header("", "shared", []*codegen.ImportSpec{codegen.NewImport("value", "fmt")})}, right: []codegen.Section{codegen.Header("", "shared", []*codegen.ImportSpec{codegen.NewImport("value", "time")})}, error: "conflicting import bindings in gen/shared.go: value"},
		{name: "invalid syntax", left: []codegen.Section{codegen.Header("", "shared", nil)}, right: []codegen.Section{codegen.NewRawSection("type", "type Broken struct {\n")}, error: "expected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, reverse := range []bool{false, true} {
				groups := [][]codegen.Section{tc.left, tc.right}
				if reverse {
					groups[0], groups[1] = groups[1], groups[0]
				}
				files := []*codegen.File{{Path: "gen/shared.go", Sections: groups[0]}, {Path: "gen/shared.go", Sections: groups[1]}}
				_, err := mergeFilesByPath(files)[0].Render(t.TempDir())
				require.ErrorContains(t, err, tc.error)
			}
		})
	}
}

func TestSharedDeclarationPreservesInitialization(t *testing.T) {
	files := []*codegen.File{
		{Path: "shared.go", Sections: []codegen.Section{
			codegen.Header("", "shared", []*codegen.ImportSpec{codegen.SimpleImport("fmt")}),
			codegen.NewRawSection("types", `var Events []int
func record(n int) int { Events = append(Events,n); return n }
var _ = record(3)
func init() { record(1) }
const ( A = iota; B )
type Named fmt.Stringer
`),
		}},
		{Path: "shared.go", Sections: []codegen.Section{
			codegen.Header("", "shared", []*codegen.ImportSpec{codegen.SimpleImport("time")}),
			codegen.NewRawSection("types", `var _ = record(3)
func init() { record(2) }
const ( A = iota; B )
type Moment time.Time
`),
		}},
		// This contribution deliberately has no source-header. Its first section
		// must be retained instead of being assumed to be a header.
		{Path: "shared.go", Sections: []codegen.Section{codegen.NewRawSection("types", "const C = 5\n")}},
	}
	dir := t.TempDir()
	_, err := mergeFilesByPath(files)[0].Render(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/merge\n\ngo 1.27.3\n"), 0600))
	const test = `package shared
import ("reflect"; "testing")
func TestValues(t *testing.T) {
 if A!=0 || B!=1 || C!=5 { t.Errorf("constants = %d %d %d", A,B,C) }
 if !reflect.DeepEqual(Events, []int{3,3,1,2}) { t.Errorf("events = %v", Events) }
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shared_test.go"), []byte(test), 0600))
	output, err := testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}

func TestSharedDeclarationUnmergedBoundaries(t *testing.T) {
	t.Run("ordinary single Go file", func(t *testing.T) {
		file := &codegen.File{Path: "shared.go", Sections: []codegen.Section{codegen.Header("", "shared", nil), codegen.NewRawSection("types", "type A string\ntype A int\n")}}
		merged := mergeFilesByPath([]*codegen.File{file})
		require.Same(t, file, merged[0])
		require.Equal(t, 2, len(merged[0].AllSections()))
	})
	t.Run("non Go composition", func(t *testing.T) {
		left := &codegen.File{Path: "shared.txt", Sections: []codegen.Section{codegen.NewRawSection("header", "header\n"), codegen.NewRawSection("first", "first\n")}}
		right := &codegen.File{Path: "shared.txt", Sections: []codegen.Section{codegen.NewRawSection("header", "header\n"), codegen.NewRawSection("second", "second\n")}}
		file := mergeFilesByPath([]*codegen.File{left, right})[0]
		path, err := file.Render(t.TempDir())
		require.NoError(t, err)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "header\nfirst\nsecond\n", string(content))
	})
}
