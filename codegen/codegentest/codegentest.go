// Package codegentest provides utilities to assist writing unit test for
// codegen packages.
package codegentest

import (
	"strings"

	"goa.design/goa/v3/codegen"
)

// Sections can be used to extract the code sections that match a path suffix
// and a section name.
func Sections(files []*codegen.File, pathSuffix string, sectionName string) []codegen.Section {
	var result []codegen.Section
	for _, file := range files {
		if !strings.HasSuffix(file.Path, pathSuffix) {
			continue
		}

		for _, section := range file.Section(sectionName) {
			if section.SectionName() == sectionName {
				result = append(result, section)
			}
		}
	}
	return result
}
