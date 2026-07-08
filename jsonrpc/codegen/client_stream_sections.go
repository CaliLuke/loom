package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcSSEClientStreamSection(ed *httpcodegen.EndpointData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-sse-client-stream", func(stmt *jen.Statement) {
		writeJSONRPCSSEClientStreamType(stmt, ed)
		stmt.Line()
		writeJSONRPCSSERecv(stmt, ed)
		if ed.Method.Result != "" {
			stmt.Line()
			writeJSONRPCSSEDecodeResult(stmt, ed)
		}
		stmt.Line()
		writeJSONRPCSSEClose(stmt, ed)
	})
}

func writeJSONRPCSSEClientStreamType(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	codegen.Doc(stmt, fmt.Sprintf("%sClientStream implements the %s.%sClientStream interface using Server-Sent Events.", ed.Method.VarName, ed.ServicePkgName, ed.Method.VarName))
	stmt.Type().Id(ed.Method.VarName+"ClientStream").Struct(
		jen.Id("resp").Op("*").Qual("net/http", "Response"),
		jen.Id("reader").Op("*").Qual("bufio", "Reader"),
		jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")),
		jen.Id("closed").Bool(),
		jen.Id("lock").Qual("sync", "Mutex"),
		jen.Id("readLock").Qual("sync", "Mutex"),
	)
	stmt.Line()
	writeJSONRPCSSEReadEvent(stmt, ed)
	stmt.Line()
	writeJSONRPCSSEReadLine(stmt, ed)
	stmt.Line()
	writeJSONRPCSSEMarkClosed(stmt, ed)
}

func writeJSONRPCSSEReadEvent(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName+"ClientStream")).
		Id("readSSEEvent").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(jen.Index().Byte(), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Var().Id("event").Qual("bytes", "Buffer")
			g.Line()
			g.Id("s").Dot("lock").Dot("Lock").Call()
			g.If(jen.Id("s").Dot("closed")).Block(
				jen.Id("s").Dot("lock").Dot("Unlock").Call(),
				jen.Return(jen.Nil(), jen.Qual("io", "EOF")),
			)
			g.Id("reader").Op(":=").Id("s").Dot("reader")
			g.Id("s").Dot("lock").Dot("Unlock").Call()
			g.Line()
			g.For().BlockFunc(func(fg *jen.Group) {
				fg.List(jen.Id("line"), jen.Err()).Op(":=").Id("s").Dot("readSSELine").Call(jen.Id("ctx"), jen.Id("reader"))
				fg.If(jen.Err().Op("!=").Nil()).Block(
					jen.If(jen.Err().Op("==").Qual("io", "EOF").Op("&&").Id("event").Dot("Len").Call().Op(">").Lit(0)).Block(
						jen.Return(jen.Id("event").Dot("Bytes").Call(), jen.Nil()),
					),
					jen.Return(jen.Nil(), jen.Err()),
				)
				fg.Line()
				fg.Id("event").Dot("WriteString").Call(jen.Id("line"))
				fg.Line()
				fg.Id("line").Op("=").Qual("strings", "TrimRight").Call(jen.Id("line"), jen.Lit("\r\n"))
				fg.If(jen.Id("line").Op("==").Lit("")).Block(
					jen.If(jen.Id("event").Dot("Len").Call().Op(">").Lit(0)).Block(
						jen.Return(jen.Id("event").Dot("Bytes").Call(), jen.Nil()),
					),
					jen.Continue(),
				)
			})
		})
}

func writeJSONRPCSSEReadLine(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName+"ClientStream")).
		Id("readSSELine").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("reader").Op("*").Qual("bufio", "Reader"),
		).
		Params(jen.String(), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Type().Id("readLineResult").Struct(
				jen.Id("line").String(),
				jen.Id("err").Error(),
			)
			g.Line()
			g.If(jen.Err().Op(":=").Id("ctx").Dot("Err").Call(), jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Lit(""), jen.Err()),
			)
			g.Line()
			g.Id("readc").Op(":=").Make(jen.Chan().Id("readLineResult"), jen.Lit(1))
			g.Go().Func().Params().Block(
				jen.List(jen.Id("line"), jen.Err()).Op(":=").Id("reader").Dot("ReadString").Call(jen.LitByte('\n')),
				jen.Id("readc").Op("<-").Id("readLineResult").Values(jen.Dict{
					jen.Id("line"): jen.Id("line"),
					jen.Id("err"):  jen.Err(),
				}),
			).Call()
			g.Line()
			g.Var().Id("result").Id("readLineResult")
			g.Select().Block(
				jen.Case(jen.Id("result").Op("=").Op("<-").Id("readc")).Block(),
				jen.Case(jen.Op("<-").Id("ctx").Dot("Done").Call()).Block(
					jen.Select().Block(
						jen.Case(jen.Id("result").Op("=").Op("<-").Id("readc")).Block(),
						jen.Default().Block(
							jen.Id("_").Op("=").Id("s").Dot("Close").Call(),
							jen.Return(jen.Lit(""), jen.Id("ctx").Dot("Err").Call()),
						),
					),
				),
			)
			g.Return(jen.Id("result").Dot("line"), jen.Id("result").Dot("err"))
		})
}

