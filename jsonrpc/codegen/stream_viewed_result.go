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
			writeStreamResultBodyInit(g, "res", "w.view", ed)
			g.Return(jen.Id("w").Dot("stream").Dot("conn").Dot("WriteJSON").Call(
				jen.Id("ctx"),
				jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeNotification").Call(jen.Lit(ed.Method.Name), jen.Id("body")),
			))
		})
}

// writeStreamResultBodyInit declares body, the response body of the result in
// resultVar of ed. A viewed result is rendered with the view that the design
// fixes or else with the view that the Go expression view evaluates to.
func writeStreamResultBodyInit(g *jen.Group, resultVar, view string, ed *httpcodegen.EndpointData) {
	if ed.Result != nil && len(ed.Result.Responses) > 0 && len(ed.Result.Responses[0].ServerBody) > 0 && ed.Result.Responses[0].ServerBody[0].Init != nil {
		body := ed.Result.Responses[0].ServerBody[0]
		if code, ok := viewedStreamResultBodyInit(resultVar, view, body, ed); ok {
			g.Add(codegen.Expr(code))
			return
		}
		g.Id("body").Op(":=").Id(body.Init.Name).Call(jen.Id(resultVar))
		return
	}
	g.Id("body").Op(":=").Id(resultVar)
}

// viewedStreamResultBodyInit renders the statements that declare body, the
// response body of the viewed result in resultVar of ed, and reports whether
// ed has a viewed result that body renders. body is the response body of ed
// that the result is converted to when it has no views. The statements
// project the result with the view that the design fixes or, when the design
// fixes none, with the view that the Go expression view evaluates to, as the
// HTTP streams do. They then build body with the response body constructor of
// the projected view and return the error of a view that the result type does
// not define.
func viewedStreamResultBodyInit(resultVar, view string, body *httpcodegen.TypeData, ed *httpcodegen.EndpointData) (string, bool) {
	viewed := ed.Method.ViewedResult
	if viewed == nil || body == nil || body.Init == nil {
		return "", false
	}
	bodies := viewedResponseBodies(body, ed)
	if viewed.ViewName != "" {
		view = strconv.Quote(viewed.ViewName)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "vres, err := %s.%s(%s, %s)\n", ed.ServicePkgName, viewed.Init.Name, resultVar, view)
	b.WriteString("if err != nil {\n\treturn err\n}\n")
	if len(bodies) == 1 {
		fmt.Fprintf(&b, "body := %s(vres.Projected)", bodies[0].Init.Name)
		return b.String(), true
	}
	b.WriteString("var body any\nswitch vres.View {\n")
	for _, vb := range bodies {
		if vb.Init == nil {
			continue
		}
		if vb.View == "default" {
			b.WriteString("case \"default\", \"\":\n")
		} else {
			fmt.Fprintf(&b, "case %q:\n", vb.View)
		}
		fmt.Fprintf(&b, "\tbody = %s(vres.Projected)\n", vb.Init.Name)
	}
	b.WriteString("}")
	return b.String(), true
}

// viewedResponseBodies returns the response bodies of the response of ed that
// declares body, one for each view of the result, or body alone when no
// response declares it.
func viewedResponseBodies(body *httpcodegen.TypeData, ed *httpcodegen.EndpointData) []*httpcodegen.TypeData {
	if ed.Result != nil {
		for _, resp := range ed.Result.Responses {
			for _, sb := range resp.ServerBody {
				if sb == body {
					return resp.ServerBody
				}
			}
		}
	}
	return []*httpcodegen.TypeData{body}
}
