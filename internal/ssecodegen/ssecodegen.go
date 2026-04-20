package ssecodegen

import (
	"strings"

	"github.com/dave/jennifer/jen"
)

type (
	// HeaderOptions controls how generated SSE header initialization behaves.
	HeaderOptions struct {
		// PreserveExisting keeps caller-supplied header values when they are set.
		PreserveExisting bool
		// IncludeAccelBuffering emits X-Accel-Buffering: no for proxy buffering control.
		IncludeAccelBuffering bool
	}
)

// InitHeadersSource renders source code that initializes SSE response headers
// for the given writer expression.
func InitHeadersSource(writerExpr string, opts HeaderOptions) string {
	var b strings.Builder
	b.WriteString("s.once.Do(func() {\n")
	b.WriteString("\theader := " + writerExpr + ".Header()\n")
	writeHeaderSourceLine(&b, "Content-Type", "text/event-stream", opts.PreserveExisting)
	writeHeaderSourceLine(&b, "Cache-Control", "no-cache", opts.PreserveExisting)
	writeHeaderSourceLine(&b, "Connection", "keep-alive", opts.PreserveExisting)
	if opts.IncludeAccelBuffering {
		writeHeaderSourceLine(&b, "X-Accel-Buffering", "no", opts.PreserveExisting)
	}
	b.WriteString("\t" + writerExpr + ".WriteHeader(http.StatusOK)\n")
	b.WriteString("})")
	return b.String()
}

// InitHeadersBody returns Jennifer statements that initialize SSE response
// headers for the given writer expression.
func InitHeadersBody(writerExpr string, opts HeaderOptions) []jen.Code {
	return []jen.Code{
		jen.Id("s").Dot("once").Dot("Do").Call(
		jen.Func().Params().BlockFunc(func(g *jen.Group) {
			g.Id("header").Op(":=").Add(renderExpr(writerExpr)).Dot("Header").Call()
			appendHeaderBodyLines(g, opts)
			g.Add(renderExpr(writerExpr)).Dot("WriteHeader").Call(jen.Qual("net/http", "StatusOK"))
		}),
		),
	}
}

// WriteAndFlushSource renders source that writes an SSE event with the given
// call expression and flushes the response controller.
func WriteAndFlushSource(writeCall string, writerExpr string) string {
	var b strings.Builder
	b.WriteString("if err := " + writeCall + "; err != nil {\n\treturn err\n}\n\n")
	b.WriteString("return http.NewResponseController(" + writerExpr + ").Flush()")
	return b.String()
}

// WriteAndFlushBody returns Jennifer statements that write an SSE event using
// the provided call and then flush the response controller for the writer.
func WriteAndFlushBody(writeCall jen.Code, writerExpr string) []jen.Code {
	return []jen.Code{
		jen.If(
			jen.Err().Op(":=").Add(writeCall),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Return(jen.Err()),
		),
		jen.Return(jen.Qual("net/http", "NewResponseController").Call(renderExpr(writerExpr)).Dot("Flush").Call()),
	}
}

func appendHeaderBodyLines(g *jen.Group, opts HeaderOptions) {
	appendHeaderSet(g, "Content-Type", "text/event-stream", opts.PreserveExisting)
	appendHeaderSet(g, "Cache-Control", "no-cache", opts.PreserveExisting)
	appendHeaderSet(g, "Connection", "keep-alive", opts.PreserveExisting)
	if opts.IncludeAccelBuffering {
		appendHeaderSet(g, "X-Accel-Buffering", "no", opts.PreserveExisting)
	}
}

func appendHeaderSet(g *jen.Group, name, value string, preserveExisting bool) {
	if preserveExisting {
		g.If(jen.Id("header").Dot("Get").Call(jen.Lit(name)).Op("==").Lit("")).Block(
			jen.Id("header").Dot("Set").Call(jen.Lit(name), jen.Lit(value)),
		)
		return
	}
	g.Id("header").Dot("Set").Call(jen.Lit(name), jen.Lit(value))
}

func writeHeaderSourceLine(b *strings.Builder, name, value string, preserveExisting bool) {
	if preserveExisting {
		b.WriteString("\tif header.Get(\"" + name + "\") == \"\" {\n")
		b.WriteString("\t\theader.Set(\"" + name + "\", \"" + value + "\")\n")
		b.WriteString("\t}\n")
		return
	}
	b.WriteString("\theader.Set(\"" + name + "\", \"" + value + "\")\n")
}

func renderExpr(expr string) *jen.Statement {
	parts := strings.Split(expr, ".")
	stmt := jen.Id(parts[0])
	for _, part := range parts[1:] {
		stmt = stmt.Dot(part)
	}
	return stmt
}
