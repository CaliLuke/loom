package codegen

import (
	"bytes"

	"github.com/dave/jennifer/jen"
)

func renderJen(stmt *jen.Statement) string {
	if stmt == nil {
		return ""
	}
	var buf bytes.Buffer
	_ = stmt.Render(&buf)
	return buf.String()
}
