package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/uniongen"
)

func typeDeclSection(name string, data *TypeData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		addHTTPDocOrBlank(stmt, data.Description)
		decl := stmt.Type().Id(data.VarName)
		if data.Def == "loom.JSONValue" {
			decl.Op("=")
		}
		decl.Add(codegen.Expr(data.Def))
		if data.FlatFormUnionField == "" {
			stmt.Line()
			return
		}

		stmt.Line()
		codegen.CommentBlock(stmt, "MarshalFormValues marshals the synthetic request body wrapper using the wrapped union field at the top level.")
		marshal := stmt.Func().
			Params(jen.Id("body").Id(data.VarName)).
			Id("MarshalFormValues").
			Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
			Params(jen.Error())
		if data.FlatFormUnionPointer {
			marshal.Block(
				jen.If(jen.Id("body").Dot(data.FlatFormUnionField).Op("==").Nil()).Block(jen.Return(jen.Nil())),
				jen.Return(jen.Id("body").Dot(data.FlatFormUnionField).Dot("MarshalFormValues").Call(jen.Id("values"), jen.Id("prefix"))),
			)
		} else {
			marshal.Block(
				jen.Return(jen.Id("body").Dot(data.FlatFormUnionField).Dot("MarshalFormValues").Call(jen.Id("values"), jen.Id("prefix"))),
			)
		}

		stmt.Line()
		codegen.CommentBlock(stmt, "UnmarshalFormValues unmarshals the synthetic request body wrapper using the wrapped union field at the top level.")
		unmarshal := stmt.Func().
			Params(jen.Id("body").Op("*").Id(data.VarName)).
			Id("UnmarshalFormValues").
			Params(jen.Id("values").Qual("net/url", "Values"), jen.Id("prefix").String()).
			Params(jen.Error())
		if data.FlatFormUnionPointer {
			unmarshal.Block(
				jen.If(
					jen.Id("values").Dot("Get").Call(
						jen.Id("loomhttp").Dot("FormChildKey").Call(jen.Id("prefix"), jen.Lit(data.FlatFormUnionTypeKey)),
					).Op("==").Lit(""),
				).Block(jen.Return(jen.Nil())),
				jen.Var().Id("value").Add(codegen.Expr(strings.TrimPrefix(data.FlatFormUnionRef, "*"))),
				jen.If(jen.Err().Op(":=").Id("value").Dot("UnmarshalFormValues").Call(jen.Id("values"), jen.Id("prefix")), jen.Err().Op("!=").Nil()).Block(
					jen.Return(jen.Err()),
				),
				jen.Id("body").Dot(data.FlatFormUnionField).Op("=").Op("&").Id("value"),
				jen.Return(jen.Nil()),
			)
		} else {
			unmarshal.Block(
				jen.Return(
					jen.Op("(&").Id("body").Dot(data.FlatFormUnionField).Op(")").
						Dot("UnmarshalFormValues").
						Call(jen.Id("values"), jen.Id("prefix")),
				),
			)
		}
		stmt.Line()
	})
}

func unionTypeSection(name string, data *uniongen.Type) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		uniongen.Emit(stmt, data)
	})
}

func bodyInitSection(name string, init *InitData, client bool) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		args, code := initRenderData(init, client)
		stmt.Line()
		codegen.Doc(stmt, init.Description)
		stmt.Func().
			Id(init.Name).
			ParamsFunc(func(group *jen.Group) {
				for _, arg := range args {
					group.Id(arg.VarName).Add(codegen.TypeRef(arg.TypeRef))
				}
			}).
			Add(codegen.TypeRef(init.ReturnTypeRef)).
			BlockFunc(func(group *jen.Group) {
				if code != "" {
					appendHTTPRawBlock(group, code)
				}
				group.Return(jen.Id("body"))
			})
		stmt.Line()
	})
}

func typeInitSection(name string, init *InitData, client bool, pkgs *codegen.NameScope) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		args, code := initRenderData(init, client)
		typ := initRenderTarget(client)
		fieldInitCode := ""
		if !init.SkipFieldInit {
			fieldInitCode = strings.TrimRight(fieldCode(init, typ, pkgs), "\n\t ")
		}

		stmt.Line()
		codegen.Doc(stmt, init.Description)
		stmt.Func().
			Id(init.Name).
			ParamsFunc(func(group *jen.Group) {
				for _, arg := range args {
					group.Id(arg.VarName).Add(codegen.TypeRef(arg.TypeRef))
				}
			}).
			Add(codegen.TypeRef(init.ReturnTypeRef)).
			BlockFunc(func(group *jen.Group) {
				appendInitResult(group, init, code, client)
				if fieldInitCode != "" {
					appendHTTPRawBlock(group, fieldInitCode)
				}
				if code != "" || fieldInitCode != "" {
					group.Line()
				}
				if init.ReturnTypeAttribute != "" {
					group.Return(jen.Id("res"))
					return
				}
				group.Return(jen.Id("v"))
			})
		stmt.Line()
	})
}

// appendInitResult appends the statements of the constructor init that build
// its result from the transform code: the returned struct res when init
// initializes the single attribute ReturnTypeAttribute, or v otherwise.
func appendInitResult(group *jen.Group, init *InitData, code string, client bool) {
	switch {
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
		codegen.Doc(stmt, fmt.Sprintf("Validate%s runs the validations defined on %s", data.VarName, data.Name))
		stmt.Func().
			Id("Validate" + data.VarName).
			Params(jen.Id("body").Add(codegen.TypeRef(data.Ref))).
			Params(jen.Id("err").Error()).
			BlockFunc(func(group *jen.Group) {
				if data.ValidateDef != "" {
					appendHTTPRawBlock(group, data.ValidateDef)
				}
				group.Return()
			})
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
