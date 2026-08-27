package generator

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/designfingerprint"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

// TestGenerateMergesSamePathFiles verifies that when two generators emit content
// targeting the same output path, Generate merges the sections into a single
// file rather than overwriting earlier content. This is a regression test for
// an issue where only a later section (e.g., a union value method) remained and
// the earlier struct definition was lost.
func TestGenerateMergesSamePathFiles(t *testing.T) {
	t.Cleanup(func() { generatorLoader = generators })

	// Fake generators emit two files with identical Path, one containing a
	// type definition and the other containing a method. Without merging, the
	// second write would overwrite the first.
	generatorLoader = func(cmd string) ([]genfunc, error) {
		return []genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "merge_test.go")}
				f.Sections = []codegen.Section{
					codegen.Header("User types", "types", nil),
					codegen.NewRawSection("struct-type", "type MergeTest struct{}\n"),
				}
				return []*codegen.File{f}, nil
			},
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "merge_test.go")}
				f.Sections = []codegen.Section{
					codegen.Header("User types", "types", nil),
					codegen.NewRawSection("method", "func (*MergeTest) Marker() {}\n"),
				}
				return []*codegen.File{f}, nil
			},
		}, nil
	}

	dir := t.TempDir()
	_, err := Generate(dir, "gen", false)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Read the merged output directly from the temp dir regardless of how
	// outputs are relativized by Generate.
	outpath := filepath.Join(dir, codegen.Gendir, "types", "merge_test.go")
	bs, err := os.ReadFile(outpath)
	if err != nil {
		t.Fatalf("failed reading merged file: %v", err)
	}
	content := string(bs)
	if !strings.Contains(content, "type MergeTest struct{}") {
		t.Fatalf("merged file missing struct definition:\n%s", content)
	}
	if !strings.Contains(content, "func (*MergeTest) Marker() {}") {
		t.Fatalf("merged file missing method definition:\n%s", content)
	}
}

func TestGenerateManifestFingerprintsDesignBeforeGeneratorsMutateIt(t *testing.T) {
	originalServices := expr.Root.Services
	t.Cleanup(func() {
		expr.Root.Services = originalServices
		generatorLoader = generators
	})

	var expectedDigest string
	generatorLoader = func(cmd string) ([]genfunc, error) {
		return []genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				var err error
				expectedDigest, err = designfingerprint.Digest(expr.Root, cmd, genpkg, codegen.DesignVersion)
				require.NoError(t, err)
				expr.Root.Services = append(expr.Root.Services, &expr.ServiceExpr{Name: "generator-mutation"})
				return nil, nil
			},
		}, nil
	}

	dir := t.TempDir()
	_, err := Generate(dir, "gen", false)
	require.NoError(t, err)

	manifestData, err := os.ReadFile(filepath.Join(dir, codegen.Gendir, "loom.json"))
	require.NoError(t, err)
	var manifest map[string]string
	require.NoError(t, json.Unmarshal(manifestData, &manifest))
	require.Equal(t, expectedDigest, manifest["design_digest"])
}

// TestGenerateParallelManyFiles verifies that parallel file writing correctly
// handles multiple files (more than runtime.NumCPU()) to exercise the worker
// pool distribution. This ensures all workers process files and all files are
// written correctly.
func TestGenerateParallelManyFiles(t *testing.T) {
	t.Cleanup(func() { generatorLoader = generators })

	// Generate 20 files to ensure we exceed typical CPU counts and exercise
	// the worker pool's work distribution.
	const numFiles = 20
	generatorLoader = func(cmd string) ([]genfunc, error) {
		return []genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				files := make([]*codegen.File, numFiles)
				for i := 0; i < numFiles; i++ {
					f := &codegen.File{
						Path: filepath.Join(codegen.Gendir, "types", filepath.Join("parallel", filepath.Join("file"+string(rune('a'+i%26)), "test"+string(rune('0'+i/26))+".go"))),
					}
					f.Sections = []codegen.Section{
						codegen.Header("Types", "types", nil),
						codegen.NewRawSection("type-def", "type Test"+string(rune('A'+i))+" struct{ ID int }\n"),
					}
					files[i] = f
				}
				return files, nil
			},
		}, nil
	}

	dir := t.TempDir()
	outputs, err := Generate(dir, "gen", false)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	outputs = assertManifestFile(t, dir, outputs)

	// Verify all generator files were written
	if len(outputs) != numFiles {
		t.Fatalf("expected %d output files, got %d", numFiles, len(outputs))
	}

	// Verify each file exists and has correct content
	for i := 0; i < numFiles; i++ {
		outpath := filepath.Join(dir, codegen.Gendir, "types", filepath.Join("parallel", filepath.Join("file"+string(rune('a'+i%26)), "test"+string(rune('0'+i/26))+".go")))
		bs, err := os.ReadFile(outpath)
		if err != nil {
			t.Fatalf("failed reading file %d: %v", i, err)
		}
		content := string(bs)
		expected := "type Test" + string(rune('A'+i)) + " struct"
		if !strings.Contains(content, expected) {
			t.Fatalf("file %d missing expected content %q:\n%s", i, expected, content)
		}
	}
}

