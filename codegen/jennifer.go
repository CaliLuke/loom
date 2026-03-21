package codegen

import (
	"bytes"
	"fmt"

	"github.com/dave/jennifer/jen"
)

// JenniferSection renders a Jennifer statement into a file section.
func JenniferSection(name string, build func(*jen.Statement)) (*SectionTemplate, error) {
	stmt := jen.Empty()
	build(stmt)

	var buf bytes.Buffer
	if err := stmt.Render(&buf); err != nil {
		return nil, fmt.Errorf("render jennifer section %q: %w", name, err)
	}

	return &SectionTemplate{Name: name, Source: buf.String()}, nil
}

// MustJenniferSection renders a Jennifer statement into a file section and
// panics if rendering fails.
func MustJenniferSection(name string, build func(*jen.Statement)) *SectionTemplate {
	section, err := JenniferSection(name, build)
	if err != nil {
		panic(err)
	}
	return section
}
