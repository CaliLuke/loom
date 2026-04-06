package codegen

import (
	"bytes"

	"github.com/dave/jennifer/jen"

	rootcodegen "github.com/CaliLuke/loom/codegen"
)

func renderJen(stmt *jen.Statement) string {
	if stmt == nil {
		return ""
	}
	var buf bytes.Buffer
	_ = stmt.Render(&buf)
	return buf.String()
}

func renderJenLine(stmt *jen.Statement) string {
	rendered := renderJen(stmt)
	if rendered != "" && rendered[len(rendered)-1] != '\n' {
		rendered += "\n"
	}
	return rendered
}

func exprCode(code string) *jen.Statement {
	return rootcodegen.Expr(code)
}
