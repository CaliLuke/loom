package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func exampleServiceStructSection(data *Data) codegen.Section {
	return codegen.NewJenniferSection("basic-service-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s service example implementation.\nThe example methods log the requests and return zero values.", data.Name))
		stmt.Type().Id(data.VarName + "srvc").Struct()
	})
}

func exampleServiceInitSection(data *Data) codegen.Section {
	return codegen.NewJenniferSection("basic-service-init", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("New%s returns the %s service implementation.", data.StructName, data.Name))
		stmt.Func().Id("New" + data.StructName).Params().Add(codegen.TypeRef(data.PkgName + ".Service")).Block(
			jen.Return(jen.Op("&").Id(data.VarName + "srvc").Values()),
		)
	})
}

func exampleSecurityAuthSection(data *Data) codegen.Section {
	return codegen.NewJenniferSection("security-authfuncs", func(stmt *jen.Statement) {
		for _, scheme := range data.Schemes {
			codegen.Doc(stmt, fmt.Sprintf("%sAuth implements the authorization logic for service %q for the %q security scheme.", scheme.Type, data.Name, scheme.SchemeName))
			authParams := []jen.Code{jen.Id("token").String()}
			switch scheme.Type {
			case "Basic":
				authParams = []jen.Code{jen.Id("user"), jen.Id("pass").String()}
			case "APIKey":
				authParams = []jen.Code{jen.Id("key").String()}
			}
			stmt.Func().
				Params(jen.Id("s").Op("*").Id(data.VarName+"srvc")).
				Id(scheme.Type+"Auth").
				ParamsFunc(func(group *jen.Group) {
					group.Id("ctx").Add(codegen.TypeRef("context.Context"))
					for _, param := range authParams {
						group.Add(param)
					}
					group.Id("scheme").Add(codegen.TypeRef("*security." + scheme.Type + "Scheme"))
				}).
				Params(codegen.TypeRef("context.Context"), jen.Error()).
				BlockFunc(func(group *jen.Group) {
					addExampleCommentBlock(group, `TBD: add authorization logic.

In case of authorization failure this function should return
one of the generated error structs, e.g.:

   return ctx, myservice.MakeUnauthorizedError("invalid token")

Alternatively this function may return an instance of
loom.ServiceError with a Name field value that matches one of
the design error names, e.g:

   return ctx, loom.PermanentError("unauthorized", "invalid token")`)
					group.Return(jen.Id("ctx"), jen.Qual("fmt", "Errorf").Call(jen.Lit("not implemented")))
				})
			stmt.Line()
		}
	})
}

func exampleEndpointSection(data *basicEndpointData) codegen.Section {
	return codegen.NewJenniferSection("basic-endpoint", func(stmt *jen.Statement) {
		codegen.Doc(stmt, data.Description)
		stmt.Func().
			Params(jen.Id("s").Op("*").Id(data.ServiceVarName + "srvc")).
			Id(data.VarName).
			ParamsFunc(func(group *jen.Group) {
				group.Id("ctx").Add(codegen.TypeRef("context.Context"))
				if data.PayloadFullRef != "" {
					group.Id("p").Add(codegen.TypeRef(data.PayloadFullRef))
				}
				if data.ServerStream != nil {
					group.Id("stream").Add(codegen.TypeRef(data.StreamInterface))
					return
				}
				if data.SkipRequestBodyEncodeDecode {
					group.Id("req").Add(codegen.TypeRef("io.ReadCloser"))
				}
			}).
			ParamsFunc(func(group *jen.Group) {
				if data.ServerStream != nil {
					group.Id("err").Error()
					return
				}
				if data.Result != "" {
					group.Id("res").Add(codegen.TypeRef(data.ResultFullRef))
				}
				if data.SkipResponseBodyEncodeDecode {
					group.Id("resp").Add(codegen.TypeRef("io.ReadCloser"))
				}
				if data.ViewedResult != nil && data.ViewedResult.ViewName == "" {
					group.Id("view").String()
				}
				group.Id("err").Error()
			}).
			BlockFunc(func(group *jen.Group) {
				appendExampleRawBlock(group, renderExampleEndpointBody(data))
			})
	})
}

