//nolint:errcheck // Generator helpers write only to in-memory builders.
package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func addStreamWrappersSection(stmt *jen.Statement, streams []*StreamInterceptorData, server bool) {
	for _, stream := range streams {
		stmt.Line()
		addStreamUnwrapSection(stmt, stream, server)
		addStreamSendSection(stmt, stream)
		addStreamRecvSection(stmt, stream)
		if stream.MustClose {
			stmt.Line()
			codegen.Doc(stmt, "Close closes the stream.")
			stmt.Func().
				Params(jen.Id("w").Op("*").Id("wrapped" + stream.Interface)).
				Id("Close").
				Params().
				Error().
				Block(
					jen.Return(jen.Id("w").Dot("stream").Dot("Close").Call()),
				)
		}
		stmt.Line()
	}
}

func addStreamUnwrapSection(stmt *jen.Statement, stream *StreamInterceptorData, server bool) {
	if !server && stream.SendTypeRef == "" {
		return
	}
	codegen.Doc(stmt, "Unwrap returns the underlying stream type.")
	stmt.Func().
		Params(jen.Id("w").Op("*").Id("wrapped" + stream.Interface)).
		Id("Unwrap").
		Params().
		Any().
		Block(
			jen.Return(jen.Id("w").Dot("stream")),
		)
}

func addStreamSendSection(stmt *jen.Statement, stream *StreamInterceptorData) {
	if stream.SendTypeRef == "" {
		return
	}
	stmt.Line()
	codegen.Doc(stmt, fmt.Sprintf("%s streams instances of %q after executing the applied interceptor.", stream.SendName, stream.Interface))
	stmt.Func().
		Params(jen.Id("w").Op("*").Id("wrapped" + stream.Interface)).
		Id(stream.SendName).
		Params(jen.Id("v").Add(codegen.TypeRef(stream.SendTypeRef))).
		Error().
		Block(
			jen.Return(jen.Id("w").Dot(stream.SendWithContextName).Call(jen.Id("w").Dot("ctx"), jen.Id("v"))),
		)
	stmt.Line()
	codegen.Doc(stmt, fmt.Sprintf("%s streams instances of %q after executing the applied interceptor with context.", stream.SendWithContextName, stream.Interface))
	stmt.Func().
		Params(jen.Id("w").Op("*").Id("wrapped"+stream.Interface)).
		Id(stream.SendWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Add(codegen.TypeRef(stream.SendTypeRef))).
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawGroup(group, "if w.sendWithContext == nil {\n\treturn w.stream."+stream.SendWithContextName+"(ctx, v)\n}\nreturn w.sendWithContext(ctx, v)")
		})
}

func addStreamRecvSection(stmt *jen.Statement, stream *StreamInterceptorData) {
	if stream.RecvTypeRef == "" {
		return
	}
	stmt.Line()
	codegen.Doc(stmt, fmt.Sprintf("%s reads instances of %q from the stream after executing the applied interceptor.", stream.RecvName, stream.Interface))
	stmt.Func().
		Params(jen.Id("w").Op("*").Id("wrapped"+stream.Interface)).
		Id(stream.RecvName).
		Params().
		Params(codegen.TypeRef(stream.RecvTypeRef), jen.Error()).
		Block(
			jen.Return(jen.Id("w").Dot(stream.RecvWithContextName).Call(jen.Id("w").Dot("ctx"))),
		)
	stmt.Line()
	codegen.Doc(stmt, fmt.Sprintf("%s reads instances of %q from the stream after executing the applied interceptor with context.", stream.RecvWithContextName, stream.Interface))
	stmt.Func().
		Params(jen.Id("w").Op("*").Id("wrapped"+stream.Interface)).
		Id(stream.RecvWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(codegen.TypeRef(stream.RecvTypeRef), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			addRawGroup(group, "if w.recvWithContext == nil {\n\treturn w.stream."+stream.RecvWithContextName+"(ctx)\n}\nreturn w.recvWithContext(ctx)")
		})
}

func addRawGroup(group *jen.Group, code string) {
	if strings.TrimSpace(code) == "" {
		return
	}
	if strings.HasPrefix(code, "\n") {
		group.Line()
	}
	group.Add(codegen.Expr(strings.TrimRight(code, "\n")))
}
