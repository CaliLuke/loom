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

// ObserveOption configures generated observer emissions for SSE write and
// flush failures. When CtxExpr is set to an in-scope Go expression that
// yields a context.Context (typically "ctx"), the generated code emits
// stream-failure events via loomtransport.Observe; leaving it empty keeps
// the legacy bare write+flush code shape.
type ObserveOption struct {
	// CtxExpr is the in-scope expression that yields the request context.
	CtxExpr string
	// Transport is the loomtransport.TransportKind constant identifier
	// (e.g. "loomtransport.TransportHTTP") that classifies emitted events.
	// When empty, "loomtransport.TransportHTTP" is used.
	Transport string
}

// WriteAndFlushSource renders source that writes an SSE event with the given
// call expression and flushes the response controller. When opts.CtxExpr is
// non-empty the generated code also reports stream write and flush failures
// through loomtransport.Observe before returning the error.
func WriteAndFlushSource(writeCall string, writerExpr string, opts ...ObserveOption) string {
	ctxExpr, transport := observeArgs(opts)
	var b strings.Builder
	b.WriteString("if err := " + writeCall + "; err != nil {\n")
	if ctxExpr != "" {
		b.WriteString("\tloomtransport.Observe(" + ctxExpr + ", loomtransport.Event{Kind: loomtransport.EventKindStreamFailure, Reason: loomtransport.ReasonStreamWriteFailed, Transport: " + transport + "})\n")
	}
	b.WriteString("\treturn err\n}\n\n")
	if ctxExpr == "" {
		b.WriteString("return http.NewResponseController(" + writerExpr + ").Flush()")
		return b.String()
	}
	b.WriteString("if err := http.NewResponseController(" + writerExpr + ").Flush(); err != nil {\n")
	b.WriteString("\tloomtransport.Observe(" + ctxExpr + ", loomtransport.Event{Kind: loomtransport.EventKindStreamFailure, Reason: loomtransport.ReasonStreamFlushFailed, Transport: " + transport + "})\n")
	b.WriteString("\treturn err\n}\n")
	b.WriteString("return nil")
	return b.String()
}

// WriteAndFlushBody returns Jennifer statements that write an SSE event using
// the provided call and then flush the response controller for the writer.
//
// When ObserveOption.CtxExpr is provided, the emitted observer references use
// the `loomtransport` alias so the generated code shares the import
// registered in the surrounding server file header. We deliberately avoid
// jen.Qual on observability/transport here to keep
// TestGeneratorFilesDoNotMixNamedLoomImportsWithJenQual passing.
func WriteAndFlushBody(writeCall jen.Code, writerExpr string, opts ...ObserveOption) []jen.Code {
	ctxExpr, transport := observeArgs(opts)
	emit := func(reason string) jen.Code {
		return jen.Id("loomtransport.Observe").Call(
			renderExpr(ctxExpr),
			jen.Id("loomtransport.Event").Values(jen.Dict{
				jen.Id("Kind"):      jen.Id("loomtransport.EventKindStreamFailure"),
				jen.Id("Reason"):    jen.Id("loomtransport." + reason),
				jen.Id("Transport"): jen.Id(transport),
			}),
		)
	}
	emitWrite := func() jen.Code { return emit("ReasonStreamWriteFailed") }
	emitFlush := func() jen.Code { return emit("ReasonStreamFlushFailed") }
	writeFail := []jen.Code{jen.Return(jen.Err())}
	if ctxExpr != "" {
		writeFail = []jen.Code{emitWrite(), jen.Return(jen.Err())}
	}
	stmts := []jen.Code{
		jen.If(
			jen.Err().Op(":=").Add(writeCall),
			jen.Err().Op("!=").Nil(),
		).Block(writeFail...),
	}
	if ctxExpr == "" {
		stmts = append(stmts, jen.Return(jen.Qual("net/http", "NewResponseController").Call(renderExpr(writerExpr)).Dot("Flush").Call()))
		return stmts
	}
	flushFail := []jen.Code{emitFlush(), jen.Return(jen.Err())}
	stmts = append(stmts,
		jen.If(
			jen.Err().Op(":=").Qual("net/http", "NewResponseController").Call(renderExpr(writerExpr)).Dot("Flush").Call(),
			jen.Err().Op("!=").Nil(),
		).Block(flushFail...),
		jen.Return(jen.Nil()),
	)
	return stmts
}

func observeArgs(opts []ObserveOption) (string, string) {
	if len(opts) == 0 {
		return "", ""
	}
	ctxExpr := opts[0].CtxExpr
	transport := opts[0].Transport
	if ctxExpr != "" && transport == "" {
		transport = "loomtransport.TransportHTTP"
	}
	return ctxExpr, transport
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
