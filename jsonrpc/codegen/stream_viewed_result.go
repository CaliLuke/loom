package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// defaultViewExpr is the Go expression of the view that renders a viewed
// result when the stream that sends it has no view to select.
const defaultViewExpr = `"default"`

// addDynamicViewStreamWrapperMethods adds the SetView and SendNotification
// methods of the WebSocket stream wrapper wrapper of ed, a method whose
// viewed result has no view fixed by the design. The wrapper renders the
// results that it sends with the view set with SetView, so it cannot delegate
// its notifications to the connection stream, which renders the default
// view.
func addDynamicViewStreamWrapperMethods(stmt *jen.Statement, wrapper string, ed *httpcodegen.EndpointData) {
	codegen.Doc(stmt, "SetView sets the view used to render the results that the stream sends.")
	stmt.Func().Params(jen.Id("w").Op("*").Id(wrapper)).
		Id("SetView").
		Params(jen.Id("view").String()).
		Block(jen.Id("w").Dot("view").Op("=").Id("view"))
	stmt.Line()
	stmt.Func().Params(jen.Id("w").Op("*").Id(wrapper)).
		Id("SendNotification").
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("res").Add(codegen.TypeRef(ed.Result.Ref))).
		Error().
		BlockFunc(func(g *jen.Group) {
			g.Id("ctx").Op("=").Id("loomtransport.WithStreamObserver").Call(jen.Id("ctx"), jen.Id("w").Dot("observation"))
			writeStreamResultBodyInit(g, "res", "w.view", ed)
			writeStreamMessage(g, jen.Id("w").Dot("stream").Dot("conn"), jen.Id("ctx"),
				jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeNotification").Call(jen.Lit(ed.Method.Name), jen.Id("body")), ed)
		})
}

// writeStreamResultBodyInit declares body, the response body of the result in
// resultVar of ed. A viewed result is rendered with the view that the design
// fixes or else with the view that the Go expression view evaluates to.
func writeStreamResultBodyInit(g *jen.Group, resultVar, view string, ed *httpcodegen.EndpointData) {
	response := ed.Result.Responses[0]
	if code, ok := viewedStreamResultBodyInit(resultVar, view, response, ed); ok {
		g.Add(codegen.Expr(code))
		return
	}
	g.Add(codegen.Expr(responseBodyInit(resultVar, response, response.ServerBody)))
}

// viewedStreamResultBodyInit renders the statements that declare body, the
// response body of the viewed result in resultVar of ed, and reports whether
// ed has a viewed result. response owns the body variants for ed. The statements
// project the result with the view that the design fixes or, when the design
// fixes none, with the view that the Go expression view evaluates to, as the
// HTTP streams do. They then build body with the response body constructor of
// the projected view and return the error of a view that the result type does
// not define.
func viewedStreamResultBodyInit(resultVar, view string, response *httpcodegen.ResponseData, ed *httpcodegen.EndpointData) (string, bool) {
	viewed := ed.Method.ViewedResult
	if viewed == nil || len(response.ServerBody) == 0 {
		return "", false
	}
	bodies := response.ServerBody
	if viewed.ViewName != "" {
		view = strconv.Quote(viewed.ViewName)
	}
	if view == defaultViewExpr {
		// NewViewed returns the requested view, so other body constructors
		// cannot be reached when the stream always selects the default.
		for _, vb := range bodies {
			if vb.View == "default" {
				bodies = []*httpcodegen.TypeData{vb}
				break
			}
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "vres, err := %s.%s(%s, %s)\n", ed.ServicePkgName, viewed.Init.Name, resultVar, view)
	b.WriteString("if err != nil {\n\treturn err\n}\n")
	b.WriteString(responseBodyInit("vres", response, bodies))
	return b.String(), true
}

// responseBodyInit binds the source required by the shared response plan and
// declares body for protocol framing. It does not infer projection from whether
// a conversion constructor exists. View validation remains with the caller.
func responseBodyInit(source string, response *httpcodegen.ResponseData, bodies []*httpcodegen.TypeData) string {
	if !response.EncodePlan.HasBody {
		return "var body any"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "var body any\n{\nres := %s\n", source)
	if len(bodies) == 1 {
		writeResponseBodyAssignment(&b, response, bodies[0])
	} else {
		b.WriteString("switch res.View {\n")
		for _, body := range bodies {
			if body.View == "default" {
				b.WriteString("case \"default\", \"\":\n")
			} else {
				fmt.Fprintf(&b, "case %q:\n", body.View)
			}
			writeResponseBodyAssignment(&b, response, body)
		}
		b.WriteString("}\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// streamResponse returns the response owning a body in the SSE conversion
// family. The family is built from endpoint responses by HTTP analysis.
func streamResponse(body *httpcodegen.TypeData, ed *httpcodegen.EndpointData) *httpcodegen.ResponseData {
	for _, response := range ed.Result.Responses {
		for _, candidate := range response.ServerBody {
			if candidate == body {
				return response
			}
		}
	}
	panic("SSE response body has no owning response plan")
}

// writeStreamMessage writes the converted body and, for a viewed result, the
// view from that same projection in one JSON-RPC envelope.
func writeStreamMessage(g *jen.Group, connection, ctx, constructor jen.Code, ed *httpcodegen.EndpointData) {
	message := constructor
	if ed.Method.ViewedResult != nil {
		g.Id("message").Op(":=").Add(constructor)
		g.Id("message").Dot("View").Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "ResultView").Call(jen.Id("vres").Dot("View"))
		message = jen.Id("message")
	}
	g.Return(jen.Add(connection).Dot("WriteJSON").Call(ctx, message))
}

func writeResponseBodyAssignment(output *strings.Builder, response *httpcodegen.ResponseData, body *httpcodegen.TypeData) {
	if body.Init != nil && body.Init.ServerErrorAware {
		fmt.Fprintf(output, "convertedBody, conversionError := %s\nif conversionError != nil {\nreturn conversionError\n}\nbody = convertedBody\n", response.EncodePlan.BodyValues[body])
		return
	}
	fmt.Fprintf(output, "body = %s\n", response.EncodePlan.BodyValues[body])
}
