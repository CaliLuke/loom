package codegen

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

// ResponseContractTestFiles returns consumer-owned HTTP response contract test
// scaffolds. Existing files are never overwritten.
func ResponseContractTestFiles(genpkg string, data *ServicesData) []*codegen.File {
	files := make([]*codegen.File, 0, len(data.HTTPData))
	for _, service := range data.Expressions.Services {
		serviceData := data.Get(service.Name())
		if !serviceHasResponseContractCases(serviceData) {
			continue
		}
		files = append(files, responseContractTestFile(genpkg, serviceData))
	}
	return files
}

func serviceHasResponseContractCases(data *ServiceData) bool {
	for _, endpoint := range data.Endpoints {
		if len(endpoint.ResponseContractCases) > 0 {
			return true
		}
	}
	return false
}

func responseContractTestFile(genpkg string, data *ServiceData) *codegen.File {
	serverAlias := data.Service.PkgName + "svr"
	imports := []*codegen.ImportSpec{
		{Path: "net/http"},
		{Path: "testing"},
		codegen.LoomNamedImport("http", "loomhttp"),
		{
			Path: path.Join(genpkg, "http", data.Service.PathName, "server"),
			Name: serverAlias,
		},
	}
	return &codegen.File{
		Path: filepath.Join("internal", "contracttest", data.Service.PathName+"_http_test.go"),
		Sections: []codegen.Section{
			codegen.Header(fmt.Sprintf("%s HTTP response contract tests", data.Service.Name), "contracttest", imports),
			responseContractTestSection(data, serverAlias),
		},
		SkipExist: true,
	}
}

func responseContractTestSection(data *ServiceData, serverAlias string) codegen.Section {
	prefix := data.Service.PkgName
	publicPrefix := codegen.Goify(prefix, true)
	scenarioType := prefix + "ResponseContractScenario"
	scenariosInit := prefix + "ResponseContractScenarios"
	testName := "Test" + publicPrefix + "HTTPResponseContracts"

	return codegen.NewJenniferSection("response-contract-test", func(stmt *jen.Statement) {
		stmt.Type().Id(scenarioType).Func().Params(
			jen.Op("*").Qual("testing", "T"),
		).Op("*").Qual("net/http", "Response")
		stmt.Line()

		stmt.Func().Id(scenariosInit).Params().Map(jen.String()).Id(scenarioType).Block(
			jen.Return(jen.Map(jen.String()).Id(scenarioType).Values()),
		)
		stmt.Line()

		stmt.Func().Id(testName).Params(jen.Id("t").Op("*").Qual("testing", "T")).BlockFunc(func(group *jen.Group) {
			group.Id("scenarios").Op(":=").Id(scenariosInit).Call()
			group.For(
				jen.List(jen.Id("_"), jen.Id("contract")).Op(":=").Range().Add(codegen.Expr(serverAlias+".ResponseContractCases")).Call(),
			).Block(
				jen.List(jen.Id("scenario"), jen.Id("ok")).Op(":=").Id("scenarios").Index(jen.Id("contract").Dot("ID")),
				jen.If(jen.Op("!").Id("ok")).Block(
					jen.Id("t").Dot("Errorf").Call(jen.Lit("missing response contract scenario %q"), jen.Id("contract").Dot("ID")),
					jen.Continue(),
				),
				jen.Id("t").Dot("Run").Call(
					jen.Id("contract").Dot("ID"),
					jen.Func().Params(jen.Id("t").Op("*").Qual("testing", "T")).Block(
						jen.Id("response").Op(":=").Id("scenario").Call(jen.Id("t")),
						jen.If(
							jen.Id("err").Op(":=").Add(codegen.Expr("loomhttp.ValidateResponseContract")).Call(
								jen.Id("response"),
								jen.Id("contract"),
							),
							jen.Id("err").Op("!=").Nil(),
						).Block(
							jen.Id("t").Dot("Error").Call(jen.Id("err")),
						),
					),
				),
			)
		})
		stmt.Line()
	})
}
