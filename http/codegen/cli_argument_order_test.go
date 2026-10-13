package codegen

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestCLIArgumentOrder(t *testing.T) {
	for _, shape := range []string{"plain", "single", "multipart", "interceptors", "combined", "split", "stream", "subset", "file-only"} {
		t.Run(shape, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.CLIArgumentOrderDSL("http", shape))
			services := CreateHTTPServices(root)
			var declaration *ast.FuncDecl
			for _, file := range ClientCLIFiles("example.com/order/gen", services) {
				for _, section := range file.Section("parse-endpoint") {
					parsed := parseCLIOrderSection(t, "package cli\n", section)
					for _, decl := range parsed.Decls {
						if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "ParseEndpoint" {
							declaration = fn
						}
					}
				}
			}
			require.NotNil(t, declaration)
			file := ExampleCLI("example.com/order/gen", root.API.Servers[0], services)
			require.NotNil(t, file)
			sections := file.Section("cli-http-end")
			require.Len(t, sections, 1)
			parsed := parseCLIOrderSection(t, "package main\nfunc example() {\n", sections[0])
			var call *ast.CallExpr
			ast.Inspect(parsed, func(node ast.Node) bool {
				candidate, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if selector, ok := candidate.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "ParseEndpoint" {
					call = candidate
				}
				return true
			})
			require.NotNil(t, call)
			params := make([]string, 0, len(declaration.Type.Params.List)-6)
			args := make([]string, 0, len(call.Args)-6)
			for _, field := range declaration.Type.Params.List[6:] {
				params = append(params, cliOrderParameterKey(t, field))
			}
			for _, arg := range call.Args[6:] {
				args = append(args, cliOrderArgumentKey(t, arg))
			}
			require.Equal(t, params, args, "example arguments must match the declared dependency identity at every position")
		})
	}
}

func TestCLIArgumentPlanPreservesSelectedIdentity(t *testing.T) {
	root := RunHTTPDSL(t, testdata.CLIArgumentOrderDSL("http", "stream"))
	services := CreateHTTPServices(root)
	server := root.API.Servers[0]
	server.Services = []string{"beta", "alpha"}
	plan := buildCLIArgumentPlan(server, services)
	alpha, beta := services.Get("alpha"), services.Get("beta")
	require.Equal(t, []*ServiceData{beta, alpha}, plan.services)
	require.Equal(t, []cliArgument{
		{kind: cliDialer},
		{kind: cliConfigurer, service: beta},
		{kind: cliConfigurer, service: alpha},
		{kind: cliMultipartEncoder, service: beta, endpoint: beta.Endpoint("up")},
		{kind: cliInterceptors, service: beta},
		{kind: cliMultipartEncoder, service: alpha, endpoint: alpha.Endpoint("up")},
		{kind: cliInterceptors, service: alpha},
	}, plan.arguments)
	transport := httpClientCLITransport()
	commands := []*commandData{buildCommandData(beta, transport), buildCommandData(alpha, transport)}
	commands = allocateHTTPCommandIdentifiers(commands, []*codegen.ImportSpec{{Path: "example.com/betac", Name: "betac"}}, transport)
	require.NotEqual(t, "betac", commands[0].PkgName)
	section := codegen.NewJenniferSection("parameters", func(stmt *jen.Statement) {
		stmt.Func().Id("parameters").ParamsFunc(func(group *jen.Group) {
			appendCLIArgumentParameters(group, plan, commands, transport)
		}).Block()
	})
	parsed := parseCLIOrderSection(t, "package cli\n", section)
	fields := parsed.Decls[0].(*ast.FuncDecl).Type.Params.List
	for i, command := range commands {
		require.Same(t, plan.services[i], command.source)
		field := fields[i+1]
		require.Equal(t, transport.StreamingConfigurerName(command.VarName), field.Names[0].Name)
		pointer, ok := field.Type.(*ast.StarExpr)
		require.True(t, ok)
		selector := pointer.X.(*ast.SelectorExpr)
		require.Equal(t, command.PkgName, selector.X.(*ast.Ident).Name)
		require.Equal(t, "ConnConfigurer", selector.Sel.Name)
	}
	require.Equal(t, commands[0].PkgName, fields[3].Type.(*ast.SelectorExpr).X.(*ast.Ident).Name)
	require.Equal(t, commands[1].PkgName, fields[5].Type.(*ast.SelectorExpr).X.(*ast.Ident).Name)
	require.Same(t, beta.Endpoint("up"), commands[0].Subcommands[0].source)
	require.Same(t, alpha.Endpoint("up"), commands[1].Subcommands[0].source)
}

func TestCLIArgumentPlanExcludesEndpointFreeServices(t *testing.T) {
	root := RunHTTPDSL(t, testdata.CLIArgumentOrderDSL("http", "file-only"))
	services := CreateHTTPServices(root)
	alpha := services.Get("alpha")
	require.Empty(t, alpha.Endpoints)
	require.NotEmpty(t, alpha.Service.ClientInterceptors)
	root.API.Servers[0].Services = []string{"alpha"}
	plan := buildCLIArgumentPlan(root.API.Servers[0], services)
	require.Empty(t, plan.services)
	require.Empty(t, plan.arguments)
	require.False(t, plan.needsDialer())
	require.Empty(t, renderExampleCLIStreaming(plan))
	require.NotContains(t, renderExampleCLIEnd(plan, "api", "nil"), "Interceptors")
	root.API.Servers[0].Services = []string{"alpha", "beta"}
	plan = buildCLIArgumentPlan(root.API.Servers[0], services)
	require.Equal(t, []*ServiceData{services.Get("beta")}, plan.services)
	for _, argument := range plan.arguments {
		require.NotSame(t, alpha, argument.service)
	}
}

func parseCLIOrderSection(t *testing.T, prefix string, section codegen.Section) *ast.File {
	t.Helper()
	var output bytes.Buffer
	output.WriteString(prefix)
	require.NoError(t, section.Write(&output))
	parsed, err := parser.ParseFile(token.NewFileSet(), "cli.go", output.Bytes(), 0)
	require.NoError(t, err)
	return parsed
}

func cliOrderParameterKey(t *testing.T, field *ast.Field) string {
	t.Helper()
	if _, ok := field.Type.(*ast.StarExpr); ok {
		return "configurer"
	}
	selector, ok := field.Type.(*ast.SelectorExpr)
	require.True(t, ok, "unexpected parser parameter type %T", field.Type)
	if strings.HasSuffix(selector.Sel.Name, "EncoderFunc") {
		return "multipart:" + selector.Sel.Name
	}
	if selector.Sel.Name == "ClientInterceptors" {
		return "interceptors:" + selector.X.(*ast.Ident).Name
	}
	return field.Names[0].Name
}

func cliOrderArgumentKey(t *testing.T, arg ast.Expr) string {
	t.Helper()
	switch arg := arg.(type) {
	case *ast.SelectorExpr:
		return "multipart:" + arg.Sel.Name
	case *ast.Ident:
		if arg.Name == "nil" {
			return "configurer"
		}
		if strings.HasSuffix(arg.Name, "Interceptors") {
			return "interceptors:" + strings.TrimSuffix(arg.Name, "Interceptors")
		}
		return arg.Name
	default:
		t.Fatalf("unexpected example dependency %T", arg)
		return ""
	}
}
