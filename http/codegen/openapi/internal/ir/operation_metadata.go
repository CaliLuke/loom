package ir

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiinternal "github.com/CaliLuke/loom/http/codegen/openapi/internal"
	"github.com/CaliLuke/loom/internal/securityreq"
)

const defaultOperationIDFormat = "{service}.{method}(.{routeIndex})"

var (
	routeIndexReplacementRegExp = regexp.MustCompile(`\((.*){routeIndex}\)`)
	operationIDSeparatorRegExp  = regexp.MustCompile(`_+`)
)

// BuildRouteOperation analyzes one route-scoped HTTP operation including
// parameters and OpenAPI metadata.
func BuildRouteOperation(route *expr.RouteExpr, path string, bodies *EndpointBodies, rand *expr.ExampleGenerator, apiMeta expr.MetaExpr, closeObjects bool) *Operation {
	if route == nil || route.Endpoint == nil {
		return nil
	}
	endpoint := route.Endpoint
	method := endpoint.MethodExpr
	service := endpoint.Service

	summary := fmt.Sprintf("%s %s", endpoint.Name(), service.Name())
	for _, meta := range []expr.MetaExpr{apiMeta, service.ServiceExpr.Meta, endpoint.Meta, method.Meta} {
		if value, ok := meta.Last("openapi:summary"); ok {
			if value == "{path}" {
				summary = route.Path
			} else {
				summary = value
			}
		}
	}

	operationIDFormat := defaultOperationIDFormat
	for _, meta := range []expr.MetaExpr{apiMeta, method.Service.Meta, endpoint.Meta, method.Meta} {
		if value, ok := meta.Last("openapi:operationId"); ok {
			operationIDFormat = value
		}
	}

	routeIndex := 0
	for index, current := range endpoint.Routes {
		if current == route {
			routeIndex = index
			break
		}
	}

	requestBody := buildRequestBody(endpoint, bodies, rand, closeObjects)
	responseMap := buildResponses(endpoint, bodies, rand, closeObjects)
	responses := make(map[string]*ResponseRef, len(responseMap))
	for status, response := range responseMap {
		responses[status] = &ResponseRef{Value: response}
	}

	operationID := parseOperationIDTemplate(operationIDFormat, service.Name(), endpoint.Name(), routeIndex)
	extensions := mergeExtensions(openapi.ExtensionsFromExpr(method.Meta), buildAsyncOperationExtension(endpoint, path, rand, closeObjects))

	_, deprecated := endpoint.Meta.Last("openapi:deprecated")
	return &Operation{
		Tags:         operationTagNames(endpoint.Meta, service.Meta, service.Name()),
		Summary:      summary,
		Description:  endpoint.Description(),
		OperationID:  operationID,
		Parameters:   buildParameters(endpoint, path, rand, closeObjects),
		RequestBody:  wrapRequestBody(requestBody),
		Responses:    responses,
		Deprecated:   deprecated,
		Security:     buildOperationSecurity(endpoint),
		ExternalDocs: externalDocs(method.Docs, method.Meta),
		Extensions:   extensions,
	}
}

func buildParameters(endpoint *expr.HTTPEndpointExpr, path string, rand *expr.ExampleGenerator, closeObjects bool) []*ParameterRef {
	params := append(paramsFromPath(endpoint, path, rand, closeObjects), paramsFromHeadersAndCookies(endpoint, rand, closeObjects)...)
	if endpoint.MapQueryParams != nil {
		name := *endpoint.MapQueryParams
		if name == "" {
			name = "payload"
		}
		params = append(params, &ParameterRef{
			Value: &Parameter{
				Name:        name,
				Description: "Query parameters",
				In:          "query",
				Required:    name == "payload" || endpoint.MethodExpr.Payload.IsRequired(name),
				Schema: &Schema{
					Type: "object",
					AdditionalProperties: &BoolOrSchema{
						Bool: boolPtr(true),
					},
				},
				Style: "deepObject",
			},
		})
	}
	return params
}

func paramsFromPath(endpoint *expr.HTTPEndpointExpr, path string, rand *expr.ExampleGenerator, closeObjects bool) []*ParameterRef {
	var (
		res       []*ParameterRef
		params    = endpoint.Params
		wildcards = expr.ExtractHTTPWildcards(path)
	)
	codegen.WalkMappedAttr(params, func(name, parameterName string, required bool, attr *expr.AttributeExpr) error { // nolint: errcheck
		location := "query"
		if slices.Contains(wildcards, name) {
			location = "path"
			required = true
		}
		if location != "path" && openapiinternal.IsSecurityParameter(endpoint, location, parameterName) {
			return nil
		}
		res = append(res, paramFor(attr, parameterName, location, required, rand, closeObjects))
		return nil
	})
	return res
}

