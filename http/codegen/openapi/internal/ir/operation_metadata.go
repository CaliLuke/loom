package ir

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

const defaultOperationIDFormat = "{service}.{method}(.{routeIndex})"

var (
	routeIndexReplacementRegExp = regexp.MustCompile(`\((.*){routeIndex}\)`)
	operationIDSeparatorRegExp  = regexp.MustCompile(`_+`)
)

// BuildRouteOperation analyzes one route-scoped HTTP operation including
// parameters and OpenAPI metadata. It returns an error for a credential
// location that OpenAPI cannot represent.
func BuildRouteOperation(route *expr.RouteExpr, path string, bodies *EndpointBodies, rand *expr.ExampleGenerator, apiMeta expr.MetaExpr, closeObjects bool) (*Operation, error) {
	if route == nil || route.Endpoint == nil {
		return nil, nil
	}
	endpointIR := transportir.BuildEndpoint(route.Endpoint)
	if bodies != nil && bodies.prepared != nil {
		endpointIR = bodies.prepared
	}
	bindings, err := newSecurityBindings(endpointIR.Security.Requirements)
	if err != nil {
		return nil, err
	}
	return buildRouteOperationFromIR(endpointIR, transportir.RouteForExpr(endpointIR, route, path), path, bodies, rand, apiMeta, closeObjects, bindings), nil
}

func buildRouteOperationFromIR(endpointIR *transportir.Endpoint, routeIR *transportir.Route, path string, bodies *EndpointBodies, rand *expr.ExampleGenerator, apiMeta expr.MetaExpr, closeObjects bool, bindings *securityBindings) *Operation {
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

	requestBody := buildRequestBody(endpointIR, bodies, closeObjects)
	responseMap := buildResponses(endpointIR, bodies, rand, closeObjects)
	if routeIR.Method == "HEAD" {
		for _, response := range responseMap {
			response.Content = nil
		}
	}
	responses := make(map[string]*ResponseRef, len(responseMap))
	for status, response := range responseMap {
		responses[status] = &ResponseRef{Value: response}
	}

	operationID := ParseOperationIDTemplate(operationIDFormat, service.Name, endpointIR.Name, routeIR.Index)
	extensions := mergeExtensions(openapi.ExtensionsFromExpr(endpointIR.MethodMeta), buildAsyncOperationExtension(endpointIR, path, bodies))

	_, deprecated := endpointIR.Meta.Last("openapi:deprecated")
	return &Operation{
		Tags:         operationTagNames(endpointIR.Meta, endpointIR.MethodMeta, service.Meta, service.Name),
		Summary:      summary,
		Description:  endpointIR.Description,
		OperationID:  operationID,
		Parameters:   buildParameters(endpointIR, rand, closeObjects, bodies.locations),
		RequestBody:  wrapRequestBody(requestBody),
		Responses:    responses,
		Deprecated:   deprecated,
		Security:     buildOperationSecurity(endpointIR, bindings),
		ExternalDocs: externalDocs(endpointIR.MethodDocs),
		Extensions:   extensions,
	}
}

func buildParameters(endpointIR *transportir.Endpoint, rand *expr.ExampleGenerator, closeObjects bool, prepared ...map[*expr.AttributeExpr]*Schema) []*ParameterRef {
	params := append(paramsFromPath(endpointIR, rand, closeObjects, prepared...), paramsFromHeadersAndCookies(endpointIR, rand, closeObjects, prepared...)...)
	params = addFileResponseRequestParameters(endpointIR, params)
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
				Style:            "deepObject",
				WholeQueryString: true,
			},
		})
	}
	return params
}

func paramsFromPath(endpointIR *transportir.Endpoint, rand *expr.ExampleGenerator, closeObjects bool, prepared ...map[*expr.AttributeExpr]*Schema) []*ParameterRef {
	var params []*ParameterRef
	for _, parameter := range endpointIR.Request.PathParams {
		params = append(params, preparedParamFor(parameter.Attribute, parameter.Value, parameter.HTTPName, "path", true, rand, closeObjects, prepared...))
	}
	if endpointIR.Request.MapQueryParams != nil {
		return params
	}
	for _, parameter := range endpointIR.Request.QueryParams {
		if endpointIR.Security.IsParameter("query", parameter.HTTPName) {
			continue
		}
		params = append(params, preparedParamFor(parameter.Attribute, parameter.Value, parameter.HTTPName, "query", parameter.Required, rand, closeObjects, prepared...))
	}
	return params
}

