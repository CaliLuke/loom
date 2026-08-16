package codegen

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// ResponseContractTestFiles returns consumer-owned JSON-RPC response contract
// test scaffolds. Existing files are never overwritten.
func ResponseContractTestFiles(genpkg string, data *httpcodegen.ServicesData) []*codegen.File {
	var files []*codegen.File
	for _, service := range data.Root.API.JSONRPC.Services {
		contractData := buildResponseContractServiceData(service, data)
		if !contractData.hasCases() {
			continue
		}
		files = append(files, responseContractTestFile(genpkg, contractData))
	}
	return files
}

func responseContractTestFile(genpkg string, data *responseContractServiceData) *codegen.File {
	scope := codegen.NewNameScope()
	serverAlias := scope.Unique(data.Service.Service.PkgName+"jssvr", "jssvr")
	imports := []*codegen.ImportSpec{
		{Path: "testing"},
		codegen.LoomNamedImport("jsonrpc", "loomjsonrpc"),
		{
			Path: path.Join(genpkg, "jsonrpc", data.Service.Service.PathName, "server"),
			Name: serverAlias,
		},
	}
	return &codegen.File{
		Path: filepath.Join("internal", "contracttest", data.Service.Service.PathName+"_jsonrpc_test.go"),
		Sections: []codegen.Section{
			codegen.ScaffoldHeader(fmt.Sprintf("%s JSON-RPC response contract tests", data.Service.Service.Name), "contracttest", imports),
			responseContractTestSection(data, serverAlias),
		},
		SkipExist: true,
	}
}

func responseContractTestSection(data *responseContractServiceData, serverAlias string) codegen.Section {
	prefix := data.Service.Service.PkgName
	publicPrefix := codegen.Goify(prefix, true)
	scenarioType := prefix + "JSONRPCResponseContractScenario"
	scenariosInit := prefix + "JSONRPCResponseContractScenarios"
	testName := "Test" + publicPrefix + "JSONRPCResponseContracts"

	return codegen.NewJenniferSection("response-contract-test", func(stmt *jen.Statement) {
		stmt.Type().Id(scenarioType).Func().Params(
			jen.Op("*").Qual("testing", "T"),
			codegen.TypeRef("loomjsonrpc.ResponseContractCase"),
		).Op("*").Add(codegen.TypeRef("loomjsonrpc.ResponseContractObservation"))
		stmt.Line()
		stmt.Func().Id(scenariosInit).Params().Map(jen.String()).Id(scenarioType).Block(
			jen.Return(jen.Map(jen.String()).Id(scenarioType).Values()),
		)
		stmt.Line()
		stmt.Func().Id(testName).Params(jen.Id("t").Op("*").Qual("testing", "T")).Block(
			jen.Id("scenarios").Op(":=").Id(scenariosInit).Call(),
			jen.For(jen.Id("id").Op(":=").Range().Id("scenarios")).Block(
				jen.Id("matched").Op(":=").False(),
				jen.For(jen.List(jen.Id("_"), jen.Id("contract")).Op(":=").Range().Add(codegen.Expr(serverAlias+".ResponseContractCases")).Call()).Block(
					jen.If(jen.Id("contract").Dot("ID").Op("==").Id("id")).Block(
						jen.Id("matched").Op("=").True(),
						jen.Break(),
					),
				),
				jen.If(jen.Op("!").Id("matched")).Block(
					jen.Id("t").Dot("Errorf").Call(jen.Lit("JSON-RPC response contract scenario %q has no declared contract"), jen.Id("id")),
				),
			),
			jen.For(jen.List(jen.Id("_"), jen.Id("contract")).Op(":=").Range().Add(codegen.Expr(serverAlias+".ResponseContractCases")).Call()).Block(
				jen.List(jen.Id("scenario"), jen.Id("ok")).Op(":=").Id("scenarios").Index(jen.Id("contract").Dot("ID")),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Id("t").Dot("Errorf").Call(jen.Lit("missing JSON-RPC response contract scenario %q"), jen.Id("contract").Dot("ID")),
					jen.Continue(),
				),
				jen.Id("t").Dot("Run").Call(
					jen.Id("contract").Dot("ID"),
					jen.Func().Params(jen.Id("t").Op("*").Qual("testing", "T")).Block(
						jen.Id("observation").Op(":=").Id("scenario").Call(jen.Id("t"), jen.Id("contract")),
						jen.If(
							jen.Id("err").Op(":=").Add(codegen.Expr("loomjsonrpc.ValidateResponseContract")).Call(jen.Id("observation"), jen.Id("contract")),
							jen.Id("err").Op("!=").Nil(),
						).Block(jen.Id("t").Dot("Error").Call(jen.Id("err"))),
					),
				),
			),
		)
		stmt.Line()
	})
}
