package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
)

// textFieldLoadCode converts a string flag through the custom field's parser.
// Validation uses the original text and optional values retain their pointer.
func textFieldLoadCode(flag *cli.FlagData, arg *InitArgData, payloadRef string) (*jen.Statement, bool) {
	raw := arg.Locals.Raw
	code := jen.Id(raw).Op(":=").Id(flag.FullName)
	if arg.DefaultValue != nil {
		code.Line().If(jen.Id(raw).Op("==").Lit("")).Block(jen.Id(raw).Op("=").Lit(arg.DefaultValue))
	}
	parse := new(jen.Statement)
	target := arg.VarName
	if arg.Pointer {
		target = arg.Locals.Val
		parse.Var().Id(target).Add(codegen.TypeRef(arg.TypeName)).Line()
	}
	parse.If(jen.Err().Op(":=").Id(target).Dot("UnmarshalText").Call(jen.Index().Byte().Call(jen.Id(raw))), jen.Err().Op("!=").Nil()).Block(
		jen.Var().Id("zero").Add(codegen.TypeRef(payloadRef)),
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("invalid value for "+arg.Name+": %w"), jen.Err())),
	)
	if arg.Validate != "" {
		parse.Line().Add(codegen.Expr(arg.Validate)).Line().If(jen.Err().Op("!=").Nil()).Block(
			jen.Var().Id("zero").Add(codegen.TypeRef(payloadRef)),
			jen.Return(jen.Id("zero"), jen.Err()),
		)
	}
	if arg.Pointer {
		parse.Line().Id(arg.VarName).Op("=").Op("&").Id(target)
	}
	if !arg.Required {
		code.Line().If(jen.Id(raw).Op("!=").Lit("")).Block(parse)
	} else {
		code.Line().Add(parse)
	}
	return code, arg.Validate != ""
}
