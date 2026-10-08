package cli

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
)

// BuildCommandData builds the data needed by CLI code generators to render the
// parsing of the service command.
func BuildCommandData(data *service.Data) *CommandData {
	description := data.Description
	if description == "" {
		description = fmt.Sprintf("Make requests to the %q service", data.Name)
	}

	var interceptors *InterceptorData
	if len(data.ClientInterceptors) > 0 {
		interceptors = &InterceptorData{
			VarName: codegen.Goify(data.Name, false) + "Inter",
			PkgName: data.PkgName,
		}
	}

	return &CommandData{
		Name:         codegen.KebabCase(data.Name),
		VarName:      codegen.Goify(data.Name, false),
		Description:  description,
		PkgName:      data.PkgName + "c",
		UsageName:    codegen.Goify(data.Name, false) + "Usage",
		FlagSetName:  codegen.Goify(data.Name, false) + "Flags",
		Interceptors: interceptors,
	}
}

// BuildSubcommandData builds the data needed by CLI code generators to render
// the CLI parsing of the service sub-command.
func BuildSubcommandData(data *service.Data, m *service.MethodData, buildFunction *BuildFunctionData, flags []*FlagData) *SubcommandData {
	en := m.Name
	name := codegen.KebabCase(en)
	fullName := goifyTerms(data.Name, en)
	description := subcommandDescription(m)
	conversion := buildSubcommandConversion(m.Payload, buildFunction, flags)
	interceptors := buildSubcommandInterceptors(data, m)
	sub := &SubcommandData{
		Name:          name,
		FullName:      fullName,
		PayloadType:   m.Payload,
		UsageName:     fullName + "Usage",
		FlagSetName:   fullName + "Flags",
		Description:   description,
		Flags:         flags,
		MethodVarName: m.VarName,
		BuildFunction: buildFunction,
		Conversion:    conversion,
		Interceptors:  interceptors,
	}
	generateExample(sub, data.Name)

	return sub
}

func subcommandDescription(m *service.MethodData) string {
	if m.Description != "" {
		return m.Description
	}
	return fmt.Sprintf("Make request to the %q endpoint", m.Name)
}

func buildSubcommandConversion(payload string, buildFunction *BuildFunctionData, flags []*FlagData) *jen.Statement {
	if payload == "" || buildFunction != nil || len(flags) == 0 {
		return nil
	}
	flag := flags[0]
	target, prefix, suffix := subcommandConversionTarget(payload)
	conv, _, check := conversionCode("*"+flagValueName(flag), target, payload, flag.Unmarshal, false)
	conversion := codegen.Expr(prefix).Add(conv).Add(codegen.Expr(suffix))
	if !check {
		return conversion
	}
	return codegen.Expr("var err error\n").Add(conversion).Line().If(jen.Err().Op("!=").Nil()).Block(
		buildSubcommandConversionError(flag, payload),
	)
}

func subcommandConversionTarget(payload string) (target, prefix, suffix string) {
	if flagType(payload) != "JSON" {
		return "data", "", ""
	}
	return "val", "var val " + payload + "\n", "\ndata = val"
}

func buildSubcommandConversionError(flag *FlagData, payload string) *jen.Statement {
	if flagType(payload) == "JSON" {
		if flag.Example == "" {
			return jen.Return(
				jen.Nil(),
				jen.Nil(),
				jen.Qual("fmt", "Errorf").Call(
					jen.Lit("invalid JSON for "+flag.FullName+"Flag, \nerror: %w"),
					jen.Err(),
				),
			)
		}
		return jen.Return(
			jen.Nil(),
			jen.Nil(),
			jen.Qual("fmt", "Errorf").Call(
				jen.Lit("invalid JSON for "+flag.FullName+"Flag, \nerror: %w, \nexample of valid JSON:\n%s"),
				jen.Err(),
				jen.Lit(flag.Example),
			),
		)
	}
	return jen.Return(
		jen.Nil(),
		jen.Nil(),
		jen.Qual("fmt", "Errorf").Call(
			jen.Lit("invalid value for "+flag.FullName+"Flag, must be "+flag.Type),
		),
	)
}

func buildSubcommandInterceptors(data *service.Data, m *service.MethodData) *InterceptorData {
	if len(m.ClientInterceptors) == 0 {
		return nil
	}
	return &InterceptorData{
		VarName: codegen.Goify(data.Name, false) + "Inter",
		PkgName: data.PkgName,
	}
}
