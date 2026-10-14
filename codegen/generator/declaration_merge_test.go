package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	d "github.com/CaliLuke/loom/dsl"
)

func TestSharedDeclarationMerge(t *testing.T) {
	cases := []struct {
		name     string
		sections []string
		sources  []string
		counts   map[string]int
		conflict string
	}{
		{name: "same category distinct types", sections: []string{"service-user-type", "service-user-type"}, sources: []string{"type FooBAR struct{}", "type FooBar struct{}"}, counts: map[string]int{"type FooBAR": 1, "type FooBar": 1}},
		{name: "same category different receivers", sections: []string{"method", "method"}, sources: []string{"type A struct{}\nfunc (*A) Marker() {}", "type B struct{}\nfunc (*B) Marker() {}"}, counts: map[string]int{"func (*A) Marker": 1, "func (*B) Marker": 1}},
		{name: "overlapping declaration sections", sections: []string{"union", "union"}, sources: []string{"type Shared struct{}\nfunc (*Shared) First() {}", "type Shared struct{}\nfunc (*Shared) Second() {}"}, counts: map[string]int{"type Shared": 1, "func (*Shared) First": 1, "func (*Shared) Second": 1}},
		{name: "duplicate under different categories", sections: []string{"type", "result"}, sources: []string{"type Shared struct{}", "type Shared struct{}"}, counts: map[string]int{"type Shared": 1}},
		{name: "conflicting types", sections: []string{"type", "type"}, sources: []string{"type Shared string", "type Shared int"}, conflict: "Shared"},
		{name: "conflicting method receiver forms", sections: []string{"first", "second"}, sources: []string{"type Shared struct{}\nfunc (Shared) Value() int { return 1 }", "func (*Shared) Value() int { return 2 }"}, conflict: "Value"},
		{name: "const groups retain iota", sections: []string{"first", "second"}, sources: []string{"const ( A = iota; B )", "const ( A = iota; B )"}, counts: map[string]int{"A = iota": 1}},
		{name: "overlapping const groups rejected", sections: []string{"const", "const"}, sources: []string{"const ( A = iota; B )", "const ( B = iota; C )"}, conflict: "B"},
		{name: "init occurrences", sections: []string{"init", "init"}, sources: []string{"func init() {}", "func init() {}"}, counts: map[string]int{"func init()": 2}},
		{name: "blank occurrences", sections: []string{"blank", "blank"}, sources: []string{"var _ = 1", "var _ = 1"}, counts: map[string]int{"var _ = 1": 2}},
		{name: "cross kind namespace", sections: []string{"type", "value"}, sources: []string{"type Shared string", "var Shared = 1"}, conflict: "Shared"},
		{name: "generic method", sections: []string{"type", "method"}, sources: []string{"type Shared[T any] struct{}\nfunc (*Shared[T]) Value() int { return 1 }", "func (*Shared[T]) Value() int { return 1 }"}, counts: map[string]int{"Value() int": 1}},
		{name: "independent type specs", sections: []string{"type", "type"}, sources: []string{"type (A string; B int)", "type (B int; C bool)"}, counts: map[string]int{"A string": 1, "B int": 1, "C bool": 1}},
		{name: "atomic tuple overlap", sections: []string{"var", "var"}, sources: []string{"var A, B = 1, 2", "var B, C = 2, 3"}, conflict: "B"},
		{name: "transitive tuple overlap", sections: []string{"var", "var", "var"}, sources: []string{"var A, B = 1, 2", "var B, C = 2, 3", "var C, D = 3, 4"}, conflict: "B, C"},
		{name: "mixed blank effect rejected", sections: []string{"var", "var"}, sources: []string{"var A, _ = 1, 2", "var A, _ = 1, 2"}, conflict: "A"},
		{name: "documentation is not definition", sections: []string{"type", "result"}, sources: []string{"// First service.\ntype Shared struct{}", "// Second service.\ntype Shared struct{}"}, counts: map[string]int{"type Shared": 1}},
		{name: "function directives retained", sections: []string{"first", "second"}, sources: []string{"//go:noinline\nfunc Shared() {}", "func Shared() {}"}, conflict: "Shared"},
		{name: "type directives retained", sections: []string{"first", "second"}, sources: []string{"//go:notinheap\ntype Shared struct{}", "type Shared struct{}"}, conflict: "Shared"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var previousError string
			for _, reverse := range []bool{false, true} {
				var files []*codegen.File
				for i := range tc.sources {
					j := i
					if reverse {
						j = len(tc.sources) - 1 - i
					}
					files = append(files, &codegen.File{Path: "gen/shared.go", Sections: []codegen.Section{codegen.Header("Shared", "shared", nil), codegen.NewRawSection(tc.sections[j], tc.sources[j]+"\n")}})
				}
				merged := mergeFilesByPath(files)
				require.Len(t, merged, 1)
				path, err := merged[0].Render(t.TempDir())
				if tc.conflict != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.conflict)
					if previousError != "" {
						require.Equal(t, previousError, err.Error())
					}
					previousError = err.Error()
					continue
				}
				require.NoError(t, err)
				content, err := os.ReadFile(path)
				require.NoError(t, err)
				for s, n := range tc.counts {
					require.Equal(t, n, strings.Count(string(content), s), "%q in\n%s", s, content)
				}
			}
		})
	}
}

func TestSharedDeclarationFilenameCollision(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{Service}, nil
	}
	codegen.RunDSL(t, func() {
		d.API("collision", func() {
		})
		a := d.Type("FooBAR", func() {
			d.Meta("struct:pkg:path", "types/common")
			d.Attribute("first", d.String)
		})
		b := d.Type("FooBar", func() {
			d.Meta("struct:pkg:path", "types/common")
			d.Attribute("second", d.Int)
		})
		d.Service("alpha", func() {
			d.Method("read", func() {
				d.Result(a)
			})
		})
		d.Service("beta", func() {
			d.Method("read", func() {
				d.Result(b)
			})
		})
	})
	dir := t.TempDir()
	_, err := Generate(dir, "gen", "github.com/CaliLuke/loom/codegen/generator", false)
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(dir, "gen/types/common/foo_bar.go"))
	require.NoError(t, err)
	require.Contains(t, string(content), "type FooBAR struct")
	require.Contains(t, string(content), "type FooBar struct")
}
