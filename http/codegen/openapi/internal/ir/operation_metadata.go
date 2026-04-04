package ir

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/openapi"
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
	endpointIR := transportir.BuildEndpoint(route.Endpoint)
	routeIndex := routeIndexInEndpoint(route)
	for _, routeIR := range endpointIR.Routes {
		if routeIR.Index == routeIndex && routeIR.SourcePath == route.Path {
			return buildRouteOperationFromIR(endpointIR, routeIR, path, bodies, rand, apiMeta, closeObjects)
		}
	}
	for _, routeIR := range endpointIR.Routes {
		if routeIR.Path == path {
			return buildRouteOperationFromIR(endpointIR, routeIR, path, bodies, rand, apiMeta, closeObjects)
		}
	}
	return buildRouteOperationFromIR(endpointIR, endpointIR.Routes[0], path, bodies, rand, apiMeta, closeObjects)
}

func buildRouteOperationFromIR(endpointIR *transportir.Endpoint, routeIR *transportir.Route, path string, bodies *EndpointBodies, rand *expr.ExampleGenerator, apiMeta expr.MetaExpr, closeObjects bool) *Operation {
	service := endpointIR.Service

	summary := fmt.Sprintf("%s %s", endpointIR.Name, service.Name)
	for _, meta := range []expr.MetaExpr{apiMeta, service.ServiceMeta, endpointIR.Meta, endpointIR.MethodMeta} {
		if value, ok := meta.Last("openapi:summary"); ok {
			if value == "{path}" {
				summary = routeIR.SourcePath
			} else {
				summary = value
			}
		}
	}

	operationIDFormat := defaultOperationIDFormat
	for _, meta := range []expr.MetaExpr{apiMeta, service.ServiceMeta, endpointIR.Meta, endpointIR.MethodMeta} {
		if value, ok := meta.Last("openapi:operationId"); ok {
			operationIDFormat = value
		}
	}

	requestBody := buildRequestBody(endpointIR, bodies, rand, closeObjects)
	responseMap := buildResponses(endpointIR, bodies, rand, closeObjects)
	responses := make(map[string]*ResponseRef, len(responseMap))
	for status, response := range responseMap {
		responses[status] = &ResponseRef{Value: response}
	}

	operationID := parseOperationIDTemplate(operationIDFormat, service.Name, endpointIR.Name, routeIR.Index)
	extensions := mergeExtensions(openapi.ExtensionsFromExpr(endpointIR.MethodMeta), buildAsyncOperationExtension(endpointIR, path, rand, closeObjects))

	_, deprecated := endpointIR.Meta.Last("openapi:deprecated")
	return &Operation{
		Tags:         operationTagNames(endpointIR.Meta, service.Meta, service.Name),
		Summary:      summary,
		Description:  endpointIR.Description,
		OperationID:  operationID,
		Parameters:   buildParameters(endpointIR, rand, closeObjects),
		RequestBody:  wrapRequestBody(requestBody),
		Responses:    responses,
		Deprecated:   deprecated,
		Security:     buildOperationSecurity(endpointIR),
		ExternalDocs: externalDocs(endpointIR.MethodDocs, endpointIR.MethodMeta),
		Extensions:   extensions,
	}
}

func buildParameters(endpointIR *transportir.Endpoint, rand *expr.ExampleGenerator, closeObjects bool) []*ParameterRef {
	params := append(paramsFromPath(endpointIR, rand, closeObjects), paramsFromHeadersAndCookies(endpointIR, rand, closeObjects)...)
	if endpointIR.Request.MapQueryParams != nil {
		name := *endpointIR.Request.MapQueryParams
		if name == "" {
			name = "payload"
		}
		params = append(params, &ParameterRef{
			Value: &Parameter{
				Name:        name,
				Description: "Query parameters",
				In:          "query",
				Required:    name == "payload" || endpointIR.Request.Payload.IsRequired(name),
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

func paramsFromPath(endpointIR *transportir.Endpoint, rand *expr.ExampleGenerator, closeObjects bool) []*ParameterRef {
	var params []*ParameterRef
	for _, parameter := range endpointIR.Request.PathParams {
		params = append(params, paramFor(parameter.Attribute, parameter.HTTPName, "path", true, rand, closeObjects))
	}
	if endpointIR.Request.MapQueryParams != nil {
		return params
	}
	for _, parameter := range endpointIR.Request.QueryParams {
		if isSecurityParameter(endpointIR.Security, "query", parameter.HTTPName) {
			continue
		}
		params = append(params, paramFor(parameter.Attribute, parameter.HTTPName, "query", parameter.Required, rand, closeObjects))
	}
	return params
}

func paramsFromHeadersAndCookies(endpointIR *transportir.Endpoint, rand *expr.ExampleGenerator, closeObjects bool) []*ParameterRef {
	var params []*ParameterRef

	for _, parameter := range endpointIR.Request.Headers {
		if isSecurityParameter(endpointIR.Security, "header", parameter.HTTPName) {
			continue
		}
		params = append(params, paramFor(parameter.Attribute, parameter.HTTPName, "header", parameter.Required, rand, closeObjects))
	}
	for _, parameter := range endpointIR.Request.Cookies {
		if isSecurityParameter(endpointIR.Security, "cookie", parameter.HTTPName) {
			continue
		}
		params = append(params, paramFor(parameter.Attribute, parameter.HTTPName, "cookie", parameter.Required, rand, closeObjects))
	}

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

func buildOperationSecurity(endpointIR *transportir.Endpoint) []map[string][]string {
	if endpointIR == nil {
		return nil
	}
	if endpointIR.Security.Disabled {
		return []map[string][]string{}
	}
	if len(endpointIR.Security.Requirements) == 0 {
		return nil
	}
	return securityreq.OpenAPI(endpointIR.Security.Requirements)
}

func routeIndexInEndpoint(route *expr.RouteExpr) int {
	if route == nil || route.Endpoint == nil {
		return 0
	}
	for index, current := range route.Endpoint.Routes {
		if current == route {
			return index
		}
	}
	return 0
}

func isSecurityParameter(security *transportir.Security, in, name string) bool {
	if security == nil {
		return false
	}
	for _, parameter := range security.Parameters {
		if parameter.In == in && parameter.Name == name {
			return true
		}
	}
	return false
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