func paramsFromHeadersAndCookies(endpoint *expr.HTTPEndpointExpr, rand *expr.ExampleGenerator, closeObjects bool) []*ParameterRef {
	var params []*ParameterRef

	expr.WalkMappedAttr(endpoint.Headers, func(name, element string, attr *expr.AttributeExpr) error { // nolint: errcheck
		if openapiinternal.IsSecurityParameter(endpoint, "header", element) {
			return nil
		}
		required := endpoint.Headers.IsRequiredNoDefault(name)
		params = append(params, paramFor(attr, element, "header", required, rand, closeObjects))
		return nil
	})
	expr.WalkMappedAttr(endpoint.Cookies, func(name, element string, attr *expr.AttributeExpr) error { // nolint: errcheck
		if openapiinternal.IsSecurityParameter(endpoint, "cookie", element) {
			return nil
		}
		required := endpoint.Cookies.IsRequiredNoDefault(name)
		params = append(params, paramFor(attr, element, "cookie", required, rand, closeObjects))
		return nil
	})

	return params
}

func paramFor(attr *expr.AttributeExpr, name, in string, required bool, rand *expr.ExampleGenerator, closeObjects bool) *ParameterRef {
	parameter := &Parameter{
		Name:            name,
		In:              in,
		ComponentName:   componentMetaValue(attr, "openapi:component:parameter"),
		Description:     attr.Description,
		AllowEmptyValue: in != "path",
		Required:        required,
		Schema:          NewAnalyzer(rand, closeObjects).AnalyzeSchema(attr),
		Extensions:      openapi.ExtensionsFromExpr(attr.Meta),
	}
	initExamples(parameter, attr, rand, closeObjects)
	return &ParameterRef{Value: parameter}
}

func wrapRequestBody(body *RequestBody) *RequestBodyRef {
	if body == nil {
		return nil
	}
	return &RequestBodyRef{Value: body}
}

func externalDocs(docs *expr.DocsExpr, meta expr.MetaExpr) *ExternalDocs {
	od := openapi.DocsFromExpr(docs, meta)
	if od == nil {
		return nil
	}
	return &ExternalDocs{
		Description: od.Description,
		URL:         od.URL,
		Extensions:  cloneMap(od.Extensions),
	}
}

func buildOperationSecurity(endpoint *expr.HTTPEndpointExpr) []map[string][]string {
	if endpoint == nil || endpoint.MethodExpr == nil {
		return nil
	}
	if _, ok := endpoint.MethodExpr.Meta["security:no"]; ok {
		return []map[string][]string{}
	}
	if len(endpoint.Requirements) == 0 {
		return nil
	}
	return securityreq.OpenAPI(endpoint.Requirements)
}

func operationTagNames(endpointMeta, serviceMeta expr.MetaExpr, serviceName string) []string {
	tagNames := openapi.TagNamesFromExpr(endpointMeta)
	if len(tagNames) > 0 {
		return tagNames
	}
	tagNames = openapi.TagNamesFromExpr(serviceMeta)
	if len(tagNames) > 0 {
		return tagNames
	}
	return []string{serviceName}
}

func parseOperationIDTemplate(template, service, method string, routeIndex int) string {
	if !strings.Contains(template, "{") && routeIndex == 0 {
		return template
	}
	replacer := strings.NewReplacer(
		"{service}", canonicalOperationIDComponent(service),
		"{method}", canonicalOperationIDComponent(method),
	)
	operationID := replacer.Replace(template)
	if routeIndex == 0 {
		return routeIndexReplacementRegExp.ReplaceAllString(operationID, "")
	}
	if separator := routeIndexReplacementRegExp.FindStringSubmatch(template); separator != nil {
		return routeIndexReplacementRegExp.ReplaceAllString(operationID, fmt.Sprintf("%s%d", separator[1], routeIndex))
	}
	return fmt.Sprintf("%s#%d", operationID, routeIndex)
}

func canonicalOperationIDComponent(name string) string {
	component := codegen.SnakeCase(name)
	var builder strings.Builder
	builder.Grow(len(component))
	for _, r := range component {
		switch {
		case unicode.IsLower(r), unicode.IsDigit(r):
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}
	component = operationIDSeparatorRegExp.ReplaceAllString(builder.String(), "_")
	component = strings.Trim(component, "_")
	if component == "" {
		return "operation"
	}
	return component
}
