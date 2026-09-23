package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"
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

func TestCommentBlock(t *testing.T) {
	var buf bytes.Buffer
	stmt := jen.Empty()
	CommentBlock(stmt, "Widget may return the following errors:\n- alpha\n- beta")
	stmt.Func().Id("Answer").Params().Block(
		jen.Return(),
	)
	if err := stmt.Render(&buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	got := strings.TrimSpace(strings.ReplaceAll(buf.String(), "\t", ""))
	want := strings.TrimSpace(strings.ReplaceAll(`// Widget may return the following errors:
// - alpha
// - beta
func Answer() {
	return
}`, "\t", ""))
	if got != want {
		t.Fatalf("unexpected block code:\n%s", buf.String())
	}
}

func TestCommentHelpersCannotEscapeComments(t *testing.T) {
	cases := map[string]struct {
		text string
		want string
	}{
		"leading blank line closes block comment": {
			text: "\nfoo */ func injected() {} /*",
			want: "// foo */ func injected() {} /*\nvar X int",
		},
		"line opening a block comment": {
			text: "/* opens a block\n*/ func injected() {} /*",
			want: "// /* opens a block\n// */ func injected() {} /*\nvar X int",
		},
		"line starting with a line comment": {
			text: "// already a comment",
			want: "// // already a comment\nvar X int",
		},
		"blank paragraph separator": {
			text: "First.\n\nSecond.",
			want: "// First.\n//\n// Second.\nvar X int",
		},
		"carriage returns break lines": {
			text: "one\rtwo\r\nthree",
			want: "// one\n// two\n// three\nvar X int",
		},
		"empty text": {
			text: "",
			want: "//\nvar X int",
		},
		"bytes Go source cannot hold": {
			text: "nul \x00 bom \ufeff bad \xff",
			want: "// nul \ufffd bom \ufffd bad \ufffd\nvar X int",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, renderCommentedVar(t, Doc, tc.text), "Doc")
			require.Equal(t, tc.want, renderCommentedVar(t, CommentBlock, tc.text), "CommentBlock")
		})
	}
}

func TestLineComment(t *testing.T) {
	long := "SlashWithBasePathNoTrailingBasePathNoTrailingPath returns the URL path to the BasePathNoTrailing service SlashWithBasePathNoTrailing HTTP endpoint."
	cases := map[string]struct {
		text string
		want string
	}{
		"long line is not wrapped": {text: long + " ", want: "// " + long},
		"newline cannot inject":    {text: "a\n*/ func Injected() {} /*", want: "// a\n// */ func Injected() {} /*"},
		"blank lines":              {text: "\na\n\nb\n", want: "// a\n//\n// b"},
		"empty":                    {text: "", want: "//"},
		"unsafe bytes":             {text: "nul \x00 bom \ufeff cr\rbad \xff", want: "// nul \ufffd bom \ufffd cr\n// bad \ufffd"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := LineComment(tc.text)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.want+"\nvar X int", renderCommentedVar(t, func(stmt *jen.Statement, text string) *jen.Statement {
				return stmt.Comment(LineComment(text)).Line()
			}, tc.text))
		})
	}
}

func TestCommentSanitizesUnsafeBytes(t *testing.T) {
	cases := map[string]struct {
		elems []string
		want  string
	}{
		"plain":        {elems: []string{"Widget documents a type."}, want: "// Widget documents a type."},
		"nul and bom":  {elems: []string{"nul \x00 bom \ufeff"}, want: "// nul \ufffd bom \ufffd"},
		"invalid utf8": {elems: []string{"bad \xff", "ok"}, want: "// bad \ufffd\n// ok"},
		"carriage":     {elems: []string{"one\rtwo"}, want: "// one\n// two"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, Comment(tc.elems...))
		})
	}
}

func renderCommentedVar(t *testing.T, add func(*jen.Statement, string) *jen.Statement, text string) string {
	t.Helper()
	file := jen.NewFile("p")
	stmt := jen.Null()
	add(stmt, text)
	stmt.Var().Id("X").Int()
	file.Add(stmt)
	var buf bytes.Buffer
	require.NoError(t, file.Render(&buf))
	return strings.TrimSpace(strings.TrimPrefix(buf.String(), "package p\n"))
}
