package codegen

import (
	"strings"

	"github.com/dave/jennifer/jen"
)

// NewJenniferSection builds a Jennifer-backed section.
func NewJenniferSection(name string, build func(*jen.Statement)) Section {
	return &JenniferSection{Name: name, Build: build}
}

// NewRawSection builds a raw source-backed section.
func NewRawSection(name, source string) Section {
	return &RawSection{Name: name, Source: source}
}

// MustJenniferSection builds a Jennifer-backed section.
func MustJenniferSection(name string, build func(*jen.Statement)) Section {
	return NewJenniferSection(name, build)
}

// Doc appends a wrapped Go doc comment followed by a blank line.
func Doc(stmt *jen.Statement, text string) *jen.Statement {
	return stmt.Comment(Comment(text)).Line()
}

// CommentBlock appends a wrapped Go comment block line by line.
func CommentBlock(stmt *jen.Statement, text string) *jen.Statement {
	for _, line := range strings.Split(Comment(text), "\n") {
		stmt.Comment(strings.TrimPrefix(line, "// ")).Line()
	}
	return stmt
}

// Expr renders a precomputed Go expression as-is.
func Expr(code string) *jen.Statement {
	return jen.Id(code)
}

// TypeRef renders a precomputed Go type reference as-is.
func TypeRef(ref string) *jen.Statement {
	return Expr(ref)
}
