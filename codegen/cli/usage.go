package cli

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

// UsageCommands builds a section that generates a help text showing
// the list of allowed commands and sub-commands.
func UsageCommands(data []*CommandData) codegen.Section {
	usages := make([]string, len(data))
	for i, cmd := range data {
		subs := make([]string, len(cmd.Subcommands))
		for i, s := range cmd.Subcommands {
			subs[i] = s.Name
		}
		var lp, rp string
		if len(subs) > 1 {
			lp = "("
			rp = ")"
		}
		usages[i] = fmt.Sprintf("%s %s%s%s", cmd.Name, lp, strings.Join(subs, "|"), rp)
	}

	return codegen.MustJenniferSection("cli-usage-commands", func(stmt *jen.Statement) {
		stmt.Comment("UsageCommands returns the set of commands and sub-commands using the format").Line()
		stmt.Comment("").Line()
		stmt.Comment("   command (subcommand1|subcommand2|...)").Line()
		stmt.Func().Id("UsageCommands").Params().Index().String().BlockFunc(func(group *jen.Group) {
			group.Return().Index().String().ValuesFunc(func(values *jen.Group) {
				for _, usage := range usages {
					values.Lit(usage)
				}
			})
		})
	})
}

// UsageExamples builds a section that generates a help text showing
// a valid invocation of the CLI tool.
func UsageExamples(data []*CommandData) codegen.Section {
	var examples []string
	for i, cmd := range data {
		if i < 5 {
			examples = append(examples, cmd.Example)
		}
	}

	return codegen.MustJenniferSection("cli-usage-examples", func(stmt *jen.Statement) {
		stmt.Comment("UsageExamples produces an example of a valid invocation of the CLI tool.").Line()
		stmt.Func().Id("UsageExamples").Params().String().BlockFunc(func(group *jen.Group) {
			if len(examples) == 0 {
				group.Return(jen.Lit(""))
				return
			}
			var expr *jen.Statement
			for i, example := range examples {
				part := jen.Id("os").Dot("Args").Index(jen.Lit(0)).Op("+").Lit(" " + example + "\\n")
				if i == 0 {
					expr = part
					continue
				}
				expr.Op("+").Add(part)
			}
			group.Return(expr)
		})
	})
}

// CommandUsage builds the section that can be used to generate the
// endpoint command usage code.
func CommandUsage(data *CommandData) codegen.Section {
	return codegen.MustJenniferSection("cli-command-usage", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%sUsage displays the usage of the %s command and its subcommands.", data.VarName, data.Name))
		stmt.Func().Id(data.VarName + "Usage").Params().BlockFunc(func(group *jen.Group) {
			group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit(printDescription(data.Description)))
			group.Qual("fmt", "Fprintf").Call(jen.Qual("os", "Stderr"), jen.Lit("Usage:\n    %s [globalflags] "+data.Name+" COMMAND [flags]\n\n"), jen.Qual("os", "Args").Index(jen.Lit(0)))
			group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("COMMAND:"))
			for _, sub := range data.Subcommands {
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("    "+sub.Name+": "+printDescription(sub.Description)))
			}
			group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"))
			group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("Additional help:"))
			group.Qual("fmt", "Fprintf").Call(jen.Qual("os", "Stderr"), jen.Lit("    %s "+data.Name+" COMMAND --help\n"), jen.Qual("os", "Args").Index(jen.Lit(0)))
		})
		stmt.Line()
		for _, sub := range data.Subcommands {
			stmt.Func().Id(sub.FullName + "Usage").Params().BlockFunc(func(group *jen.Group) {
				group.Comment("Header with flags")
				group.Qual("fmt", "Fprintf").Call(jen.Qual("os", "Stderr"), jen.Lit("%s [flags] "+data.Name+" "+sub.Name), jen.Qual("os", "Args").Index(jen.Lit(0)))
				for _, flag := range sub.Flags {
					group.Qual("fmt", "Fprint").Call(jen.Qual("os", "Stderr"), jen.Lit(" -"+flag.Name+" "+flag.Type))
				}
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"))
				group.Line()
				group.Comment("Description")
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"))
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit(printDescription(sub.Description)))
				group.Line()
				group.Comment("Flags list")
				for _, flag := range sub.Flags {
					group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("    -"+flag.Name+" "+flag.Type+": "+flag.Description))
				}
				group.Line()
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"))
				group.Qual("fmt", "Fprintln").Call(jen.Qual("os", "Stderr"), jen.Lit("Example:"))
				group.Qual("fmt", "Fprintf").Call(jen.Qual("os", "Stderr"), jen.Lit("    %s %s\n"), jen.Qual("os", "Args").Index(jen.Lit(0)), jen.Lit(sub.Example))
			})
			stmt.Line()
		}
	})
}

func printDescription(desc string) string {
	res := strings.ReplaceAll(desc, "`", "`+\"`\"+`")
	res = strings.ReplaceAll(res, "\n", "\n\t")
	return res
}
