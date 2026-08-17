package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func exampleServiceStructSection(data *Data) codegen.Section {
	return codegen.NewJenniferSection("basic-service-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s service starter implementation.\nThe starter methods fail until the application implements them.", data.Name))
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
		for _, scheme := range data.Schemes.DedupeByType() {
			codegen.Doc(stmt, fmt.Sprintf("%sAuth implements the authorization logic for service %q for the %s security scheme type.", scheme.Type, data.Name, scheme.Type))
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
				if data.FileResponse {
					group.Id("file").Add(codegen.TypeRef("*loomhttp.FileResponse"))
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
				group.Return(codegen.Expr("loom.Fault").Call(jen.Lit(data.VarName + ".HandleStream is not implemented")))
			})
	})
}

func renderExampleEndpointBody(data *basicEndpointData) string {
	var body sourceBuilder
	if data.SkipRequestBodyEncodeDecode {
		body.Add("// req is the HTTP request body stream.\n")
		body.Add("defer func() {\nerr = errors.Join(err, req.Close())\n}()\n")
	}
	body.Add(`log.Printf(ctx, "`)
	body.Add(data.ServiceVarName)
	body.Add(".")
	body.Add(data.Name)
	body.Add("\")\n")
	body.Add(fmt.Sprintf("err = loom.Fault(%q)\n", data.ServiceVarName+"."+data.VarName+" is not implemented"))
	body.Add("return\n")
	return body.String()
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
