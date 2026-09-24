package codegen

import (
	"strings"
	"unicode/utf8"

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

// NewRenderSection builds a source-backed section from a render callback.
func NewRenderSection(name string, render func() string) Section {
	return &RenderSection{Name: name, Render: render}
}

// NewTextTemplateSection builds a section rendered from a text/template source.
func NewTextTemplateSection(name, source string, funcs map[string]any, data any) Section {
	return &TextTemplateSection{
		Name:    name,
		Source:  source,
		FuncMap: funcs,
		Data:    data,
	}
}

// Doc appends a wrapped Go doc comment followed by a blank line. The comment
// is rendered from CommentLines so arbitrary design text cannot terminate it.
func Doc(stmt *jen.Statement, text string) *jen.Statement {
	return stmt.Comment(strings.Join(CommentLines(text), "\n")).Line()
}

// CommentBlock appends a wrapped Go comment block line by line. The lines come
// from CommentLines so arbitrary design text cannot terminate the comment.
func CommentBlock(stmt *jen.Statement, text string) *jen.Statement {
	for _, line := range CommentLines(text) {
		stmt.Comment(line).Line()
	}
	return stmt
}

// CommentLines wraps text into Go line comments. Every returned line starts
// with "//" and contains no newline, so passing each line to a Jennifer
// Comment call renders it verbatim: text containing "*/", "/*", or a leading
// blank line can never close the comment or inject code. Blank lines inside
// the text become "//" separators that keep a doc comment in one group, and
// leading and trailing blank lines are dropped. Comment replaces invalid
// UTF-8, NUL bytes, and byte order marks, which Go source cannot contain, with
// U+FFFD. The result always holds at least one line.
func CommentLines(text string) []string {
	return commentLines(strings.Split(Comment(text), "\n"))
}

// LineComment renders text as Go line comments without wrapping, for short
// generated sentences that embed design names. Like CommentLines it breaks
// the text on newlines and carriage returns, prefixes every line with "//",
// replaces bytes Go source cannot hold with U+FFFD, trims trailing blanks,
// and drops leading and trailing blank lines, so the result can be passed to
// a Jennifer Comment call or written into raw source and can never close the
// comment or inject code.
func LineComment(text string) string {
	lines := strings.Split(sanitizeCommentText(text), "\n")
	for i, line := range lines {
		if line = strings.TrimRight(line, " \t"); line != "" {
			line = "// " + line
		}
		lines[i] = line
	}
	return strings.Join(commentLines(lines), "\n")
}

// Expr renders a precomputed Go expression as-is.
func Expr(code string) *jen.Statement {
	return jen.Id(code)
}

// TypeRef renders a precomputed Go type reference as-is.
func TypeRef(ref string) *jen.Statement {
	return Expr(ref)
}

// PkgQual renders the identifier name qualified by the package imported under
// alias, as in "alias.name". Unlike jen.Qual, which takes an import path and
// derives the qualifier from its last element, lower-cased with every rune
// outside [a-z0-9] removed, PkgQual renders alias verbatim, so it stays
// correct for the aliases Loom computes for its generated packages (e.g.
// "event_streamerpb").
func PkgQual(alias, name string) *jen.Statement {
	return jen.Id(alias).Dot(name)
}

// commentLines drops the leading and trailing empty entries of lines, turns
// the remaining empty entries into "//" separators, and returns "//" when no
// line is left.
func commentLines(lines []string) []string {
	start, end := 0, len(lines)
	for start < end && lines[start] == "" {
		start++
	}
	for end > start && lines[end-1] == "" {
		end--
	}
	if start == end {
		return []string{"//"}
	}
	res := make([]string, 0, end-start)
	for _, line := range lines[start:end] {
		if line == "" {
			line = "//"
		}
		res = append(res, line)
	}
	return res
}

// sanitizeCommentText prepares text for a Go line comment. It replaces each
// byte sequence that Go source cannot hold with U+FFFD (invalid UTF-8 bytes,
// NUL, and byte order marks) and turns carriage returns into line breaks,
// since the Go scanner strips them from comment text and would otherwise
// merge the surrounding words.
func sanitizeCommentText(text string) string {
	if utf8.ValidString(text) && !strings.ContainsAny(text, "\x00\ufeff\r") {
		return text
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		switch r {
		case 0, '\ufeff':
			r = utf8.RuneError
		case '\r':
			r = '\n'
		}
		b.WriteRune(r)
	}
	return b.String()
}
