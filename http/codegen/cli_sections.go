package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func parseEndpointSection(flagsCode string, commands []*commandData) codegen.Section {
	return codegen.MustRenderSection("parse-endpoint", func() string {
		return renderParseEndpoint(flagsCode, commands)
	})
}

func renderParseEndpoint(flagsCode string, commands []*commandData) string {
	var b sourceBuilder
	writeParseEndpointHeader(&b, commands)
	b.Add(") (loom.Endpoint, any, error) {\n")
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
	b.Add("}\n")
	return b.String()
}

func writeParseEndpointHeader(b *sourceBuilder, commands []*commandData) {
	b.Add("\n// ParseEndpoint returns the endpoint and payload as specified on the command\n")
	b.Add("// line.\n")
	b.Add("func ParseEndpoint(\n")
	b.Add("\tscheme, host string,\n")
	b.Add("\tdoer loomhttp.Doer,\n")
	b.Add("\tenc func(*http.Request) loomhttp.Encoder,\n")
	b.Add("\tdec func(*http.Response) loomhttp.Decoder,\n")
	b.Add("\trestore bool,\n")
	writeParseEndpointStreamingParams(b, commands)
	writeParseEndpointCommandParams(b, commands)
}

func writeParseEndpointStreamingParams(b *sourceBuilder, commands []*commandData) {
	if !streamingCmdExists(commands) {
		return
	}
	b.Add("\tdialer loomhttp.Dialer,\n")
	for _, cmd := range commands {
		if cmd.NeedDialer {
			b.Addf("\t%sConfigurer *%s.ConnConfigurer,\n", cmd.VarName, cmd.PkgName)
		}
	}
}

func writeParseEndpointCommandParams(b *sourceBuilder, commands []*commandData) {
	for _, cmd := range commands {
		for _, sub := range cmd.Subcommands {
			if sub.MultipartVarName != "" {
				b.Addf("\t%s %s.%s,\n", sub.MultipartVarName, cmd.PkgName, sub.MultipartFuncName)
			}
		}
		if cmd.Interceptors != nil {
			b.Addf("\t%s %s.ClientInterceptors,\n", cmd.Interceptors.VarName, cmd.Interceptors.PkgName)
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
	return codegen.MustRenderSection("path", func() string {
		return renderPathSection(data)
	})
}

func renderPathSection(data *EndpointData) string {
	var b sourceBuilder
	for _, route := range data.Routes {
		if route.PathInit == nil {
			continue
		}
		b.Add("// " + route.PathInit.Description + "\n")
		b.Addf("func %s(", route.PathInit.Name)
		for _, arg := range route.PathInit.ServerArgs {
			b.Addf("%s %s, ", arg.VarName, arg.TypeRef)
		}
		b.Addf(") %s {\n", route.PathInit.ReturnTypeRef)
		code := strings.TrimLeft(route.PathInit.ServerCode, "\n")
		b.Add(code)
		if !strings.HasSuffix(code, "\n") {
			b.Add("\n")
		}
		b.Add("}\n")
	}
	return b.String()
}
