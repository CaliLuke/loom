package codegen

import (
	"github.com/dave/jennifer/jen"

	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// loomtransportRef renders an in-scope reference to a symbol from the
// `loomtransport` package alias. The observability/transport import is
// registered with that alias in the JSON-RPC server file header (see
// jsonrpcServerImports), so generated references must use the same alias
// instead of jen.Qual to satisfy the LoomNamedImport-versus-jen.Qual
// safety check enforced by
// TestGeneratorFilesDoNotMixNamedLoomImportsWithJenQual.
func loomtransportRef(symbol string) *jen.Statement {
	return jen.Id("loomtransport." + symbol)
}

func addJSONRPCHandleSingleSection(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	stmt.Comment("handleSingle handles a single JSON-RPC request.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("handleSingle").
		Params(
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
		).
		Block(
			jen.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
			jen.If(
				jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("req")),
				jen.Err().Op("!=").Nil(),
			).BlockFunc(func(g *jen.Group) {
				g.Add(loomtransportRef("RequestObserverFromContext")).Call(jen.Id("r").Dot("Context").Call()).Dot("Fail").Call(loomtransportRef("ReasonInvalidJSONRPCEnvelope"))
				writeParseErrorResponse(g, jen.Id("r").Dot("Context").Call())
				g.Return()
			}),
			jen.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
		)
	stmt.Line()
}

func addJSONRPCHandleBatchSection(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	stmt.Comment("handleBatch handles a batch of JSON-RPC requests.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("handleBatch").
		Params(
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
		).
		BlockFunc(func(g *jen.Group) {
			writeJSONRPCBatchDecode(g)
			writeJSONRPCEmptyBatchResponse(g)
			writeJSONRPCBatchWriter(g)
		})
	stmt.Line()
}

func writeJSONRPCBatchDecode(g *jen.Group) {
	g.Var().Id("rawReqs").Index().Qual("encoding/json", "RawMessage")
	g.If(
		jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("rawReqs")),
		jen.Err().Op("!=").Nil(),
	).BlockFunc(func(g *jen.Group) {
		g.Add(loomtransportRef("RequestObserverFromContext")).Call(jen.Id("r").Dot("Context").Call()).Dot("Fail").Call(loomtransportRef("ReasonInvalidJSONRPCBatch"))
		writeParseErrorResponse(g, jen.Id("r").Dot("Context").Call())
		g.Return()
	})
}

func writeJSONRPCEmptyBatchResponse(g *jen.Group) {
	g.If(jen.Len(jen.Id("rawReqs")).Op("==").Lit(0)).Block(
		loomtransportRef("RequestObserverFromContext").Call(jen.Id("r").Dot("Context").Call()).Dot("Fail").Call(loomtransportRef("ReasonInvalidJSONRPCBatch")),
		jen.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(
			jen.Nil(),
			jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"),
			jen.Lit("Invalid request"),
			jen.Nil(),
		),
		jen.If(
			jen.Id("encErr").Op(":=").Id("s").Dot("encoder").Call(jen.Id("r").Dot("Context").Call(), jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
			jen.Id("encErr").Op("!=").Nil(),
		).Block(
			jen.Id("s").Dot("errhandler").Call(
				jen.Id("r").Dot("Context").Call(),
				jen.Id("w"),
				jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode invalid batch response: %w"), jen.Id("encErr")),
			),
		),
		jen.Return(),
	)
}

func writeJSONRPCBatchWriter(g *jen.Group) {
	g.Add(loomtransportRef("RequestObserverFromContext")).Call(jen.Id("r").Dot("Context").Call()).Dot("SetJSONRPC").Call(
		jen.Lit(""),
		jen.Lit(""),
		jen.Len(jen.Id("rawReqs")),
		jen.False(),
	)
	g.Id("w").Dot("Header").Call().Dot("Set").Call(jen.Lit("Content-Type"), jen.Lit("application/json"))
	g.Id("writer").Op(":=").Op("&").Id("batchWriter").Values(jen.Dict{
		jen.Id("Writer"): jen.Id("w"),
	})
	g.For(
		jen.List(jen.Id("_"), jen.Id("rawReq")).Op(":=").Range().Id("rawReqs"),
	).BlockFunc(writeJSONRPCBatchItem)
	g.If(jen.Id("writer").Dot("written")).Block(
		jen.Id("writer").Dot("Writer").Dot("Write").Call(jen.Index().Byte().Values(jen.LitByte(']'))),
	)
}

func writeJSONRPCBatchItem(g *jen.Group) {
	g.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest")
	g.If(
		jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("rawReq"), jen.Op("&").Id("req")),
		jen.Err().Op("!=").Nil(),
	).Block(
		loomtransportRef("RequestObserverFromContext").Call(jen.Id("r").Dot("Context").Call()).Dot("Fail").Call(loomtransportRef("ReasonInvalidJSONRPCEnvelope")),
		jen.Id("s").Dot("encodeJSONRPCError").Call(
			jen.Id("r").Dot("Context").Call(),
			jen.Id("writer"),
			jen.Op("&").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest").Values(),
			jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"),
			jen.Lit("Invalid request"),
			jen.Nil(),
		),
		jen.Continue(),
	)
	g.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("writer"))
}

