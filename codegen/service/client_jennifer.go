package service

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/v3/codegen"
)

func clientStructSection(data *EndpointsData) codegen.Section {
	return codegen.NewJenniferSection("client-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s is the %q service client.", data.ClientVarName, data.Name))
		stmt.Type().Id(data.ClientVarName).StructFunc(func(group *jen.Group) {
			for _, method := range data.Methods {
				group.Id(method.EndpointField).Add(codegen.Expr("goa.Endpoint"))
				if method.HasMixedResults {
					group.Id(method.StreamEndpointField).Add(codegen.Expr("goa.Endpoint"))
				}
			}
		})
		stmt.Line()
	})
}

func clientInitSection(data *EndpointsData) codegen.Section {
	return codegen.NewJenniferSection("client-init", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("New%s initializes a %q service client given the endpoints.", data.ClientVarName, data.Name))
		stmt.Func().Id("New" + data.ClientVarName).ParamsFunc(func(group *jen.Group) {
			if data.ClientInitArgs != "" {
				group.Id(data.ClientInitArgs).Add(codegen.Expr("goa.Endpoint"))
				if data.HasClientInterceptors {
					group.Id("ci").Id("ClientInterceptors")
				}
				return
			}
			if data.HasClientInterceptors {
				group.Id("ci").Id("ClientInterceptors")
			}
		}).Op("*").Id(data.ClientVarName).Block(
			jen.Return(
				jen.Op("&").Id(data.ClientVarName).ValuesFunc(func(group *jen.Group) {
					for _, method := range data.Methods {
						group.Id(method.EndpointField).Op(":").Add(wrappedClientEndpointExpr(method, false))
						if method.HasMixedResults {
							group.Id(method.StreamEndpointField).Op(":").Add(wrappedClientEndpointExpr(method, true))
						}
					}
				}),
			),
		)
		stmt.Line()
	})
}

func methodSection(method *EndpointMethodData) codegen.Section {
	return codegen.NewJenniferSection("client-method", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s calls the %q endpoint of the %q service.", method.VarName, method.Name, method.ServiceName))
		if len(method.Errors) > 0 {
			codegen.CommentBlock(stmt, fmt.Sprintf("%s may return the following errors:", method.VarName))
			for _, errData := range method.Errors {
				line := fmt.Sprintf("- %q (type %s)", errData.ErrName, errData.TypeRef)
				if errData.Description != "" {
					line += ": " + errData.Description
				}
				codegen.CommentBlock(stmt, "\t"+line)
			}
			codegen.CommentBlock(stmt, "\t- error: internal error")
		}
		buildClientMethod(stmt, method, false)
		if method.HasMixedResults {
			stmt.Line()
			codegen.Doc(stmt, fmt.Sprintf("%sStream calls the %q endpoint of the %q service with server streaming enabled.", method.VarName, method.Name, method.ServiceName))
			buildClientMethod(stmt, method, true)
		}
		stmt.Line()
	})
}

func buildClientMethod(stmt *jen.Statement, method *EndpointMethodData, stream bool) {
	name := method.VarName
	resultType := method.ResultRef
	if stream {
		name += "Stream"
		resultType = method.ClientStream.Interface
	} else if method.ClientStream != nil && !method.HasMixedResults {
		resultType = method.ClientStream.Interface
	}

	stmt.Func().Params(jen.Id("c").Op("*").Id(method.ClientVarName)).Id(name).ParamsFunc(func(group *jen.Group) {
		group.Id("ctx").Qual("context", "Context")
		if method.PayloadRef != "" {
			group.Id("p").Add(codegen.TypeRef(method.PayloadRef))
		}
		if method.MethodData.SkipRequestBodyEncodeDecode {
			group.Id("req").Qual("io", "ReadCloser")
		}
	}).ParamsFunc(func(group *jen.Group) {
		if resultType != "" {
			group.Id("res").Add(codegen.TypeRef(resultType))
		}
		if method.MethodData.SkipResponseBodyEncodeDecode {
			group.Id("resp").Qual("io", "ReadCloser")
		}
		group.Id("err").Error()
	}).BlockFunc(func(group *jen.Group) {
		if resultType != "" || method.MethodData.SkipResponseBodyEncodeDecode {
			group.Var().Id("ires").Any()
		}
		lhs := jen.Id("_")
		if resultType != "" || method.MethodData.SkipResponseBodyEncodeDecode {
			lhs = jen.Id("ires")
		}
		endpointField := method.EndpointField
		if stream {
			endpointField = method.StreamEndpointField
		}
		group.List(lhs, jen.Id("err")).Op("=").Id("c").Dot(endpointField).Call(jen.Id("ctx"), requestExpr(method))
		if resultType == "" && !method.MethodData.SkipResponseBodyEncodeDecode {
			group.Return()
			return
		}
		group.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(),
		)
		if method.MethodData.SkipResponseBodyEncodeDecode {
			group.Id("o").Op(":=").Id("ires").Assert(jen.Op("*").Id(method.MethodData.ResponseStruct))
			group.ReturnFunc(func(returnGroup *jen.Group) {
				if method.ResultRef != "" {
					returnGroup.Id("o").Dot("Result")
				}
				returnGroup.Id("o").Dot("Body")
				returnGroup.Nil()
			})
			return
		}
		group.Return(
			jen.Id("ires").Assert(codegen.TypeRef(resultType)),
			jen.Nil(),
		)
	})
}

func wrappedClientEndpointExpr(method *EndpointMethodData, stream bool) *jen.Statement {
	argName := method.ArgName
	if stream {
		argName = method.StreamArgName
	}
	if len(method.ClientInterceptors) == 0 {
		return jen.Id(argName)
	}
	return jen.Id("Wrap"+method.VarName+"ClientEndpoint").Call(
		jen.Id(argName),
		jen.Id("ci"),
	)
}

func requestExpr(method *EndpointMethodData) *jen.Statement {
	if method.MethodData.SkipRequestBodyEncodeDecode {
		values := make([]jen.Code, 0, 2)
		if method.PayloadRef != "" {
			values = append(values, jen.Id("Payload").Op(":").Id("p"))
		}
		values = append(values, jen.Id("Body").Op(":").Id("req"))
		return jen.Op("&").Id(method.RequestStruct).Values(values...)
	}
	if method.PayloadRef != "" {
		return jen.Id("p")
	}
	return jen.Nil()
}
