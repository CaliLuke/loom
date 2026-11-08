package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/uniongen"
)

func typeDeclSection(name string, data *TypeData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		addHTTPDocOrBlank(stmt, data.Description)
		decl := stmt.Type().Id(data.VarName)
		if data.Def == "loom.JSONValue" {
			decl.Op("=")
		}
		decl.Add(codegen.Expr(data.Def))
		stmt.Line()
	})
}

func unionTypeSection(name string, data *uniongen.Type) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		uniongen.Emit(stmt, data)
	})
}

// appendInitResult appends the statements of the constructor init that build
// its result from the transform code: the returned struct res when init
// initializes the single attribute ReturnTypeAttribute, or v otherwise.
func appendInitResult(group *jen.Group, init *InitData, code string, client bool) {
	switch {
	case !client && init.serverOptionalObjectBody:
		group.Var().Id("v").Add(codegen.TypeRef(init.ReturnTypeRef))
		if init.serverBodyEnvelope {
			group.Id("v").Op("=").Op("&").Add(codegen.TypeRef(init.ReturnTypeName)).Values()
		}
		group.If(jen.Id("body").Op("!=").Nil()).BlockFunc(func(body *jen.Group) {
			appendHTTPRawBlock(body, code)
		})
	case code != "" && !client && init.ReturnIsOptionalBody && init.ReturnTypeAttribute != "":
		// A nil body is an absent request body: leave the attribute nil.
		group.Id("res").Op(":=").Op("&").Id(init.ReturnTypeName).Values()
		group.If(jen.Id("body").Op("!=").Nil()).BlockFunc(func(body *jen.Group) {
			appendHTTPRawBlock(body, code)
			value := body.Id("res").Dot(init.ReturnTypeAttribute).Op("=")
			if init.ReturnIsPrimitivePointer {
				value.Op("&")
			}
			value.Id("v")
		})
	case code != "":
		appendHTTPRawBlock(group, code)
		if init.ReturnTypeAttribute != "" {
			valueExpr := "v"
			if init.ReturnIsPrimitivePointer {
				valueExpr = "&v"
			} else if init.ReturnIsUnionValue {
				valueExpr = "*v"
			}
			group.Id("res").Op(":=").Op("&").Id(init.ReturnTypeName).CustomFunc(jen.Options{
				Open:      "{",
				Close:     "}",
				Separator: ",",
				Multi:     true,
			}, func(values *jen.Group) {
				values.Id(init.ReturnTypeAttribute).Op(":").Add(codegen.Expr(valueExpr))
			})
		}
	case client && init.nullableProjection != nil && init.nullableProjection.metadataOnly:
		group.Id("v").Op(":=").Op("&").Id(init.ReturnTypeName).Values()
	case client && init.errorConstructor != "":
		group.Var().Id("v").Add(codegen.TypeRef(init.ReturnTypeRef))
	case init.ReturnIsStruct:
		if init.ReturnTypeAttribute != "" {
			group.Id("res").Op(":=").Op("&").Id(init.ReturnTypeName).Values()
		} else {
			group.Id("v").Op(":=").Op("&").Id(init.ReturnTypeName).Values()
		}
	}
}

func validateSection(name string, data *TypeData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		stmt.Line()
		function := data.validationFunction
		codegen.Doc(stmt, fmt.Sprintf("%s runs the validations defined on %s", function.Name, data.Name))
		stmt.Add(codegen.Expr(function.Definition(data.ValidateDef)))
		stmt.Line()
	})
}

func initRenderData(init *InitData, client bool) ([]*InitArgData, string) {
	if client {
		return init.ClientArgs, strings.TrimRight(init.ClientCode, "\n\t ")
	}
	return init.ServerArgs, strings.TrimRight(init.ServerCode, "\n\t ")
}

func initRenderTarget(client bool) string {
	if client {
		return "client"
	}
	return "server"
}

func addHTTPDocOrBlank(stmt *jen.Statement, description string) {
	if strings.TrimSpace(description) == "" {
		stmt.Line()
		return
	}
	codegen.Doc(stmt, description)
}
