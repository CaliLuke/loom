package codegen

import (
	"bytes"
	"io"
	"maps"
	"strings"
	"text/template"

	"github.com/dave/jennifer/jen"
)

type (
	// JenniferBuilder renders one Go section using Jennifer.
	JenniferBuilder func(*jen.Statement)

	// JenniferSection renders a file section from a typed Jennifer statement.
	JenniferSection struct {
		// Name is the stable section identifier used by tests and merge logic.
		Name string
		// Build appends code to the provided statement.
		Build JenniferBuilder
	}

	// RawSection renders an exact file section from a precomputed source string.
	RawSection struct {
		// Name is the stable section identifier used by tests and merge logic.
		Name string
		// Source is written as-is.
		Source string
	}

	// RenderSection renders an exact file section by calling a source builder.
	RenderSection struct {
		// Name is the stable section identifier used by tests and merge logic.
		Name string
		// Render computes the section source when written.
		Render func() string
	}

	// TextTemplateSection renders a section from a text/template source.
	TextTemplateSection struct {
		// Name is the stable section identifier used by tests and merge logic.
		Name string
		// Source is the text/template source.
		Source string
		// FuncMap lists section-local template functions.
		FuncMap map[string]any
		// Data is passed to the template during execution.
		Data any
	}
)

// SectionName returns the stable section identifier.
func (s *JenniferSection) SectionName() string {
	return s.Name
}

// Write writes the rendered Jennifer section to the given writer.
func (s *JenniferSection) Write(w io.Writer) error {
	stmt := jen.Empty()
	if s.Build != nil {
		s.Build(stmt)
	}
	var buf bytes.Buffer
	if err := stmt.Render(&buf); err != nil {
		return err
	}
	lines := strings.Split(buf.String(), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(strings.TrimPrefix(line, "\t"), " ")
	}
	_, err := io.WriteString(w, strings.Join(lines, "\n"))
	return err
}

// SectionName returns the stable section identifier.
func (s *RawSection) SectionName() string {
	return s.Name
}

// Write writes the raw section source to the given writer.
func (s *RawSection) Write(w io.Writer) error {
	_, err := io.WriteString(w, s.Source)
	return err
}

// SectionName returns the stable section identifier.
func (s *RenderSection) SectionName() string {
	return s.Name
}

// Write writes the rendered section source to the given writer.
func (s *RenderSection) Write(w io.Writer) error {
	if s.Render == nil {
		return nil
	}
	_, err := io.WriteString(w, s.Render())
	return err
}

// SectionName returns the stable section identifier.
func (s *TextTemplateSection) SectionName() string {
	return s.Name
}

// Write renders the template-backed section to w.
func (s *TextTemplateSection) Write(w io.Writer) error {
	funcs := TemplateFuncs()
	maps.Copy(funcs, s.FuncMap)
	tmpl, err := template.New(s.Name).Funcs(funcs).Parse(s.Source)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, s.Data)
}
