package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	codegenpkg "github.com/CaliLuke/loom/codegen"
)

func grpcTypeInitSection(init *InitData) codegenpkg.Section {
	return codegenpkg.NewJenniferSection("type-init", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, init.Description)
		params := make([]jen.Code, 0, len(init.Args))
		for _, arg := range init.Args {
			params = append(params, jen.Id(arg.Name).Add(codegenpkg.TypeRef(arg.TypeRef)))
		}
		result := jen.Empty().Add(codegenpkg.TypeRef(init.ReturnTypeRef))
		if init.ErrorAware {
			result = jen.Empty().Params(codegenpkg.TypeRef(init.ReturnTypeRef), jen.Error())
		}
		stmt.Func().Id(init.Name).
			Params(params...).
			Add(result).
			BlockFunc(func(g *jen.Group) {
				if init.ErrorAware {
					g.Id("transformErr").Op(":=").New(jen.Error())
				}
				g.Add(codegenpkg.Expr(init.Code))
				if init.FieldCode != "" {
					g.Add(codegenpkg.Expr(init.FieldCode))
				}
				if init.ErrorAware {
					g.If(jen.Op("*").Id("transformErr").Op("!=").Nil()).Block(
						jen.Var().Id("zero").Add(codegenpkg.TypeRef(init.ReturnTypeRef)),
						jen.Return(jen.Id("zero"), jen.Op("*").Id("transformErr")),
					)
					g.Return(codegenpkg.Expr(init.ReturnVarName), jen.Nil())
					return
				}
				g.Return(codegenpkg.Expr(init.ReturnVarName))
			})
		stmt.Line()
	})
}

func grpcValidateSection(data *ValidationData) codegenpkg.Section {
	return codegenpkg.NewJenniferSection("validate", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s runs the validations defined on %s.", data.Name, data.SrcName))
		stmt.Func().Id(data.Name).
			Params(jen.Id(data.ArgName).Add(codegenpkg.TypeRef(data.SrcRef))).
			Params(jen.Err().Error()).
			Block(
				codegenpkg.Expr(data.Def),
				jen.Return(),
			)
		stmt.Line()
	})
}

func grpcTransformHelperSection(data *codegenpkg.TransformFunctionData) codegenpkg.Section {
	return codegenpkg.NewJenniferSection("transform-helper", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s builds a value of type %s from a value of type %s.", data.Name, data.ResultTypeRef, data.ParamTypeRef))
		params := []jen.Code{jen.Id("v").Add(codegenpkg.TypeRef(data.ParamTypeRef))}
		if data.ErrorAware {
			params = append(params, jen.Id("transformErr").Op("*").Error())
		}
		stmt.Func().Id(data.Name).
			Params(params...).
			Add(codegenpkg.TypeRef(data.ResultTypeRef)).
			Block(
				codegenpkg.Expr(data.Code),
				jen.Return(jen.Id("res")),
			)
		stmt.Line()
	})
}