func jsonrpcHandleStreamSection(data *Data) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-handle-stream", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "HandleStream manages a JSON-RPC WebSocket connection, enabling bidirectional communication between the server and client. It receives requests from the client, dispatches them to the appropriate service methods, and can send server-initiated messages back to the client as needed.")
		stmt.Func().
			Params(jen.Id("s").Op("*").Id(data.VarName+"srvc")).
			Id("HandleStream").
			Params(
				jen.Id("ctx").Add(codegen.TypeRef("context.Context")),
				jen.Id("stream").Add(codegen.TypeRef(data.PkgName+".Stream")),
			).
			Error().
			BlockFunc(func(group *jen.Group) {
				group.Add(codegen.Expr(`log.Printf(ctx, "` + data.VarName + `.HandleStream")`))
				group.Line()
				addExampleCommentBlock(group, "Example: In a real implementation you might read from an event source and send notifications via stream.Send(ctx, event). This stub returns when the context is canceled.")
				group.Select().Block(
					jen.Case(jen.Op("<-").Id("ctx").Dot("Done")).Block(
						jen.Return(jen.Id("ctx").Dot("Err").Call()),
					),
					jen.Default().Block(
						jen.Return(jen.Nil()),
					),
				)
			})
	})
}

func renderExampleEndpointBody(data *basicEndpointData) string {
	var body sourceBuilder
	if data.SkipRequestBodyEncodeDecode {
		body.Add("// req is the HTTP request body stream.\n")
		body.Add("defer req.Close()\n")
	}
	if data.Result != "" && data.ResultIsStruct && data.ServerStream == nil {
		body.Add("res = &")
		body.Add(data.ResultFullName)
		body.Add("{}\n")
	}
	if data.SkipResponseBodyEncodeDecode {
		body.Add("// resp is the HTTP response body stream.\n")
		body.Add(`resp = io.NopCloser(strings.NewReader("`)
		body.Add(data.Name)
		body.Add(`"))` + "\n")
	}
	if data.ViewedResult != nil && data.ViewedResult.ViewName == "" {
		if data.ServerStream != nil {
			body.Add("stream.SetView(")
			body.Add(fmt.Sprintf("%q", data.ResultView))
			body.Add(")\n")
		} else {
			body.Add("view = ")
			body.Add(fmt.Sprintf("%q", data.ResultView))
			body.Add("\n")
		}
	}
	body.Add(`log.Printf(ctx, "`)
	body.Add(data.ServiceVarName)
	body.Add(".")
	body.Add(data.Name)
	body.Add("\")\n")
	if data.ServerStream != nil && data.IsJSONRPC && data.ResultFullName != "" {
		body.Add("// Minimal example: emit one progress notification and one final response\n{\n")
		body.Add("notif := ")
		body.Add(exampleStreamValue(data, "progress"))
		body.Add("\nif err := stream.Send(ctx, notif); err != nil {\nreturn err\n}\n")
		body.Add("final := ")
		body.Add(exampleStreamValue(data, "done"))
		body.Add("\nreturn stream.SendAndClose(ctx, final)\n}\n")
	}
	body.Add("return\n")
	return body.String()
}

func exampleStreamValue(data *basicEndpointData, text string) string {
	if data.ResultIsStruct {
		return "&" + data.ResultFullName + "{}"
	}
	if data.ResultFullName == "string" {
		return data.ResultFullName + "(" + fmt.Sprintf("%q", text) + ")"
	}
	return data.ResultFullName + "(0)"
}

func appendExampleRawBlock(group *jen.Group, code string) {
	if trimmed := strings.TrimRight(code, "\n"); strings.TrimSpace(trimmed) != "" {
		group.Add(codegen.Expr(trimmed))
	}
}

func addExampleCommentBlock(group *jen.Group, text string) {
	for _, line := range strings.Split(text, "\n") {
		group.Comment(line)
	}
}
