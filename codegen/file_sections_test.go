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
		if strings.Contains(source, "NewTemplateSection(") ||
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
		t.Fatalf("production generators must use generic File.Sections:\n%s", strings.Join(offenders, "\n"))
	}
}

func TestAllSectionsAcceptsExternalTemplateSections(t *testing.T) {
	header := Header("External types", "external", nil)
	typeSection := &SectionTemplate{Name: "external-type", Source: "type External struct{}\n"}
	file := &File{SectionTemplates: []*SectionTemplate{header, typeSection}}

	sections := file.AllSections()
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if got := sections[1].SectionName(); got != "external-type" {
		t.Errorf("expected external-type section, got %q", got)
	}
}
