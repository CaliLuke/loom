package openapiv3

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
	"github.com/CaliLuke/loom/internal/securityreq"
)

func openAPIExampleGenerator(api *expr.APIExpr) *expr.ExampleGenerator {
	if api == nil || api.ExampleGenerator == nil {
		return nil
	}
	if value, ok := api.Meta.Last("openapi:example"); ok && value == "false" {
		return &expr.ExampleGenerator{}
	}
	return api.ExampleGenerator
}

func buildDocument(root *expr.RootExpr) *OpenAPI {
	doc := openapiir.BuildDocument(
		root.API,
		root.Types,
		root.ResultTypes,
		openapiir.WithExampleGenerator(openAPIExampleGenerator(root.API)),
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
