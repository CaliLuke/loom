package codegen

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionFilesUseGenericSections(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}

	var offenders []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".codex-upstream", "gen", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "codegen/file.go" || rel == "codegen/header.go" {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(b)
		if strings.Contains(source, "SectionTemplates:") ||
			strings.Contains(source, ".SectionTemplates") ||
			strings.Contains(source, "NewTemplateSection(") ||
			strings.Contains(source, "&codegen.SectionTemplate{") ||
			strings.Contains(source, "&SectionTemplate{") {
			offenders = append(offenders, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Fatalf("production generators must use File.Sections, not File.SectionTemplates:\n%s", strings.Join(offenders, "\n"))
	}
}
