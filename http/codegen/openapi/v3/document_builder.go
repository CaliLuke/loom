package openapiv3

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
	"github.com/CaliLuke/loom/internal/securityreq"
)

func disableOpenAPIExamples(api *expr.APIExpr) {
	if api == nil {
		return
	}
	m, ok := api.Meta.Last("openapi:example")
	if ok && m == "false" {
		api.ExampleGenerator.Randomizer = nil
	}
}

func buildDocument(root *expr.RootExpr) *OpenAPI {
	doc := openapiir.BuildDocument(
		root.API,
		root.Types,
		root.ResultTypes,
		openapiir.WithExampleValue(openAPIExampleValue),
		openapiir.WithExampleSuppression(shouldSuppressOpenAPIExamples),
	)
	paths := buildPaths(root.API.HTTP, doc, root.API)
	reusable := reusableComponentsFromIR(doc.Components)
	schemas := openapiir.RenderSchemaMap(doc.Components.Schemas)
	cleanupComponentSchemas(paths, schemas, reusable)

	return &OpenAPI{
		OpenAPI:           OpenAPIVersion,
		Info:              buildInfo(root.API),
		JSONSchemaDialect: JSONSchemaDialect,
		Components:        buildComponents(root, pruneUnusedComponentSchemas(paths, schemas, reusable), reusable),
		Paths:             paths,
		Servers:           buildServers(root.API.Servers),
		Security:          securityreq.OpenAPI(securityreq.Effective(root.API.Requirements, root.API.SessionAuths)),
		Tags:              buildTags(root.API),
		ExternalDocs:      openapi.DocsFromExpr(root.API.Docs, nil),
	}
}
