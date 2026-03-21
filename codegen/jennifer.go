package codegen

import "github.com/dave/jennifer/jen"

// NewJenniferSection builds a Jennifer-backed section.
func NewJenniferSection(name string, build func(*jen.Statement)) Section {
	return &JenniferSection{Name: name, Build: build}
}

// MustJenniferSection builds a Jennifer-backed section.
func MustJenniferSection(name string, build func(*jen.Statement)) Section {
	return NewJenniferSection(name, build)
}

// Doc appends a wrapped Go doc comment followed by a blank line.
func Doc(stmt *jen.Statement, text string) *jen.Statement {
	return stmt.Comment(Comment(text)).Line()
}

// Expr renders a precomputed Go expression as-is.
func Expr(code string) *jen.Statement {
	return jen.Id(code)
}

// TypeRef renders a precomputed Go type reference as-is.
func TypeRef(ref string) *jen.Statement {
	return Expr(ref)
}
