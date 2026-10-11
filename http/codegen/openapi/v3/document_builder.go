package openapiv3

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
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

func buildDocument(root *expr.RootExpr) (*OpenAPI, error) {
	doc, err := openapiir.BuildDocument(
		root.API,
		root.Types,
		root.ResultTypes,
		openapiir.WithExampleGenerator(openAPIExampleGenerator(root.API)),
		openapiir.WithExampleSuppression(shouldSuppressOpenAPIExamples),
	)
	if err != nil {
		return nil, err
	}
	paths, err := buildPaths(root.API.HTTP, doc, root.API)
	if err != nil {
		return nil, err
	}
	if err := validateOpenAPIIdentities(paths); err != nil {
		return nil, err
	}
	reusable := reusableComponentsFromIR(doc.Components)
	schemas := openapiir.RenderSchemaMap(doc.Components.Schemas)
	cleanupComponentSchemas(paths, schemas, reusable)

	return &OpenAPI{
		OpenAPI:           OpenAPIVersion,
		Info:              buildInfo(root.API),
		JSONSchemaDialect: JSONSchemaDialect,
		Components:        buildComponents(doc.Components.SecuritySchemes, pruneUnusedComponentSchemas(paths, schemas, reusable), reusable),
		Paths:             paths,
		Servers:           buildServers(root.API.Servers),
		Security:          cloneOperationSecurity(doc.Security),
		Tags:              buildTags(root.API),
		ExternalDocs:      openapi.DocsFromExpr(root.API.Docs),
		Extensions:        openapi.ScopedExtensionsFromExpr(root.API.Meta, "document"),
	}, nil
}

func validateOpenAPIIdentities(paths map[string]*PathItem) error {
	var claims []openapiir.OperationIdentity
	for path, item := range paths {
		if item == nil {
			continue
		}
		for method, operation := range pathItemOperationEntries(item) {
			if operation == nil {
				continue
			}
			claims = append(claims, openapiir.OperationIdentity{
				Path:        path,
				Method:      method,
				OperationID: operation.OperationID,
			})
		}
	}
	return openapiir.ValidateOperationIdentities(claims)
}

func pathItemOperationEntries(item *PathItem) map[string]*Operation {
	operations := map[string]*Operation{
		"GET": item.Get, "PUT": item.Put, "POST": item.Post, "DELETE": item.Delete,
		"OPTIONS": item.Options, "HEAD": item.Head, "PATCH": item.Patch,
		"TRACE": item.Trace, "QUERY": item.Query,
	}
	for method, operation := range item.AdditionalOperations {
		operations[method] = operation
	}
	return operations
}