// TestGenerateParallelWithMerge verifies that parallel file writing correctly
// handles file merging when multiple generators target the same path. This
// tests the interaction between mergeFilesByPath and parallel rendering.
func TestGenerateParallelWithMerge(t *testing.T) {
	t.Cleanup(func() { generatorLoader = generators })

	// Three generators: first two merge to same path, third is separate.
	// This exercises both merging and parallel writing with NumCPU workers.
	generatorLoader = func(cmd string) ([]genfunc, error) {
		return []genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f1 := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "merged.go")}
				f1.Sections = []codegen.Section{
					codegen.Header("Types", "types", nil),
					codegen.NewRawSection("type1", "type Type1 struct{}\n"),
				}
				return []*codegen.File{f1}, nil
			},
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f2 := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "merged.go")}
				f2.Sections = []codegen.Section{
					codegen.Header("Types", "types", nil),
					codegen.NewRawSection("type2", "type Type2 struct{}\n"),
				}
				return []*codegen.File{f2}, nil
			},
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f3 := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "separate.go")}
				f3.Sections = []codegen.Section{
					codegen.Header("Types", "types", nil),
					codegen.NewRawSection("type3", "type Type3 struct{}\n"),
				}
				return []*codegen.File{f3}, nil
			},
		}, nil
	}

	dir := t.TempDir()
	outputs, err := Generate(dir, "gen", false)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	outputs = assertManifestFile(t, dir, outputs)

	if len(outputs) != 2 {
		t.Fatalf("expected 2 output files, got %d", len(outputs))
	}

	// Verify merged file contains both types
	mergedPath := filepath.Join(dir, codegen.Gendir, "types", "merged.go")
	bs, err := os.ReadFile(mergedPath)
	if err != nil {
		t.Fatalf("failed reading merged file: %v", err)
	}
	content := string(bs)
	if !strings.Contains(content, "type Type1 struct{}") {
		t.Fatalf("merged file missing Type1:\n%s", content)
	}
	if !strings.Contains(content, "type Type2 struct{}") {
		t.Fatalf("merged file missing Type2:\n%s", content)
	}

	// Verify separate file
	separatePath := filepath.Join(dir, codegen.Gendir, "types", "separate.go")
	bs, err = os.ReadFile(separatePath)
	if err != nil {
		t.Fatalf("failed reading separate file: %v", err)
	}
	content = string(bs)
	if !strings.Contains(content, "type Type3 struct{}") {
		t.Fatalf("separate file missing Type3:\n%s", content)
	}
}

// TestGenerateParallelErrorHandling verifies that when file rendering fails
// in the parallel worker pool, the first error is captured and returned while
// other workers continue processing.
func TestGenerateParallelErrorHandling(t *testing.T) {
	t.Cleanup(func() { generatorLoader = generators })

	// Create multiple files where some will fail to render due to invalid paths.
	// Worker pool should capture first error but continue processing other files.
	generatorLoader = func(cmd string) ([]genfunc, error) {
		return []genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				files := make([]*codegen.File, 5)
				for i := 0; i < 5; i++ {
					f := &codegen.File{
						Path: filepath.Join(codegen.Gendir, "types", "file"+string(rune('0'+i))+".go"),
					}
					f.Sections = []codegen.Section{
						codegen.Header("Types", "types", nil),
						codegen.NewRawSection("type", "type T"+string(rune('0'+i))+" struct{}\n"),
					}
					// Make file 2 fail by adding an invalid path character after writing starts
					if i == 2 {
						// Use a FinalizeFunc that returns an error
						f.FinalizeFunc = func(fp string) error {
							return os.ErrInvalid
						}
					}
					files[i] = f
				}
				return files, nil
			},
		}, nil
	}

	dir := t.TempDir()
	_, err := Generate(dir, "gen", false)
	if err == nil {
		t.Fatal("expected error from parallel generation, got nil")
	}
	// Verify we got an error (the first one encountered)
	if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "finalize") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestGenerateParallelSingleFile verifies that parallel file writing works
// correctly with just a single file (minimal parallelism edge case).
func TestGenerateParallelSingleFile(t *testing.T) {
	t.Cleanup(func() { generatorLoader = generators })

	generatorLoader = func(cmd string) ([]genfunc, error) {
		return []genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "single.go")}
				f.Sections = []codegen.Section{
					codegen.Header("Types", "types", nil),
					codegen.NewRawSection("type", "type Single struct{}\n"),
				}
				return []*codegen.File{f}, nil
			},
		}, nil
	}

	dir := t.TempDir()
	outputs, err := Generate(dir, "gen", false)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	outputs = assertManifestFile(t, dir, outputs)

	if len(outputs) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(outputs))
	}

	outpath := filepath.Join(dir, codegen.Gendir, "types", "single.go")
	bs, err := os.ReadFile(outpath)
	if err != nil {
		t.Fatalf("failed reading file: %v", err)
	}
	content := string(bs)
	if !strings.Contains(content, "type Single struct{}") {
		t.Fatalf("file missing expected content:\n%s", content)
	}
}

// assertManifestFile checks that loom.json was emitted with the correct version
// and design digest, then returns the remaining outputs for further assertions.
func assertManifestFile(t *testing.T, dir string, outputs []string) []string {
	t.Helper()

	versionPath := filepath.Join(codegen.Gendir, "loom.json")

	// Read and validate loom.json content.
	bs, err := os.ReadFile(filepath.Join(dir, versionPath))
	if err != nil {
		t.Fatalf("failed reading loom.json: %v", err)
	}
	var data map[string]string
	if err := json.Unmarshal(bs, &data); err != nil {
		t.Fatalf("loom.json is not valid JSON: %v", err)
	}
	if v := data["loom_version"]; v != loom.Version() {
		t.Fatalf("loom.json version = %q, want %q", v, loom.Version())
	}
	if data["design_digest"] == "" {
		t.Fatal("loom.json design_digest must not be empty")
	}
	if !bytes.HasSuffix(bs, []byte("}\n")) {
		t.Fatalf("loom.json must end with exactly one LF, got final bytes %q", bs[max(0, len(bs)-2):])
	}

	// Filter loom.json out of outputs.
	var rest []string
	for _, o := range outputs {
		if filepath.Base(o) != "loom.json" {
			rest = append(rest, o)
		}
	}
	return rest
}
