package codegen

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
)

// commentFuzzSeeds mirrors design descriptions: multi-paragraph prose,
// markdown, code samples, and strings that try to close or open comments.
var commentFuzzSeeds = []string{
	"",
	"Widget documents a generated type.",
	"Widget may return the following errors:\n- alpha\n- beta",
	"First paragraph.\n\nSecond paragraph.",
	"\nLeading blank line.",
	"   \n\tindented",
	"Use `backticks` and \"quotes\".",
	"Ends block */ func injected() {} /*",
	"\nfoo */ func injected() {} /*",
	"/* opens a block\n*/ func injected() {} /*",
	"// already a comment",
	"line one\r\nline two",
	"Unicode: 日本語 — ümlaut",
	"invalid \xff utf8",
	"nul \x00 byte",
	"bom \ufeff inside",
	strings.Repeat("word ", 40),
	strings.Repeat("x", 120),
}

// FuzzDocComment checks that Doc and CommentBlock always render comments that
// the Go parser accepts, that never swallow or inject declarations, and that
// preserve the words of the input text.
func FuzzDocComment(f *testing.F) {
	for _, s := range commentFuzzSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		for name, add := range map[string]func(*jen.Statement, string) *jen.Statement{
			"Doc":          Doc,
			"CommentBlock": CommentBlock,
		} {
			checkRenderedComment(t, name, text, add)
		}
	})
}

func checkRenderedComment(t *testing.T, name, text string, add func(*jen.Statement, string) *jen.Statement) {
	t.Helper()
	file := jen.NewFile("p")
	stmt := jen.Null()
	add(stmt, text)
	stmt.Var().Id("Documented").Int()
	file.Add(stmt)
	file.Var().Id("Sentinel").Int()
	var buf bytes.Buffer
	if err := file.Render(&buf); err != nil {
		t.Errorf("%s(%q) renders invalid Go: %v", name, text, err)
		return
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "p.go", buf.Bytes(), parser.ParseComments)
	if err != nil {
		t.Errorf("%s(%q) produced unparsable Go: %v\n%s", name, text, err, buf.String())
		return
	}
	var names []string
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			names = append(names, "non-var declaration")
			continue
		}
		for _, spec := range gen.Specs {
			if vs, ok := spec.(*ast.ValueSpec); ok {
				for _, id := range vs.Names {
					names = append(names, id.Name)
				}
			}
		}
	}
	if strings.Join(names, ",") != "Documented,Sentinel" {
		t.Errorf("%s(%q) changed the declarations to %v\n%s", name, text, names, buf.String())
		return
	}
	var words []string
	for _, group := range parsed.Comments {
		for _, c := range group.List {
			if !strings.HasPrefix(c.Text, "//") {
				t.Errorf("%s(%q) rendered a block comment %q", name, text, c.Text)
				continue
			}
			words = append(words, strings.Fields(c.Text[2:])...)
		}
	}
	if want := expectedCommentWords(text); strings.Join(words, " ") != strings.Join(want, " ") {
		t.Errorf("%s(%q) comment words = %q, want %q", name, text, words, want)
	}
}

// expectedCommentWords returns the words a rendered comment must contain:
// the input words with bytes that Go source cannot hold (invalid UTF-8, NUL,
// and byte order marks) replaced by U+FFFD.
func expectedCommentWords(text string) []string {
	var b strings.Builder
	for _, r := range text {
		switch r {
		case 0, '\ufeff':
			r = '\ufffd'
		}
		b.WriteRune(r)
	}
	return strings.Fields(b.String())
}