func paramsFromHeadersAndCookies(endpointIR *transportir.Endpoint, rand *expr.ExampleGenerator, closeObjects bool, prepared ...map[*expr.AttributeExpr]*Schema) []*ParameterRef {
	var params []*ParameterRef

	for _, parameter := range endpointIR.Request.Headers {
		if endpointIR.Security.IsParameter("header", parameter.HTTPName) {
			continue
		}
		params = append(params, preparedParamFor(parameter.Attribute, parameter.Value, parameter.HTTPName, "header", parameter.Required, rand, closeObjects, prepared...))
	}
	for _, parameter := range endpointIR.Request.Cookies {
		if endpointIR.Security.IsParameter("cookie", parameter.HTTPName) {
			continue
		}
		params = append(params, preparedParamFor(parameter.Attribute, parameter.Value, parameter.HTTPName, "cookie", parameter.Required, rand, closeObjects, prepared...))
	}

	return params
}

func paramFor(
	attr *expr.AttributeExpr,
	value *transportir.ValueTarget,
	name, in string,
	rand *expr.ExampleGenerator,
) *ParameterRef {
	return preparedParamFor(attr, value, name, in, false, rand, false)
}

func preparedParamFor(
	attr *expr.AttributeExpr,
	value *transportir.ValueTarget,
	name, in string,
	required bool,
	rand *expr.ExampleGenerator,
	closeObjects bool,
	prepared ...map[*expr.AttributeExpr]*Schema,
) *ParameterRef {
	allowEmptyValue := in == "query"
	if value, ok := attr.Meta.Last("openapi:allowEmptyValue"); ok && in == "query" {
		allowEmptyValue = value == "true"
	}
	parameterContext := attributeExampleContext(attr, closeObjects, "parameter", in, name)
	schema := preparedLocationSchema(attr, parameterContext, rand, closeObjects, prepared)
	schema.Description = ""
	schema.Example = nil
	parameter := &Parameter{
		Name:            name,
		In:              in,
		ComponentName:   componentMetaValue(attr, "openapi:component:parameter"),
		Description:     attr.Description,
		AllowEmptyValue: allowEmptyValue,
		AllowReserved:   metaBool(attr.Meta, "openapi:allowReserved"),
		Style:           metaValue(attr.Meta, "openapi:style"),
		Required:        required,
		Schema:          schema,
		Extensions: openapi.MergeExtensions(
			openapi.ExtensionsFromExpr(attr.Meta),
			openapi.ScopedExtensionsFromExpr(attr.Meta, "parameter"),
		),
	}
	initExamples(parameter, attr, closeObjects, value)
	return &ParameterRef{Value: parameter}
}

func metaBool(meta expr.MetaExpr, key string) bool {
	value, ok := meta.Last(key)
	return ok && value != "false"
}

func wrapRequestBody(body *RequestBody) *RequestBodyRef {
	if body == nil {
		return nil
	}
	return &RequestBodyRef{Value: body}
}

func externalDocs(docs *expr.DocsExpr) *ExternalDocs {
	od := openapi.DocsFromExpr(docs)
	if od == nil {
		return nil
	}
	return &ExternalDocs{
		Description: od.Description,
		URL:         od.URL,
	}
}

func buildOperationSecurity(endpointIR *transportir.Endpoint, bindings *securityBindings) []map[string][]string {
	if endpointIR == nil {
		return nil
	}
	if endpointIR.Security.Disabled {
		return []map[string][]string{}
	}
	if len(endpointIR.Security.Requirements) == 0 {
		return nil
	}
	return bindings.requirements(endpointIR.Security.Requirements)
}

func operationTagNames(endpointMeta, methodMeta, serviceMeta expr.MetaExpr, serviceName string) []string {
	tagNames := openapi.TagNamesFromExpr(endpointMeta)
	for _, name := range openapi.TagNamesFromExpr(methodMeta) {
		if !slices.Contains(tagNames, name) {
			tagNames = append(tagNames, name)
		}
	}
	if len(tagNames) > 0 {
		return tagNames
	}
	tagNames = openapi.TagNamesFromExpr(serviceMeta)
	if len(tagNames) > 0 {
		return tagNames
	}
	return []string{serviceName}
}

// ParseOperationIDTemplate renders an OpenAPI operationId template. It
// replaces {service} and {method} with canonical snake_case components and
// expands or removes the optional (...{routeIndex}) group. A route index
// greater than zero is appended as "#<index>" when the template has no such
// group. A template without placeholders and route index zero is returned
// unchanged.
func ParseOperationIDTemplate(template, service, method string, routeIndex int) string {
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
		case unicode.IsDigit(r), unicode.IsMark(r), unicode.IsLetter(r) && unicode.ToLower(r) == r:
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
