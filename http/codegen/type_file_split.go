package codegen

import (
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

const typeSplitSectionThreshold = 24

type typeFileBucket struct {
	suffix  string
	title   string
	section []codegen.Section
}

// splitTypeFileIfLarge splits the types file file of package pkg by kind of
// section when it holds too many sections. The files it splits file into
// list the imports of file, so they qualify the types of the struct:pkg:path
// packages as the sections of file do.
func splitTypeFileIfLarge(file *codegen.File, title, pkg string) []*codegen.File {
	sections := file.AllSections()
	if len(sections)-1 <= typeSplitSectionThreshold {
		return []*codegen.File{file}
	}
	imports := codegen.HeaderDataForSection(file.HeaderSection()).Imports

	buckets := []typeFileBucket{
		{suffix: "requests", title: title + " request types"},
		{suffix: "responses", title: title + " response types"},
		{suffix: "unions", title: title + " union types"},
		{suffix: "validation", title: title + " validation helpers"},
		{suffix: "helpers", title: title + " helper types"},
	}
	for _, section := range sections[1:] {
		idx := typeSectionBucket(section.SectionName())
		buckets[idx].section = append(buckets[idx].section, section)
	}

	files := make([]*codegen.File, 0, len(buckets))
	dir := filepath.Dir(file.Path)
	for _, bucket := range buckets {
		if len(bucket.section) == 0 {
			continue
		}
		path := filepath.Join(dir, "types_"+bucket.suffix+".go")
		fileSections := append([]codegen.Section{codegen.Header(bucket.title, pkg, imports)}, bucket.section...)
		files = append(files, &codegen.File{Path: path, Sections: fileSections})
	}
	return files
}

func typeSectionBucket(name string) int {
	switch {
	case strings.Contains(name, "union-type"):
		return 2
	case strings.Contains(name, "validate"):
		return 3
	case strings.Contains(name, "request"):
		return 0
	case strings.Contains(name, "response") || strings.Contains(name, "error"):
		return 1
	default:
		return 4
	}
}
