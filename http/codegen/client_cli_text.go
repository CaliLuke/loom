package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func isTextUnionArg(arg *InitArgData) bool {
	return arg.parameter != nil && arg.parameter.Styled && expr.AsUnion(arg.Type) != nil
}

func textUnionFieldLoadCode(flag *cli.FlagData, arg *InitArgData, payloadRef string) *jen.Statement {
	var out strings.Builder
	element := arg.parameter
	out.WriteString("{\n")
	if !arg.Required {
		fmt.Fprintf(&out, "if %s != \"\" {\n", flag.FullName)
	}
	out.WriteString("if flagError := func() error {\n")
	switch element.location {
	case "query":
		if textUnionFlagUsesPairs(element) {
			fmt.Fprintf(&out, "parameterPairs, err := loomhttp.ParseRawPairs(%s)\n", flag.FullName)
		} else {
			fmt.Fprintf(&out, "parameterPairs, err := loomhttp.RawPairsFromValue(%q, %s)\n", element.HTTPName, flag.FullName)
		}
		out.WriteString("if err != nil {\nreturn err\n}\n")
	case "path":
		fmt.Fprintf(&out, "parameterPath := %s\n", flag.FullName)
	case "header":
		fmt.Fprintf(&out, "parameterHeaders := make(http.Header)\nparameterHeaders.Set(%q, %s)\n", element.HTTPName, flag.FullName)
	case "cookie":
		if textUnionFlagUsesPairs(element) {
			fmt.Fprintf(&out, "parameterCookies, err := http.ParseCookie(%s)\nif err != nil {\nreturn err\n}\n", flag.FullName)
		} else {
			fmt.Fprintf(&out, "parameterCookie := &http.Cookie{Name: %q, Value: %s}\nif err := parameterCookie.Valid(); err != nil {\nreturn err\n}\nparameterCookies := []*http.Cookie{parameterCookie}\n", element.HTTPName, flag.FullName)
		}
	}
	out.WriteString(renderTextUnionMatcher(element, element.location))
	fmt.Fprintf(&out, "matched, present, _, err := matchTextUnion()\nif err != nil {\nreturn err\n}\nif !present {\nreturn fmt.Errorf(\"text union flag has no value\")\n}\n%s = matched\nreturn nil\n}(); flagError != nil {\nvar zero %s\nreturn zero, flagError\n}\n", arg.VarName, payloadRef)
	if !arg.Required {
		out.WriteString("}\n")
	}
	out.WriteString("}\n")
	return codegen.Expr(out.String())
}

// textUnionFlagUsesPairs selects the flag grammar from resolved ownership and
// multiplicity, including nested branches,
// never from the supplied value. All candidates receive the same raw carrier.
func textUnionFlagUsesPairs(element *Element) bool {
	for _, branch := range element.Codec.Branches {
		if textUnionCodecUsesPairs(branch, element.HTTPName) {
			return true
		}
	}
	return false
}

func textUnionCodecUsesPairs(branch httpencoding.BranchCodec, name string) bool {
	if branch.Keys.Residual || len(branch.Keys.Prefixes) > 0 || len(branch.Keys.Literals) != 1 || branch.Keys.Literals[0] != name {
		return true
	}
	if branch.Codec.Shape == httpencoding.Sequence && branch.Codec.Explode {
		return true
	}
	for _, inner := range branch.Inner {
		if textUnionCodecUsesPairs(inner, name) {
			return true
		}
	}
	return false
}

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
