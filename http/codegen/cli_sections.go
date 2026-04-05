package codegen

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func parseEndpointSection(flagsCode string, commands []*commandData) codegen.Section {
	return codegen.MustJenniferSection("parse-endpoint", func(stmt *jen.Statement) {
		stmt.Comment("ParseEndpoint returns the endpoint and payload as specified on the command").Line()
		stmt.Comment("line.").Line()
		stmt.Func().
			Id("ParseEndpoint").
			ParamsFunc(func(group *jen.Group) {
				group.Id("scheme").String()
				group.Id("host").String()
				group.Id("doer").Add(codegen.TypeRef("loomhttp.Doer"))
				group.Id("enc").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Encoder"))
				group.Id("dec").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder"))
				group.Id("restore").Bool()
				appendParseEndpointStreamingParams(group, commands)
				appendParseEndpointCommandParams(group, commands)
			}).
			Params(codegen.TypeRef("loom.Endpoint"), jen.Any(), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				appendHTTPRawBlock(group, renderParseEndpointBody(flagsCode, commands))
			})
	})
}

func renderParseEndpointBody(flagsCode string, commands []*commandData) string {
	var b sourceBuilder
	b.Add(flagsCode)
	b.Add("\n\tvar (\n")
	b.Add("\t\tdata     any\n")
	b.Add("\t\tendpoint loom.Endpoint\n")
	b.Add("\t\terr      error\n")
	b.Add("\t)\n")
	b.Add("\t{\n")
	b.Add("\t\tswitch svcn {\n")
	for _, cmd := range commands {
		writeParseEndpointCommandCase(&b, cmd)
	}
	b.Add("\t\t}\n")
	b.Add("\t}\n")
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn nil, nil, err\n")
	b.Add("\t}\n\n")
	b.Add("\treturn endpoint, data, nil\n")
	return b.String()
}

func appendParseEndpointStreamingParams(group *jen.Group, commands []*commandData) {
	if !streamingCmdExists(commands) {
		return
	}
	group.Id("dialer").Add(codegen.TypeRef("loomhttp.Dialer"))
	for _, cmd := range commands {
		if cmd.NeedDialer {
			group.Id(cmd.VarName + "Configurer").Add(codegen.TypeRef("*" + cmd.PkgName + ".ConnConfigurer"))
		}
	}
}

func appendParseEndpointCommandParams(group *jen.Group, commands []*commandData) {
	for _, cmd := range commands {
		for _, sub := range cmd.Subcommands {
			if sub.MultipartVarName != "" {
				group.Id(sub.MultipartVarName).Add(codegen.TypeRef(cmd.PkgName + "." + sub.MultipartFuncName))
			}
		}
		if cmd.Interceptors != nil {
			group.Id(cmd.Interceptors.VarName).Add(codegen.TypeRef(cmd.Interceptors.PkgName + ".ClientInterceptors"))
		}
	}
}

func writeParseEndpointCommandCase(b *sourceBuilder, cmd *commandData) {
	b.Addf("\t\tcase %q:\n", cmd.Name)
	writeParseEndpointClientInit(b, cmd)
	b.Add("\t\t\tswitch epn {\n")
	for _, sub := range cmd.Subcommands {
		writeParseEndpointSubcommandCase(b, cmd, sub)
	}
	b.Add("\t\t\t}\n")
}

func writeParseEndpointClientInit(b *sourceBuilder, cmd *commandData) {
	b.Addf("\t\t\tc := %s.NewClient(scheme, host, doer, enc, dec, restore", cmd.PkgName)
	if cmd.NeedDialer {
		b.Addf(", dialer, %sConfigurer", cmd.VarName)
	}
	b.Add(")\n")
}

func writeParseEndpointSubcommandCase(b *sourceBuilder, cmd *commandData, sub *subcommandData) {
	b.Addf("\t\t\tcase %q:\n", sub.Name)
	writeParseEndpointEndpointInit(b, cmd, sub)
	writeParseEndpointPayloadInit(b, cmd, sub)
	writeParseEndpointStreamPayloadInit(b, cmd, sub)
}

func writeParseEndpointEndpointInit(b *sourceBuilder, cmd *commandData, sub *subcommandData) {
	b.Addf("\t\t\t\tendpoint = c.%s(", sub.MethodVarName)
	if sub.MultipartVarName != "" {
		b.Add(sub.MultipartVarName)
	}
	b.Add(")\n")
	if sub.Interceptors != nil {
		b.Addf("\t\t\t\tendpoint = %s.Wrap%sClientEndpoint(endpoint, %s)\n", sub.Interceptors.PkgName, sub.MethodVarName, cmd.Interceptors.VarName)
	}
}

func writeParseEndpointPayloadInit(b *sourceBuilder, cmd *commandData, sub *subcommandData) {
	switch {
	case sub.BuildFunction != nil:
		b.Addf("\t\t\t\tdata, err = %s.%s(", cmd.PkgName, sub.BuildFunction.Name)
		for _, param := range sub.BuildFunction.ActualParams {
			b.Addf("*%sFlag, ", param)
		}
		b.Add(")\n")
	case sub.Conversion != "":
		b.Addf("\t\t\t\t%s\n", sub.Conversion)
	}
}

func writeParseEndpointStreamPayloadInit(b *sourceBuilder, cmd *commandData, sub *subcommandData) {
	if sub.StreamFlag == nil {
		return
	}
	if sub.BuildFunction != nil {
		b.Add("\t\t\t\tif err == nil {\n")
	}
	b.Addf("\t\t\t\t\tdata, err = %s.%s(", cmd.PkgName, sub.BuildStreamPayload)
	if sub.BuildFunction != nil || sub.Conversion != "" {
		b.Add("data, ")
	}
	b.Addf("*%sFlag)\n", sub.StreamFlag.FullName)
	if sub.BuildFunction != nil {
		b.Add("\t\t\t\t}\n")
	}
}

func pathSection(data *EndpointData) codegen.Section {
	return codegen.MustJenniferSection("path", func(stmt *jen.Statement) {
		for _, route := range data.Routes {
			if route.PathInit == nil {
				continue
			}
			stmt.Comment(route.PathInit.Description).Line()
			stmt.Func().
				Id(route.PathInit.Name).
				ParamsFunc(func(group *jen.Group) {
					for _, arg := range route.PathInit.ServerArgs {
						group.Id(arg.VarName).Add(codegen.TypeRef(arg.TypeRef))
					}
				}).
				Add(codegen.TypeRef(route.PathInit.ReturnTypeRef)).
				BlockFunc(func(group *jen.Group) {
					appendHTTPRawBlock(group, strings.TrimLeft(route.PathInit.ServerCode, "\n"))
				})
			stmt.Line()
		}
	})
}

func appendHTTPRawBlock(group *jen.Group, code string) {
	if strings.TrimSpace(code) == "" {
		return
	}
	if strings.HasPrefix(code, "\n") {
		group.Line()
	}
	group.Add(codegen.Expr(strings.TrimRight(code, "\n")))
}
