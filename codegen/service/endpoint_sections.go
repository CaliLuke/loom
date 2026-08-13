package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

var multilineValues = jen.Options{
	Open:      "{",
	Close:     "}",
	Separator: ",",
	Multi:     true,
}

func endpointsStructSection(data *EndpointsData) codegen.Section {
	return codegen.NewJenniferSection("endpoints-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, data.Description)
		stmt.Type().Id(data.VarName).StructFunc(func(group *jen.Group) {
			for _, method := range data.Methods {
				group.Id(method.VarName).Add(codegen.Expr("loom.Endpoint"))
			}
		})
		stmt.Line()
	})
}

func endpointStreamStructSection(method *EndpointMethodData) codegen.Section {
	return codegen.NewJenniferSection("endpoint-input-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s holds both the payload and the server stream of the %q method.", method.ServerStream.EndpointStruct, method.Name))
		stmt.Type().Id(method.ServerStream.EndpointStruct).StructFunc(func(group *jen.Group) {
			if method.PayloadRef != "" {
				groupDoc(group, "Payload is the method payload.")
				group.Id("Payload").Add(codegen.TypeRef(method.PayloadRef))
			}
			if method.IsJSONRPC {
				groupDoc(group, "RequestID is the JSON-RPC request ID (available for JSON-RPC transports).")
				group.Id("RequestID").Any()
			}
			groupDoc(group, fmt.Sprintf("Stream is the server stream used by the %q method to send data.", method.Name))
			group.Id("Stream").Add(codegen.TypeRef(method.ServerStream.Interface))
		})
		stmt.Line()
	})
}

func requestBodyStructSection(method *EndpointMethodData) codegen.Section {
	return codegen.NewJenniferSection("request-body-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s holds both the payload and the HTTP request body reader of the %q method.", method.RequestStruct, method.Name))
		stmt.Type().Id(method.RequestStruct).StructFunc(func(group *jen.Group) {
			if method.PayloadRef != "" {
				groupDoc(group, "Payload is the method payload.")
				group.Id("Payload").Add(codegen.TypeRef(method.PayloadRef))
			}
			groupDoc(group, "Body streams the HTTP request body.")
			group.Id("Body").Qual("io", "ReadCloser")
		})
		stmt.Line()
	})
}

func responseBodyStructSection(method *EndpointMethodData) codegen.Section {
	return codegen.NewJenniferSection("response-body-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s holds both the result and the HTTP response body reader of the %q method.", method.ResponseStruct, method.Name))
		stmt.Type().Id(method.ResponseStruct).StructFunc(func(group *jen.Group) {
			if method.ResultRef != "" {
				groupDoc(group, "Result is the method result.")
				group.Id("Result").Add(codegen.TypeRef(method.ResultRef))
			}
			groupDoc(group, "Body streams the HTTP response body.")
			group.Id("Body").Qual("io", "ReadCloser")
		})
		stmt.Line()
	})
}

func fileResponseStructSection(method *EndpointMethodData) codegen.Section {
	return codegen.NewJenniferSection("file-response-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s holds both the result and seekable file response of the %q method.", method.FileResponseStruct, method.Name))
		stmt.Type().Id(method.FileResponseStruct).StructFunc(func(group *jen.Group) {
			if method.ResultRef != "" {
				groupDoc(group, "Result is the method result.")
				group.Id("Result").Add(codegen.TypeRef(method.ResultRef))
			}
			groupDoc(group, "File is the seekable HTTP file response.")
			group.Id("File").Add(codegen.TypeRef("*loomhttp.FileResponse"))
		})
		stmt.Line()
	})
}

func endpointsInitSection(data *EndpointsData) codegen.Section {
	return codegen.NewJenniferSection("endpoints-init", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("New%s wraps the methods of the %q service with endpoints.", data.VarName, data.Name))
		stmt.Func().Id("New" + data.VarName).ParamsFunc(func(group *jen.Group) {
			group.Id("s").Id(data.ServiceVarName)
			if data.HasServerInterceptors {
				group.Id("si").Id("ServerInterceptors")
			}
		}).Op("*").Id(data.VarName).BlockFunc(func(group *jen.Group) {
			if len(data.Schemes) > 0 {
				group.Comment("Casting service to Authorizer interface")
				group.Id("a").Op(":=").Id("s").Assert(jen.Id("Authorizer"))
			}
			if data.HasServerInterceptors {
				group.Id("endpoints").Op(":=").Add(multilineEndpointsLiteral(data, true))
			} else {
				group.Return(multilineEndpointsLiteral(data, true))
				return
			}
			for _, method := range data.Methods {
				if len(method.ServerInterceptors) == 0 {
					continue
				}
				group.Id("endpoints").Dot(method.VarName).Op("=").Id("Wrap"+method.VarName+"Endpoint").Call(
					jen.Id("endpoints").Dot(method.VarName),
					jen.Id("si"),
				)
			}
			group.Return(jen.Id("endpoints"))
		})
		stmt.Line()
	})
}

func endpointsUseSection(data *EndpointsData) codegen.Section {
	return codegen.NewJenniferSection("endpoints-use", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("Use applies the given middleware to all the %q service endpoints.", data.Name))
		stmt.Func().Params(jen.Id("e").Op("*").Id(data.VarName)).Id("Use").Params(
			jen.Id("m").Func().Params(codegen.Expr("loom.Endpoint")).Add(codegen.Expr("loom.Endpoint")),
		).BlockFunc(func(group *jen.Group) {
			for _, method := range data.Methods {
				group.Id("e").Dot(method.VarName).Op("=").Id("m").Call(jen.Id("e").Dot(method.VarName))
			}
		})
		stmt.Line()
	})
}

func multilineEndpointsLiteral(data *EndpointsData, pointer bool) *jen.Statement {
	lit := jen.Id(data.VarName).CustomFunc(multilineValues, func(group *jen.Group) {
		for _, method := range data.Methods {
			group.Id(method.VarName).Op(":").Add(newEndpointCall(method))
		}
	})
	if pointer {
		return jen.Op("&").Add(lit)
	}
	return lit
}

func newEndpointCall(method *EndpointMethodData) *jen.Statement {
	return jen.Id("New" + method.VarName + "Endpoint").CallFunc(func(group *jen.Group) {
		group.Id("s")
		for _, scheme := range method.Schemes.DedupeByType() {
			group.Id("a").Dot(scheme.Type + "Auth")
		}
	})
}

func groupDoc(group *jen.Group, text string) {
	for _, line := range strings.Split(codegen.Comment(text), "\n") {
		group.Comment(strings.TrimPrefix(line, "// "))
	}
}
