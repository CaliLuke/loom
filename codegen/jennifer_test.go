package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
)

func TestJenniferSection(t *testing.T) {
	section := NewJenniferSection("answer", func(stmt *jen.Statement) {
		stmt.Comment(Comment("Answer returns the generated answer.")).Line()
		stmt.Func().Id("Answer").Params().Int().Block(
			jen.Return(jen.Lit(42)),
		)
	})

	code := SectionCode(t, section)
	if code != `// Answer returns the generated answer.
func Answer() int {
	return 42
}
` {
		t.Fatalf("unexpected code:\n%s", code)
	}
}

func TestJenniferHelpers(t *testing.T) {
	var buf bytes.Buffer
	stmt := jen.Empty()
	Doc(stmt, "Widget documents a generated type.")
	stmt.Var().Id("x").Op("=").Add(Expr("&body"))
	stmt.Line()
	stmt.Var().Id("y").Add(TypeRef("mypkg.Type"))
	if err := stmt.Render(&buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	got := strings.TrimSpace(strings.ReplaceAll(buf.String(), "\t", ""))
	if want := "// Widget documents a generated type.\nvar x = &body\nvar y mypkg.Type"; got != want {
		t.Fatalf("unexpected helper code:\n%s", got)
	}
}
