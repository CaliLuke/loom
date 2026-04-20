package cli

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

// FlagsCode returns a string containing the code that parses the command-line
// flags to infer the command (service), sub-command (method), and the
// arguments (method payload) invoked by the tool. It panics if any error
// occurs during the generation of flag parsing code.
func FlagsCode(data []*CommandData) string {
	return renderJenniferStatement(FlagsCodeStatement(data))
}

// FlagsCodeStatement builds the CLI flag parsing statements.
func FlagsCodeStatement(data []*CommandData) *jen.Statement {
	stmt := new(jen.Statement)
	appendFlagDefinitions(stmt, data)
	appendUsageAssignments(stmt, data)
	appendTopLevelParsing(stmt)
	appendServiceSelection(stmt, data)
	appendEndpointSelection(stmt, data)
	appendEndpointFlagParsing(stmt)
	stmt.Line()
	return stmt
}

func appendFlagDefinitions(stmt *jen.Statement, data []*CommandData) {
	stmt.Var().DefsFunc(func(group *jen.Group) {
		for _, cmd := range data {
			group.Id(cmd.VarName+"Flags").Op("=").Qual("flag", "NewFlagSet").Call(jen.Lit(cmd.Name), jen.Qual("flag", "ContinueOnError"))
			appendSubcommandFlagDefinitions(group, cmd.Subcommands)
		}
	}).Line()
}

func appendSubcommandFlagDefinitions(group *jen.Group, subcommands []*SubcommandData) {
	for _, sub := range subcommands {
		group.Id(sub.FullName+"Flags").Op("=").Qual("flag", "NewFlagSet").Call(jen.Lit(sub.Name), jen.Qual("flag", "ExitOnError"))
		for _, flag := range sub.Flags {
			group.Id(flag.FullName+"Flag").Op("=").Id(sub.FullName+"Flags").Dot("String").Call(
				jen.Lit(flag.Name),
				jen.Lit(flagDefaultValue(flag)),
				jen.Lit(flag.Description),
			)
		}
	}
}

func flagDefaultValue(flag *FlagData) string {
	if flag.Default != nil {
		return fmt.Sprint(flag.Default)
	}
	if flag.Required {
		return "REQUIRED"
	}
	return ""
}

func appendUsageAssignments(stmt *jen.Statement, data []*CommandData) {
	for _, cmd := range data {
		stmt.Id(cmd.VarName + "Flags").Dot("Usage").Op("=").Id(cmd.VarName + "Usage")
		stmt.Line()
		for _, sub := range cmd.Subcommands {
			stmt.Id(sub.FullName + "Flags").Dot("Usage").Op("=").Id(sub.FullName + "Usage")
			stmt.Line()
		}
	}
	stmt.Line()
}

func appendTopLevelParsing(stmt *jen.Statement) {
	stmt.If(
		jen.Err().Op(":=").Qual("flag", "CommandLine").Dot("Parse").Call(jen.Qual("os", "Args").Index(jen.Lit(1), jen.Empty())),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Nil(), jen.Nil(), jen.Err()),
	)
	stmt.Line()
	stmt.If(jen.Qual("flag", "NArg").Call().Op("<").Lit(2)).Block(
		jen.Return(jen.Nil(), jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("not enough arguments"))),
	)
	stmt.Line()
}

func appendServiceSelection(stmt *jen.Statement, data []*CommandData) {
	stmt.Var().Defs(
		jen.Id("svcn").String(),
		jen.Id("svcf").Op("*").Qual("flag", "FlagSet"),
	)
	stmt.Line()
	stmt.Block(
		jen.Id("svcn").Op("=").Qual("flag", "Arg").Call(jen.Lit(0)),
		serviceSelectionSwitch(data),
	)
	stmt.Line()
	stmt.If(
		jen.Err().Op(":=").Id("svcf").Dot("Parse").Call(codegen.Expr("flag.Args()[1:]")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Nil(), jen.Nil(), jen.Err()),
	)
	stmt.Line()
}

func serviceSelectionSwitch(data []*CommandData) *jen.Statement {
	return jen.Switch(jen.Id("svcn")).BlockFunc(func(group *jen.Group) {
		for _, cmd := range data {
			group.Case(jen.Lit(cmd.Name)).Block(
				jen.Id("svcf").Op("=").Id(cmd.VarName + "Flags"),
			)
		}
		group.Default().Block(
			jen.Return(jen.Nil(), jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("unknown service %q"), jen.Id("svcn"))),
		)
	})
}

func appendEndpointSelection(stmt *jen.Statement, data []*CommandData) {
	stmt.Var().Defs(
		jen.Id("epn").String(),
		jen.Id("epf").Op("*").Qual("flag", "FlagSet"),
	)
	stmt.Line()
	stmt.Block(
		jen.Id("epn").Op("=").Id("svcf").Dot("Arg").Call(jen.Lit(0)),
		endpointSelectionSwitch(data),
	)
	stmt.Line()
	stmt.If(jen.Id("epf").Op("==").Nil()).Block(
		jen.Return(jen.Nil(), jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("unknown %q endpoint %q"), jen.Id("svcn"), jen.Id("epn"))),
	)
	stmt.Line()
}

func endpointSelectionSwitch(data []*CommandData) *jen.Statement {
	return jen.Switch(jen.Id("svcn")).BlockFunc(func(group *jen.Group) {
		for _, cmd := range data {
			group.Case(jen.Lit(cmd.Name)).Block(
				subcommandSelectionSwitch(cmd.Subcommands),
			)
		}
	})
}

func subcommandSelectionSwitch(subcommands []*SubcommandData) *jen.Statement {
	return jen.Switch(jen.Id("epn")).BlockFunc(func(group *jen.Group) {
		for _, sub := range subcommands {
			group.Case(jen.Lit(sub.Name)).Block(
				jen.Id("epf").Op("=").Id(sub.FullName + "Flags"),
			)
		}
	})
}

func appendEndpointFlagParsing(stmt *jen.Statement) {
	stmt.If(jen.Id("svcf").Dot("NArg").Call().Op(">").Lit(1)).Block(
		jen.If(
			jen.Err().Op(":=").Id("epf").Dot("Parse").Call(jen.Id("svcf").Dot("Args").Call().Index(jen.Lit(1), jen.Empty())),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Return(jen.Nil(), jen.Nil(), jen.Err()),
		),
	)
}

func renderJenniferStatement(stmt *jen.Statement) string {
	file := jen.NewFile("cli")
	file.Func().Id("render").Params().Params(jen.Any(), jen.Any(), jen.Error()).Block(stmt)
	var b strings.Builder
	if err := file.Render(&b); err != nil {
		panic(err)
	}
	src := b.String()
	marker := "func render() {\n"
	idx := strings.Index(src, marker)
	if idx == -1 {
		return src
	}
	body := src[idx+len(marker):]
	body = strings.TrimSuffix(body, "}\n")
	return body
}