func writeJSONRPCSSEMarkClosed(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName+"ClientStream")).
		Id("markClosed").
		Params().
		Block(
			jen.Id("s").Dot("lock").Dot("Lock").Call(),
			jen.Id("s").Dot("closed").Op("=").True(),
			jen.Id("s").Dot("lock").Dot("Unlock").Call(),
		)
}

func writeJSONRPCSSERecv(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	codegen.Doc(stmt, ed.Method.ClientStream.RecvDesc)
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName+"ClientStream")).
		Id(ed.Method.ClientStream.RecvName).
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(codegen.TypeRef(ed.Result.Ref), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Id("s").Dot("readLock").Dot("Lock").Call()
			g.Defer().Id("s").Dot("readLock").Dot("Unlock").Call()
			g.Line()
			g.Var().Id("zero").Add(codegen.TypeRef(ed.Result.Ref))
			g.Line()
			g.Id("s").Dot("lock").Dot("Lock").Call()
			g.If(jen.Id("s").Dot("closed")).Block(
				jen.Id("s").Dot("lock").Dot("Unlock").Call(),
				jen.Return(jen.Id("zero"), jen.Qual("io", "EOF")),
			)
			g.Id("s").Dot("lock").Dot("Unlock").Call()
			g.Line()
			g.For().Block(
				jen.List(jen.Id("rawEvent"), jen.Err()).Op(":=").Id("s").Dot("readSSEEvent").Call(jen.Id("ctx")),
				jen.If(jen.Err().Op("!=").Nil()).Block(
					jen.Id("s").Dot("markClosed").Call(),
					jen.Return(jen.Id("zero"), jen.Err()),
				),
				jen.Line(),
				jen.List(jen.Id("parsedEvent"), jen.Err()).Op(":=").Add(codegen.Expr("loomhttp.ParseSSEEvent")).Call(jen.Id("rawEvent")),
				jen.If(jen.Err().Op("!=").Nil()).Block(
					jen.Id("s").Dot("markClosed").Call(),
					jen.Return(jen.Id("zero"), jen.Err()),
				),
				jen.Line(),
				jen.List(jen.Id("eventType"), jen.Id("data")).Op(":=").List(
					jen.Id("parsedEvent").Dot("Type"),
					jen.Index().Byte().Call(jen.Id("parsedEvent").Dot("Data")),
				),
				jen.Line(),
				jen.Switch(jen.Id("eventType")).BlockFunc(func(sg *jen.Group) {
					sg.Case(jen.Lit("notification")).BlockFunc(func(cg *jen.Group) {
						writeSSEClientNotificationCase(cg, ed)
					})
					sg.Case(jen.Lit("response")).BlockFunc(func(cg *jen.Group) {
						writeSSEClientResponseCase(cg, ed, false)
					})
					sg.Case(jen.Lit("error")).BlockFunc(func(cg *jen.Group) {
						writeSSEClientErrorCase(cg)
					})
					sg.Case(jen.Lit(""), jen.Lit("message")).BlockFunc(func(cg *jen.Group) {
						writeSSEClientMessageCase(cg, ed)
					})
					sg.Default().Block(
						jen.Continue(),
					)
				}),
			)
		})
}

func writeSSEClientNotificationCase(g *jen.Group, ed *httpcodegen.EndpointData) {
	g.Var().Id("notification").Struct(
		jen.Id("JSONRPC").String().Tag(map[string]string{"json": "jsonrpc"}),
		jen.Id("Method").String().Tag(map[string]string{"json": "method"}),
		jen.Id("Params").Qual("encoding/json", "RawMessage").Tag(map[string]string{"json": "params"}),
	)
	g.If(
		jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("notification")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse notification: %w"), jen.Err())),
	)
	g.If(jen.Id("notification").Dot("JSONRPC").Op("!=").Lit("2.0")).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("invalid JSON-RPC version: %s"), jen.Id("notification").Dot("JSONRPC"))),
	)
	g.If(jen.Id("notification").Dot("Method").Op("!=").Lit(ed.Method.Name)).Block(
		jen.Continue(),
	)
	if ed.Method.Result != "" {
		g.List(jen.Id("result"), jen.Err()).Op(":=").Id("s").Dot("decodeResult").Call(jen.Id("notification").Dot("Params"))
		g.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to decode result: %w"), jen.Err())),
		)
		g.Return(jen.Id("result"), jen.Nil())
		return
	}
	g.Return(jen.Id("zero"), jen.Nil())
}