func addJSONRPCProcessRequestSection(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	stmt.Comment("processRequest processes a single JSON-RPC request.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("processRequest").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
			jen.Id("req").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
			jen.Id("w").Qual("net/http", "ResponseWriter"),
		).
		BlockFunc(func(g *jen.Group) {
			writeJSONRPCProcessRequestBody(g, data.Endpoints)
		})
	stmt.Line()
}

func writeJSONRPCProcessRequestBody(g *jen.Group, endpoints []*httpcodegen.EndpointData) {
	// Record JSON-RPC envelope fields on the request observer as soon as the
	// envelope has been decoded. Pre-decode rejection events emitted earlier
	// already left these fields empty, satisfying the plan's contract that
	// JSON-RPC fields are present only after a successful decode.
	g.Add(loomtransportRef("RequestObserverFromContext")).Call(jen.Id("ctx")).Dot("SetJSONRPC").Call(
		jen.Id("req").Dot("Method"),
		jen.Qual("github.com/CaliLuke/loom/jsonrpc", "IDToString").Call(jen.Id("req").Dot("ID")),
		jen.Lit(0),
		jen.Op("!").Id("req").Dot("HasID"),
	)
	writeJSONRPCInvalidRequestCheck(g, jen.Id("req").Dot("Invalid"), jen.Lit("Invalid request"), "ReasonInvalidJSONRPCEnvelope")
	writeJSONRPCInvalidRequestCheck(g, jen.Id("req").Dot("JSONRPC").Op("!=").Lit("2.0"), jen.Lit("Invalid request"), "ReasonInvalidJSONRPCEnvelope")
	writeJSONRPCInvalidRequestCheck(g, jen.Id("req").Dot("Method").Op("==").Lit(""), jen.Lit("Missing method field"), "ReasonInvalidJSONRPCMethod")
	g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(sg *jen.Group) {
		writeJSONRPCMethodDispatch(sg, endpoints)
		sg.Default().Block(
			loomtransportRef("RequestObserverFromContext").Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef("ReasonUnsupportedMethod")),
			jen.Id("s").Dot("encodeJSONRPCError").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Id("req"),
				jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"),
				jen.Lit("Method not found"),
				jen.Nil(),
			),
		)
	})
}

func writeJSONRPCInvalidRequestCheck(g *jen.Group, condition jen.Code, message jen.Code, reason string) {
	g.If(condition).Block(
		loomtransportRef("RequestObserverFromContext").Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef(reason)),
		jen.Id("s").Dot("encodeJSONRPCError").Call(
			jen.Id("ctx"),
			jen.Id("w"),
			jen.Id("req"),
			jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"),
			message,
			jen.Nil(),
		),
		jen.Return(),
	)
}

func writeParseErrorResponse(g *jen.Group, ctx jen.Code) {
	g.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(
		jen.Nil(),
		jen.Qual("github.com/CaliLuke/loom/jsonrpc", "ParseError"),
		jen.Lit("Parse error"),
		jen.Nil(),
	)
	g.If(
		jen.Id("encErr").Op(":=").Id("s").Dot("encoder").Call(ctx, jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
		jen.Id("encErr").Op("!=").Nil(),
	).Block(
		jen.Id("s").Dot("errhandler").Call(
			ctx,
			jen.Id("w"),
			jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode parse error response: %w"), jen.Id("encErr")),
		),
	)
}

func writeJSONRPCMethodDispatch(g *jen.Group, endpoints []*httpcodegen.EndpointData) {
	for _, endpoint := range endpoints {
		g.Case(jen.Lit(endpoint.Method.Name)).Block(
			jen.If(
				jen.Err().Op(":=").Id("s").Dot(endpoint.Method.VarName).Call(jen.Id("ctx"), jen.Id("r"), jen.Id("req"), jen.Id("w")),
				jen.Err().Op("!=").Nil(),
			).Block(
				loomtransportRef("RequestObserverFromContext").Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef("ReasonHandlerError")),
				jen.Id("s").Dot("errhandler").Call(
					jen.Id("ctx"),
					jen.Id("w"),
					jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for "+endpoint.Method.Name+": %w"), jen.Err()),
				),
			),
		)
	}
}
