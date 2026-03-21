package codegen

import (
	"io"

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
	return stmt.Render(w)
}
