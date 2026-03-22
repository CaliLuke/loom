package openapiv3

import (
	"fmt"
	"hash"
	"maps"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
)

type (
	// EndpointBodies describes the request and response HTTP bodies of an endpoint
	// using JSON schema. Each body may be described via a reference to a schema
	// described in the "Components" section of the OpenAPI document or an actual
	// JSON schema data structure. There may also be additional notes attached to
	// each body definition to account for cases that are not directly supported in
	// OpenAPI such as streaming. The possible response bodies are indexed by HTTP
	// status, there may be more than one when the result type defined multiple
	// views.
	EndpointBodies struct {
		RequestBody    *openapi.Schema
		ResponseBodies map[int][]*openapi.Schema
	}

	// schemafier adapts the OpenAPI IR analyzer/renderer to the legacy v3 schema
	// helper API used throughout this package.
	schemafier struct {
		analyzer     *openapiir.Analyzer
		schemas      map[string]*openapi.Schema
		schemaHashes map[string]uint64
	}
)

// newSchemafier initializes a schemafier.
func newSchemafier(rand *expr.ExampleGenerator) *schemafier {
	return &schemafier{
		analyzer: openapiir.NewAnalyzer(
			rand,
			false,
			openapiir.WithExampleValue(openAPIExampleValue),
			openapiir.WithExampleSuppression(shouldSuppressOpenAPIExamples),
		),
		schemas:      make(map[string]*openapi.Schema),
		schemaHashes: make(map[string]uint64),
	}
}

// buildBodyTypes traverses the design and builds the JSON schemas that
// represent the request and response bodies of each endpoint. The result is a
// map of method details indexed by service name plus rendered component
// schemas indexed by type name.
//
// NOTE: entries are nil when the corresponding type is Empty.
func buildBodyTypes(api *expr.APIExpr, types []expr.UserType, resultTypes []*expr.ResultTypeExpr) (map[string]map[string]*EndpointBodies, map[string]*openapi.Schema) {
	bodyTypes := openapiir.BuildBodyTypes(
		api,
		types,
		resultTypes,
		openapiir.WithExampleValue(openAPIExampleValue),
		openapiir.WithExampleSuppression(shouldSuppressOpenAPIExamples),
	)
	renderedServices, renderedComponents := openapiir.RenderBodyTypes(bodyTypes)
	services := make(map[string]map[string]*EndpointBodies, len(renderedServices))
	for serviceName, methods := range renderedServices {
		renderedMethods := make(map[string]*EndpointBodies, len(methods))
		for methodName, bodies := range methods {
			renderedMethods[methodName] = &EndpointBodies{
				RequestBody:    bodies.RequestBody,
				ResponseBodies: bodies.ResponseBodies,
			}
		}
		services[serviceName] = renderedMethods
	}
	return services, renderedComponents
}

func (sf *schemafier) schemafy(attr *expr.AttributeExpr, noref ...bool) *openapi.Schema {
	rendered := openapiir.RenderSchema(sf.analyzer.AnalyzeSchema(attr, noref...))
	sf.sync()
	return rendered
}

func (sf *schemafier) uniquify(name string, h uint64) string {
	if _, ok := sf.schemas[name]; !ok {
		return name
	}
	candidate := fmt.Sprintf("%s_%016x", name, h)
	if _, ok := sf.schemas[candidate]; !ok {
		return candidate
	}
	for i := 2; ; i++ {
		fallback := fmt.Sprintf("%s_%016x_%d", name, h, i)
		if _, ok := sf.schemas[fallback]; !ok {
			return fallback
		}
	}
}

func (sf *schemafier) claimExplicitName(name string, h uint64) string {
	if existingHash, ok := sf.schemaHashes[name]; ok && existingHash != h {
		panic(fmt.Sprintf("openapi: explicit component name %q is claimed by multiple different schemas; use distinct Meta(\"openapi:typename\", ...) values", name))
	}
	sf.schemaHashes[name] = h
	return name
}

func (sf *schemafier) hashAttribute(att *expr.AttributeExpr, h hash.Hash64) uint64 {
	return sf.analyzer.HashAttribute(att, h)
}

func (sf *schemafier) sync() {
	sf.schemas = openapiir.RenderSchemaMap(sf.analyzer.Components())
	sf.schemaHashes = maps.Clone(sf.analyzer.SchemaHashes())
}
