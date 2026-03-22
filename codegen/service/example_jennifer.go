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
			argName := "token"
			switch scheme.Type {
			case "Basic":
				argName = "user, pass"
			case "APIKey":
				argName = "key"
			}
			stmt.Add(codegen.Expr(fmt.Sprintf(`func (s *%ssrvc) %sAuth(ctx context.Context, %s string, scheme *security.%sScheme) (context.Context, error) {
//
// TBD: add authorization logic.
//
// In case of authorization failure this function should return
// one of the generated error structs, e.g.:
//
//    return ctx, myservice.MakeUnauthorizedError("invalid token")
//
// Alternatively this function may return an instance of
// loom.ServiceError with a Name field value that matches one of
// the design error names, e.g:
//
//    return ctx, loom.PermanentError("unauthorized", "invalid token")
//
return ctx, fmt.Errorf("not implemented")
}`, data.VarName, scheme.Type, argName, scheme.Type)))
			stmt.Line()
		}
	})
}

func exampleEndpointSection(data *basicEndpointData) codegen.Section {
	return codegen.NewJenniferSection("basic-endpoint", func(stmt *jen.Statement) {
		codegen.Doc(stmt, data.Description)
		stmt.Add(codegen.Expr(renderExampleEndpoint(data)))
	})
}

func jsonrpcHandleStreamSection(data *Data) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-handle-stream", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(fmt.Sprintf(`// HandleStream manages a JSON-RPC WebSocket connection, enabling bidirectional
// communication between the server and client. It receives requests from the
// client, dispatches them to the appropriate service methods, and can send
// server-initiated messages back to the client as needed.
func (s *%ssrvc) HandleStream(ctx context.Context, stream %s.Stream) error {
log.Printf(ctx, %q)

// Example: In a real implementation you might read from an event source
// and send notifications via stream.Send(ctx, event). This stub returns
// when the context is canceled.
select {
case <-ctx.Done():
	return ctx.Err()
default:
	return nil
}
}`, data.VarName, data.PkgName, data.VarName+".HandleStream")))
	})
}

func renderExampleEndpoint(data *basicEndpointData) string {
	var signature strings.Builder
	signature.WriteString("func (s *")
	signature.WriteString(data.ServiceVarName)
	signature.WriteString("srvc) ")
	signature.WriteString(data.VarName)
	signature.WriteString("(ctx context.Context")
	if data.PayloadFullRef != "" {
		signature.WriteString(", p ")
		signature.WriteString(data.PayloadFullRef)
	}
	if data.ServerStream != nil {
		signature.WriteString(", stream ")
		signature.WriteString(data.StreamInterface)
		signature.WriteString(") (err error)")
	} else {
		if data.SkipRequestBodyEncodeDecode {
			signature.WriteString(", req io.ReadCloser")
		}
		signature.WriteString(") (")
		if data.Result != "" {
			signature.WriteString("res ")
			signature.WriteString(data.ResultFullRef)
			signature.WriteString(", ")
		}
		if data.SkipResponseBodyEncodeDecode {
			signature.WriteString("resp io.ReadCloser, ")
		}
		if data.ViewedResult != nil && data.ViewedResult.ViewName == "" {
			signature.WriteString("view string, ")
		}
		signature.WriteString("err error)")
	}

	var body strings.Builder
	body.WriteString(signature.String())
	body.WriteString(" {\n")
	if data.SkipRequestBodyEncodeDecode {
		body.WriteString("// req is the HTTP request body stream.\n")
		body.WriteString("defer req.Close()\n")
	}
	if data.Result != "" && data.ResultIsStruct && data.ServerStream == nil {
		body.WriteString("res = &")
		body.WriteString(data.ResultFullName)
		body.WriteString("{}\n")
	}
	if data.SkipResponseBodyEncodeDecode {
		body.WriteString("// resp is the HTTP response body stream.\n")
		body.WriteString(`resp = io.NopCloser(strings.NewReader("`)
		body.WriteString(data.Name)
		body.WriteString(`"))` + "\n")
	}
	if data.ViewedResult != nil && data.ViewedResult.ViewName == "" {
		if data.ServerStream != nil {
			body.WriteString("stream.SetView(")
			body.WriteString(fmt.Sprintf("%q", data.ResultView))
			body.WriteString(")\n")
		} else {
			body.WriteString("view = ")
			body.WriteString(fmt.Sprintf("%q", data.ResultView))
			body.WriteString("\n")
		}
	}
	body.WriteString(`log.Printf(ctx, "`)
	body.WriteString(data.ServiceVarName)
	body.WriteString(".")
	body.WriteString(data.Name)
	body.WriteString("\")\n")
	if data.ServerStream != nil && data.IsJSONRPC && data.ResultFullName != "" {
		body.WriteString("// Minimal example: emit one progress notification and one final response\n{\n")
		body.WriteString("notif := ")
		body.WriteString(exampleStreamValue(data, "progress"))
		body.WriteString("\nif err := stream.Send(ctx, notif); err != nil {\nreturn err\n}\n")
		body.WriteString("final := ")
		body.WriteString(exampleStreamValue(data, "done"))
		body.WriteString("\nreturn stream.SendAndClose(ctx, final)\n}\n")
	}
	body.WriteString("return\n}")
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
