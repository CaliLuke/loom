package codegen

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
)

func parseEndpointSection(common []*cli.CommandData, commands []*commandData) codegen.Section {
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
				group.Add(cli.FlagsCodeStatement(common))
				appendHTTPParseEndpointBody(group, commands)
			})
	})
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

func appendHTTPParseEndpointBody(group *jen.Group, commands []*commandData) {
	group.Var().Defs(
		jen.Id("data").Any(),
		jen.Id("endpoint").Add(codegen.TypeRef("loom.Endpoint")),
		jen.Err().Error(),
	)
	group.BlockFunc(func(block *jen.Group) {
		block.Switch(jen.Id("svcn")).BlockFunc(func(switchGroup *jen.Group) {
			for _, cmd := range commands {
				switchGroup.Case(jen.Lit(cmd.Name)).BlockFunc(func(caseGroup *jen.Group) {
					appendHTTPCommandCase(caseGroup, cmd)
				})
			}
		})
	})
	group.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Nil(), jen.Err()),
	)
	group.Return(jen.Id("endpoint"), jen.Id("data"), jen.Nil())
}

func appendHTTPCommandCase(group *jen.Group, cmd *commandData) {
	group.Id("c").Op(":=").Id(cmd.PkgName).Dot("NewClient").Call(httpCommandClientArgs(cmd)...)
	group.Switch(jen.Id("epn")).BlockFunc(func(endpointSwitch *jen.Group) {
		for _, sub := range cmd.Subcommands {
			endpointSwitch.Case(jen.Lit(sub.Name)).BlockFunc(func(subGroup *jen.Group) {
				appendHTTPEndpointCase(subGroup, cmd, sub)
			})
		}
	})
}

func httpCommandClientArgs(cmd *commandData) []jen.Code {
	args := []jen.Code{
		jen.Id("scheme"),
		jen.Id("host"),
		jen.Id("doer"),
		jen.Id("enc"),
		jen.Id("dec"),
		jen.Id("restore"),
	}
	if cmd.NeedDialer {
		args = append(args, jen.Id("dialer"), jen.Id(cmd.VarName+"Configurer"))
	}
	return args
}

func appendHTTPEndpointCase(group *jen.Group, cmd *commandData, sub *subcommandData) {
	group.Id("endpoint").Op("=").Id("c").Dot(sub.MethodVarName).Call(httpEndpointCallArgs(sub)...)
	appendHTTPInterceptors(group, cmd, sub)
	appendHTTPBuildData(group, cmd, sub)
	appendHTTPBuildStreamPayload(group, cmd, sub)
}

func httpEndpointCallArgs(sub *subcommandData) []jen.Code {
	if sub.MultipartVarName == "" {
		return nil
	}
	return []jen.Code{jen.Id(sub.MultipartVarName)}
}

func appendHTTPInterceptors(group *jen.Group, cmd *commandData, sub *subcommandData) {
	if sub.Interceptors == nil {
		return
	}
	group.Id("endpoint").Op("=").Id(sub.Interceptors.PkgName).
		Dot("Wrap"+sub.MethodVarName+"ClientEndpoint").
		Call(jen.Id("endpoint"), jen.Id(cmd.Interceptors.VarName))
}

func appendHTTPBuildData(group *jen.Group, cmd *commandData, sub *subcommandData) {
	switch {
	case sub.BuildFunction != nil:
		group.List(jen.Id("data"), jen.Err()).Op("=").Id(cmd.PkgName).Dot(sub.BuildFunction.Name).Call(httpBuildFunctionArgs(sub)...)
	case sub.Conversion != nil:
		group.Add(sub.Conversion)
	}
}

func httpBuildFunctionArgs(sub *subcommandData) []jen.Code {
	args := make([]jen.Code, 0, len(sub.BuildFunction.ActualParams))
	for _, param := range sub.BuildFunction.ActualParams {
		args = append(args, jen.Op("*").Id(param+"Flag"))
	}
	return args
}

func appendHTTPBuildStreamPayload(group *jen.Group, cmd *commandData, sub *subcommandData) {
	if sub.StreamFlag == nil {
		return
	}
	call := jen.List(jen.Id("data"), jen.Err()).Op("=").Id(cmd.PkgName).Dot(sub.BuildStreamPayload).Call(httpStreamPayloadArgs(sub)...)
	if sub.BuildFunction != nil {
		group.If(jen.Err().Op("==").Nil()).Block(call)
		return
	}
	group.Add(call)
}

func httpStreamPayloadArgs(sub *subcommandData) []jen.Code {
	streamArgs := []jen.Code{}
	if sub.BuildFunction != nil || sub.Conversion != nil {
		streamArgs = append(streamArgs, jen.Id("data"))
	}
	streamArgs = append(streamArgs, jen.Op("*").Id(sub.StreamFlag.FullName+"Flag"))
	return streamArgs
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