func writeSSEClientResponseCase(g *jen.Group, ed *httpcodegen.EndpointData, closeOnSuccess bool) {
	g.Var().Id("response").Add(codegen.TypeRef("jsonrpc.Response"))
	g.If(
		jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("response")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse response: %w"), jen.Err())),
	)
	g.If(jen.Id("response").Dot("Error").Op("!=").Nil()).Block(
		func() jen.Code {
			if closeOnSuccess {
				return jen.Block(
					jen.Id("s").Dot("markClosed").Call(),
					jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("JSON-RPC error %d: %s"), jen.Id("response").Dot("Error").Dot("Code"), jen.Id("response").Dot("Error").Dot("Message"))),
				)
			}
			return jen.Block(
				jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("JSON-RPC error %d: %s"), jen.Id("response").Dot("Error").Dot("Code"), jen.Id("response").Dot("Error").Dot("Message"))),
			)
		}(),
	)
	if ed.Method.Result != "" {
		g.If(jen.Id("response").Dot("Result").Op("==").Nil()).Block(
			jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("missing result in response"))),
		)
		g.List(jen.Id("resultBytes"), jen.Err()).Op(":=").Qual("encoding/json", "Marshal").Call(jen.Id("response").Dot("Result"))
		g.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to marshal result: %w"), jen.Err())),
		)
		g.List(jen.Id("result"), jen.Err()).Op(":=").Id("s").Dot("decodeResult").Call(jen.Qual("encoding/json", "RawMessage").Call(jen.Id("resultBytes")))
		g.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to decode final result: %w"), jen.Err())),
		)
		if closeOnSuccess {
			g.Id("s").Dot("markClosed").Call()
		}
		g.Return(jen.Id("result"), jen.Nil())
		return
	}
	if closeOnSuccess {
		g.Id("s").Dot("markClosed").Call()
	}
	g.Return(jen.Id("zero"), jen.Nil())
}

func writeSSEClientErrorCase(g *jen.Group) {
	g.Var().Id("response").Add(codegen.TypeRef("jsonrpc.Response"))
	g.If(
		jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("response")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse error response: %w"), jen.Err())),
	)
	g.Id("s").Dot("markClosed").Call()
	g.If(jen.Id("response").Dot("Error").Op("!=").Nil()).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("JSON-RPC error %d: %s"), jen.Id("response").Dot("Error").Dot("Code"), jen.Id("response").Dot("Error").Dot("Message"))),
	)
	g.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("unexpected error response")))
}

func writeSSEClientMessageCase(g *jen.Group, ed *httpcodegen.EndpointData) {
	g.Var().Id("envelope").Map(jen.String()).Qual("encoding/json", "RawMessage")
	g.If(
		jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("envelope")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse message event: %w"), jen.Err())),
	)
	g.If(
		jen.List(jen.Id("_"), jen.Id("ok")).Op(":=").Id("envelope").Index(jen.Lit("method")),
		jen.Id("ok"),
	).BlockFunc(func(cg *jen.Group) {
		writeSSEClientNotificationCase(cg, ed)
	})
	writeSSEClientResponseCase(g, ed, true)
}

func writeJSONRPCSSEDecodeResult(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName+"ClientStream")).
		Id("decodeResult").
		Params(jen.Id("data").Qual("encoding/json", "RawMessage")).
		Params(codegen.TypeRef(ed.Result.Ref), jen.Error()).
		Block(
			jen.Id("resp").Op(":=").Op("&").Qual("net/http", "Response").Values(jen.Dict{
				jen.Id("StatusCode"): jen.Qual("net/http", "StatusOK"),
				jen.Id("Body"):       jen.Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewReader").Call(jen.Id("data"))),
			}),
			jen.Id("decoder").Op(":=").Id("s").Dot("decoder").Call(jen.Id("resp")),
			jen.Var().Id("result").Add(codegen.TypeRef(ed.Result.Ref)),
			jen.If(
				jen.Err().Op(":=").Id("decoder").Dot("Decode").Call(jen.Op("&").Id("result")),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Return(jen.Id("result"), jen.Err()),
			),
			jen.Return(jen.Id("result"), jen.Nil()),
		)
}

func writeJSONRPCSSEClose(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	codegen.Doc(stmt, "Close closes the stream.")
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName + "ClientStream")).
		Id("Close").
		Params().
		Error().
		BlockFunc(func(g *jen.Group) {
			g.Var().Id("body").Qual("io", "Closer")
			g.Line()
			g.Id("s").Dot("lock").Dot("Lock").Call()
			g.If(jen.Id("s").Dot("closed")).Block(
				jen.Id("s").Dot("lock").Dot("Unlock").Call(),
				jen.Return(jen.Nil()),
			)
			g.Id("s").Dot("closed").Op("=").True()
			g.If(jen.Id("s").Dot("resp").Op("!=").Nil()).Block(
				jen.Id("body").Op("=").Id("s").Dot("resp").Dot("Body"),
			)
			g.Id("s").Dot("lock").Dot("Unlock").Call()
			g.Line()
			g.If(jen.Id("body").Op("!=").Nil()).Block(
				jen.Return(jen.Id("body").Dot("Close").Call()),
			)
			g.Return(jen.Nil())
		})
}
