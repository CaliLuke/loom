package codegen

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

type responseContractFieldExecution struct {
	transportRef  string
	contractField string
	scenarios     string
	result        string
	missingFormat string
	validatorRef  string
}

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

func serviceHasSSEResponseContractCases(data *ServiceData) bool {
	for _, endpoint := range data.Endpoints {
		for _, contractCase := range endpoint.ResponseContractCases {
			if contractCase.SSE != nil {
				return true
			}
		}
	}
	return false
}

func serviceHasWebSocketResponseContractCases(data *ServiceData) bool {
	for _, endpoint := range data.Endpoints {
		for _, contractCase := range endpoint.ResponseContractCases {
			if contractCase.WebSocket != nil {
				return true
			}
		}
	}
	return false
}

func serviceHasMultipartResponseContractCases(data *ServiceData) bool {
	for _, endpoint := range data.Endpoints {
		for _, contractCase := range endpoint.ResponseContractCases {
			if contractCase.Multipart != nil {
				return true
			}
		}
	}
	return false
}

func responseContractTestFile(genpkg string, data *ServiceData) *codegen.File {
	imports := make([]*codegen.ImportSpec, 0, 4)
	imports = append(imports,
		&codegen.ImportSpec{Path: "net/http"},
		&codegen.ImportSpec{Path: "testing"},
		codegen.LoomNamedImport("http", "loomhttp"),
	)
	scope := codegen.NewNameScope()
	reserveExampleImportNames(scope, imports)
	serverAlias := scope.Unique(data.Service.PkgName+"svr", "svr")
	imports = append(imports, &codegen.ImportSpec{
		Path: path.Join(genpkg, "http", data.Service.PathName, "server"),
		Name: serverAlias,
	})
	return &codegen.File{
		Path: filepath.Join("internal", "contracttest", data.Service.PathName+"_http_test.go"),
		Sections: []codegen.Section{
			codegen.ScaffoldHeader(fmt.Sprintf("%s HTTP response contract tests", data.Service.Name), "contracttest", imports),
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
	hasSSE := serviceHasSSEResponseContractCases(data)
	sseScenarioType := prefix + "SSEResponseContractScenario"
	sseScenariosInit := prefix + "SSEResponseContractScenarios"
	hasWebSocket := serviceHasWebSocketResponseContractCases(data)
	webSocketScenarioType := prefix + "WebSocketResponseContractScenario"
	webSocketScenariosInit := prefix + "WebSocketResponseContractScenarios"
	hasMultipart := serviceHasMultipartResponseContractCases(data)
	multipartScenarioType := prefix + "MultipartResponseContractScenario"
	multipartScenariosInit := prefix + "MultipartResponseContractScenarios"

	return codegen.NewJenniferSection("response-contract-test", func(stmt *jen.Statement) {
		addResponseContractScenarioDeclarations(
			stmt,
			scenarioType,
			scenariosInit,
			sseScenarioType,
			sseScenariosInit,
			hasSSE,
			webSocketScenarioType,
			webSocketScenariosInit,
			hasWebSocket,
			multipartScenarioType,
			multipartScenariosInit,
			hasMultipart,
		)
		stmt.Func().Id(testName).Params(jen.Id("t").Op("*").Qual("testing", "T")).BlockFunc(func(group *jen.Group) {
			group.Id("scenarios").Op(":=").Id(scenariosInit).Call()
			if hasSSE {
				group.Id("sseScenarios").Op(":=").Id(sseScenariosInit).Call()
			}
			if hasWebSocket {
				group.Id("webSocketScenarios").Op(":=").Id(webSocketScenariosInit).Call()
			}
			if hasMultipart {
				group.Id("multipartScenarios").Op(":=").Id(multipartScenariosInit).Call()
			}
			group.Add(responseContractManifestCheck(
				"scenarios",
				serverAlias,
				"unary",
				"response contract scenario %q has no declared contract",
			))
			if hasSSE {
				group.Add(responseContractManifestCheck(
					"sseScenarios",
					serverAlias,
					"sse",
					"SSE response contract scenario %q has no declared contract",
				))
			}
			if hasWebSocket {
				group.Add(responseContractManifestCheck(
					"webSocketScenarios",
					serverAlias,
					"websocket",
					"WebSocket response contract scenario %q has no declared contract",
				))
			}
			if hasMultipart {
				group.Add(responseContractManifestCheck(
					"multipartScenarios",
					serverAlias,
					"multipart",
					"multipart response contract scenario %q has no declared contract",
				))
			}
			group.Add(responseContractExecutionLoop(serverAlias, hasSSE, hasWebSocket, hasMultipart))
		})
		stmt.Line()
	})
}

func addResponseContractScenarioDeclarations(
	stmt *jen.Statement,
	scenarioType, scenariosInit, sseScenarioType, sseScenariosInit string,
	hasSSE bool,
	webSocketScenarioType, webSocketScenariosInit string,
	hasWebSocket bool,
	multipartScenarioType, multipartScenariosInit string,
	hasMultipart bool,
) {
	stmt.Type().Id(scenarioType).Func().Params(
		jen.Op("*").Qual("testing", "T"),
	).Op("*").Qual("net/http", "Response")
	stmt.Line()
	stmt.Func().Id(scenariosInit).Params().Map(jen.String()).Id(scenarioType).Block(
		jen.Return(jen.Map(jen.String()).Id(scenarioType).Values()),
	)
	stmt.Line()
	if hasSSE {
		stmt.Type().Id(sseScenarioType).Func().Params(
			jen.Op("*").Qual("testing", "T"),
		).Op("*").Add(codegen.Expr("loomhttp.SSEResponseContractObservation"))
		stmt.Line()
		stmt.Func().Id(sseScenariosInit).Params().Map(jen.String()).Id(sseScenarioType).Block(
			jen.Return(jen.Map(jen.String()).Id(sseScenarioType).Values()),
		)
		stmt.Line()
	}
	if hasWebSocket {
		stmt.Type().Id(webSocketScenarioType).Func().Params(
			jen.Op("*").Qual("testing", "T"),
			codegen.TypeRef("loomhttp.WebSocketResponseContract"),
		).Op("*").Add(codegen.TypeRef("loomhttp.WebSocketResponseContractObservation"))
		stmt.Line()
		stmt.Func().Id(webSocketScenariosInit).Params().Map(jen.String()).Id(webSocketScenarioType).Block(
			jen.Return(jen.Map(jen.String()).Id(webSocketScenarioType).Values()),
		)
		stmt.Line()
	}
	if hasMultipart {
		stmt.Type().Id(multipartScenarioType).Func().Params(
			jen.Op("*").Qual("testing", "T"),
			codegen.TypeRef("loomhttp.MultipartRequestContract"),
		).Op("*").Qual("net/http", "Response")
		stmt.Line()
		stmt.Func().Id(multipartScenariosInit).Params().Map(jen.String()).Id(multipartScenarioType).Block(
			jen.Return(jen.Map(jen.String()).Id(multipartScenarioType).Values()),
		)
		stmt.Line()
	}
}

func responseContractManifestCheck(scenarios, serverAlias, scenarioKind, errorFormat string) *jen.Statement {
	return jen.For(
		jen.List(jen.Id("id")).Op(":=").Range().Id(scenarios),
	).Block(
		jen.Id("matched").Op(":=").False(),
		jen.For(
			jen.List(jen.Id("_"), jen.Id("contract")).Op(":=").Range().Add(codegen.Expr(serverAlias+".ResponseContractCases")).Call(),
		).Block(
			jen.If(
				responseContractScenarioPredicate(scenarioKind).Op("&&").
					Id("contract").Dot("ID").Op("==").Id("id"),
			).Block(
				jen.Id("matched").Op("=").True(),
				jen.Break(),
			),
		),
		jen.If(jen.Op("!").Id("matched")).Block(
			jen.Id("t").Dot("Errorf").Call(jen.Lit(errorFormat), jen.Id("id")),
		),
	)
}

func responseContractScenarioPredicate(scenarioKind string) *jen.Statement {
	switch scenarioKind {
	case "sse":
		return jen.Id("contract").Dot("Transport").Op("==").Add(codegen.Expr("loomhttp.ResponseContractSSE")).Op("&&").
			Id("contract").Dot("SSE").Op("!=").Nil()
	case "websocket":
		return jen.Id("contract").Dot("Transport").Op("==").Add(codegen.Expr("loomhttp.ResponseContractWebSocket")).Op("&&").
			Id("contract").Dot("WebSocket").Op("!=").Nil()
	case "multipart":
		return jen.Id("contract").Dot("Transport").Op("==").Add(codegen.Expr("loomhttp.ResponseContractHTTP")).Op("&&").
			Id("contract").Dot("Multipart").Op("!=").Nil()
	default:
		return jen.Id("contract").Dot("Transport").Op("==").Add(codegen.Expr("loomhttp.ResponseContractHTTP")).Op("&&").
			Id("contract").Dot("Multipart").Op("==").Nil()
	}
}

func responseContractExecutionLoop(serverAlias string, hasSSE, hasWebSocket, hasMultipart bool) *jen.Statement {
	return jen.For(
		jen.List(jen.Id("_"), jen.Id("contract")).Op(":=").Range().Add(codegen.Expr(serverAlias + ".ResponseContractCases")).Call(),
	).BlockFunc(func(group *jen.Group) {
		if hasSSE {
			addSSEResponseContractExecution(group)
		}
		if hasWebSocket {
			addResponseContractFieldExecution(group, responseContractFieldExecution{
				transportRef:  "loomhttp.ResponseContractWebSocket",
				contractField: "WebSocket",
				scenarios:     "webSocketScenarios",
				result:        "observation",
				missingFormat: "missing WebSocket response contract scenario %q",
				validatorRef:  "loomhttp.ValidateWebSocketResponseContract",
			})
		}
		if hasMultipart {
			addResponseContractFieldExecution(group, responseContractFieldExecution{
				transportRef:  "loomhttp.ResponseContractHTTP",
				contractField: "Multipart",
				scenarios:     "multipartScenarios",
				result:        "response",
				missingFormat: "missing multipart response contract scenario %q",
				validatorRef:  "loomhttp.ValidateResponseContract",
			})
		}
		addUnaryResponseContractExecution(group)
	})
}

func addResponseContractFieldExecution(group *jen.Group, execution responseContractFieldExecution) {
	group.If(
		jen.Id("contract").Dot("Transport").Op("==").Add(codegen.Expr(execution.transportRef)).Op("&&").
			Id("contract").Dot(execution.contractField).Op("!=").Nil(),
	).Block(
		jen.List(jen.Id("scenario"), jen.Id("ok")).Op(":=").Id(execution.scenarios).Index(jen.Id("contract").Dot("ID")),
		jen.If(jen.Op("!").Id("ok")).Block(
			jen.Id("t").Dot("Errorf").Call(jen.Lit(execution.missingFormat), jen.Id("contract").Dot("ID")),
			jen.Continue(),
		),
		jen.Id("t").Dot("Run").Call(
			jen.Id("contract").Dot("ID"),
			jen.Func().Params(jen.Id("t").Op("*").Qual("testing", "T")).Block(
				jen.Id(execution.result).Op(":=").Id("scenario").Call(
					jen.Id("t"),
					jen.Op("*").Id("contract").Dot(execution.contractField),
				),
				jen.If(
					jen.Id("err").Op(":=").Add(codegen.Expr(execution.validatorRef)).Call(
						jen.Id(execution.result),
						jen.Id("contract"),
					),
					jen.Id("err").Op("!=").Nil(),
				).Block(
					jen.Id("t").Dot("Error").Call(jen.Id("err")),
				),
			),
		),
		jen.Continue(),
	)
}

func addSSEResponseContractExecution(group *jen.Group) {
	group.If(
		jen.Id("contract").Dot("Transport").Op("==").Add(codegen.Expr("loomhttp.ResponseContractSSE")).Op("&&").
			Id("contract").Dot("SSE").Op("!=").Nil(),
	).Block(
		jen.List(jen.Id("scenario"), jen.Id("ok")).Op(":=").Id("sseScenarios").Index(jen.Id("contract").Dot("ID")),
		jen.If(jen.Op("!").Id("ok")).Block(
			jen.Id("t").Dot("Errorf").Call(jen.Lit("missing SSE response contract scenario %q"), jen.Id("contract").Dot("ID")),
			jen.Continue(),
		),
		jen.Id("t").Dot("Run").Call(
			jen.Id("contract").Dot("ID"),
			jen.Func().Params(jen.Id("t").Op("*").Qual("testing", "T")).Block(
				jen.Id("observation").Op(":=").Id("scenario").Call(jen.Id("t")),
				jen.If(
					jen.Id("err").Op(":=").Add(codegen.Expr("loomhttp.ValidateSSEResponseContract")).Call(
						jen.Id("observation"),
						jen.Id("contract"),
					),
					jen.Id("err").Op("!=").Nil(),
				).Block(
					jen.Id("t").Dot("Error").Call(jen.Id("err")),
				),
			),
		),
		jen.Continue(),
	)
}

func addUnaryResponseContractExecution(group *jen.Group) {
	group.List(jen.Id("scenario"), jen.Id("ok")).Op(":=").Id("scenarios").Index(jen.Id("contract").Dot("ID"))
	group.If(jen.Op("!").Id("ok")).Block(
		jen.Id("t").Dot("Errorf").Call(jen.Lit("missing response contract scenario %q"), jen.Id("contract").Dot("ID")),
		jen.Continue(),
	)
	group.Id("t").Dot("Run").Call(
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
	)
}
