package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func parseEndpointSection(flagsCode string, commands []*commandData) codegen.Section {
	return codegen.NewRawSection("parse-endpoint", renderParseEndpoint(flagsCode, commands))
}

func renderParseEndpoint(flagsCode string, commands []*commandData) string {
	var b strings.Builder
	b.WriteString("\n// ParseEndpoint returns the endpoint and payload as specified on the command\n")
	b.WriteString("// line.\n")
	b.WriteString("func ParseEndpoint(\n")
	b.WriteString("\tscheme, host string,\n")
	b.WriteString("\tdoer loomhttp.Doer,\n")
	b.WriteString("\tenc func(*http.Request) loomhttp.Encoder,\n")
	b.WriteString("\tdec func(*http.Response) loomhttp.Decoder,\n")
	b.WriteString("\trestore bool,\n")
	if streamingCmdExists(commands) {
		b.WriteString("\tdialer loomhttp.Dialer,\n")
		for _, cmd := range commands {
			if cmd.NeedDialer {
				fmt.Fprintf(&b, "\t%sConfigurer *%s.ConnConfigurer,\n", cmd.VarName, cmd.PkgName)
			}
		}
	}
	for _, cmd := range commands {
		for _, sub := range cmd.Subcommands {
			if sub.MultipartVarName != "" {
				fmt.Fprintf(&b, "\t%s %s.%s,\n", sub.MultipartVarName, cmd.PkgName, sub.MultipartFuncName)
			}
		}
		if cmd.Interceptors != nil {
			fmt.Fprintf(&b, "\t%s %s.ClientInterceptors,\n", cmd.Interceptors.VarName, cmd.Interceptors.PkgName)
		}
	}
	b.WriteString(") (loom.Endpoint, any, error) {\n")
	b.WriteString(flagsCode)
	b.WriteString("\n\tvar (\n")
	b.WriteString("\t\tdata     any\n")
	b.WriteString("\t\tendpoint loom.Endpoint\n")
	b.WriteString("\t\terr      error\n")
	b.WriteString("\t)\n")
	b.WriteString("\t{\n")
	b.WriteString("\t\tswitch svcn {\n")
	for _, cmd := range commands {
		fmt.Fprintf(&b, "\t\tcase %q:\n", cmd.Name)
		fmt.Fprintf(&b, "\t\t\tc := %s.NewClient(scheme, host, doer, enc, dec, restore", cmd.PkgName)
		if cmd.NeedDialer {
			fmt.Fprintf(&b, ", dialer, %sConfigurer", cmd.VarName)
		}
		b.WriteString(")\n")
		b.WriteString("\t\t\tswitch epn {\n")
		for _, sub := range cmd.Subcommands {
			fmt.Fprintf(&b, "\t\t\tcase %q:\n", sub.Name)
			fmt.Fprintf(&b, "\t\t\t\tendpoint = c.%s(", sub.MethodVarName)
			if sub.MultipartVarName != "" {
				b.WriteString(sub.MultipartVarName)
			}
			b.WriteString(")\n")
			if sub.Interceptors != nil {
				fmt.Fprintf(&b, "\t\t\t\tendpoint = %s.Wrap%sClientEndpoint(endpoint, %s)\n", sub.Interceptors.PkgName, sub.MethodVarName, cmd.Interceptors.VarName)
			}
			if sub.BuildFunction != nil {
				fmt.Fprintf(&b, "\t\t\t\tdata, err = %s.%s(", cmd.PkgName, sub.BuildFunction.Name)
				for _, param := range sub.BuildFunction.ActualParams {
					fmt.Fprintf(&b, "*%sFlag, ", param)
				}
				b.WriteString(")\n")
			} else if sub.Conversion != "" {
				fmt.Fprintf(&b, "\t\t\t\t%s\n", sub.Conversion)
			}
			if sub.StreamFlag != nil {
				if sub.BuildFunction != nil {
					b.WriteString("\t\t\t\tif err == nil {\n")
				}
				fmt.Fprintf(&b, "\t\t\t\t\tdata, err = %s.%s(", cmd.PkgName, sub.BuildStreamPayload)
				if sub.BuildFunction != nil || sub.Conversion != "" {
					b.WriteString("data, ")
				}
				fmt.Fprintf(&b, "*%sFlag)\n", sub.StreamFlag.FullName)
				if sub.BuildFunction != nil {
					b.WriteString("\t\t\t\t}\n")
				}
			}
		}
		b.WriteString("\t\t\t}\n")
	}
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn nil, nil, err\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\treturn endpoint, data, nil\n")
	b.WriteString("}\n")
	return b.String()
}

func pathSection(data *EndpointData) codegen.Section {
	return codegen.NewRawSection("path", renderPathSection(data))
}

func renderPathSection(data *EndpointData) string {
	var b strings.Builder
	for _, route := range data.Routes {
		if route.PathInit == nil {
			continue
		}
		b.WriteString("// " + route.PathInit.Description + "\n")
		fmt.Fprintf(&b, "func %s(", route.PathInit.Name)
		for _, arg := range route.PathInit.ServerArgs {
			fmt.Fprintf(&b, "%s %s, ", arg.VarName, arg.TypeRef)
		}
		fmt.Fprintf(&b, ") %s {\n", route.PathInit.ReturnTypeRef)
		code := strings.TrimLeft(route.PathInit.ServerCode, "\n")
		b.WriteString(code)
		if !strings.HasSuffix(code, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("}\n")
	}
	return b.String()
}
