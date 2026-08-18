package openapiimport

import (
	"fmt"
	"go/token"
	"strings"
)

type responseBodyMode uint8

const (
	responseBodyEncoded responseBodyMode = iota
	responseBodyStream
	responseBodyFile
)

// documentPlan is the render-ready result of applying the importer subset.
type documentPlan struct {
	operations  []operationPlan
	diagnostics Diagnostics
}

// planDocument applies the importer subset once for both Analyze and Render.
// Rendering retains defensive errors for manually-constructed Documents, but
// source documents are admitted only when this planner reports no diagnostics.
func planDocument(document *Document) documentPlan {
	if document == nil {
		return documentPlan{diagnostics: Diagnostics{{Code: "document", Path: "#", Message: "document is nil"}}}
	}
	planner := documentPlanner{
		document: document,
		schemas:  make(map[string]NamedSchema, len(document.Components.Schemas)),
		used:     make(map[string]struct{}),
	}
	planner.index()
	if !renderableVersion(document.OpenAPIVersion) {
		planner.unsupported("openapi-version", "#", "only OpenAPI 3.0, 3.1, and 3.2 documents are renderable")
	}
	for _, named := range document.Components.Schemas {
		planner.schema(named.Schema, "#/components/schemas/"+escapeJSONPointer(named.Name))
	}
	for index := range document.Operations {
		planner.operation(&document.Operations[index], fmt.Sprintf("#/operations/%d", index))
	}
	for _, named := range document.Components.Parameters {
		path := "#/components/parameters/" + escapeJSONPointer(named.Name)
		if !renderableComponentParameterName(named.Name) {
			planner.unsupported("component-parameter-name", path, "component parameter names containing '~' or '/' are not renderable")
		}
		if _, ok := planner.used[named.Name]; !ok {
			planner.unsupported("unused-component-parameter", path, "unreferenced parameter components cannot be rendered faithfully")
		}
	}
	for _, named := range document.Components.RequestBodies {
		planner.unsupported("component-request-body", "#/components/requestBodies/"+escapeJSONPointer(named.Name), "reusable request body components are not in the strict import subset")
	}
	for _, named := range document.Components.Responses {
		planner.unsupported("component-response", "#/components/responses/"+escapeJSONPointer(named.Name), "reusable response components are not in the strict import subset")
	}
	for _, named := range document.Components.Headers {
		planner.unsupported("component-header", "#/components/headers/"+escapeJSONPointer(named.Name), "reusable header components are not in the strict import subset")
	}
	planner.diagnostics = planner.diagnostics.sorted()
	plan := documentPlan{diagnostics: planner.diagnostics}
	if len(plan.diagnostics) > 0 {
		return plan
	}
	renderer := renderer{document: document, schemas: planner.schemas}
	for index := range document.Operations {
		operation, err := renderer.planOperation(&document.Operations[index], fmt.Sprintf("#/operations/%d", index))
		if err != nil {
			plan.diagnostics = Diagnostics{{Code: "render-plan", Path: fmt.Sprintf("#/operations/%d", index), Message: strings.TrimPrefix(err.Error(), "render OpenAPI design: ")}}
			return plan
		}
		plan.operations = append(plan.operations, operation)
	}
	disambiguateSharedErrorStatuses(document.Operations, plan.operations)
	return plan
}

func disambiguateSharedErrorStatuses(operations []Operation, plans []operationPlan) {
	statusCounts := make(map[string]int)
	for _, operation := range plans {
		for _, response := range operation.failures {
			statusCounts[response.status]++
		}
	}
	for operationIndex := range plans {
		for responseIndex := range plans[operationIndex].failures {
			response := &plans[operationIndex].failures[responseIndex]
			if statusCounts[response.status] > 1 {
				response.errorName = operations[operationIndex].GoName + "Status" + response.status
			}
		}
	}
}

type documentPlanner struct {
	document    *Document
	schemas     map[string]NamedSchema
	used        map[string]struct{}
	diagnostics Diagnostics
}

func (p *documentPlanner) index() {
	for _, schema := range p.document.Components.Schemas {
		path := "#/components/schemas/" + escapeJSONPointer(schema.Name)
		if schema.Name == "" || schema.GoName == "" {
			p.unsupported("schema-name", path, "component schema names must not be empty")
			continue
		}
		identifier := "Imported" + schema.GoName
		if !token.IsIdentifier(identifier) || token.Lookup(identifier).IsKeyword() {
			p.unsupported("schema-name", path, fmt.Sprintf("component schema %q has invalid Go name %q", schema.Name, schema.GoName))
			continue
		}
		if _, exists := p.schemas[schema.Name]; exists {
			p.unsupported("schema-name", path, fmt.Sprintf("component schema %q is defined more than once", schema.Name))
			continue
		}
		p.schemas[schema.Name] = schema
	}
}

func (p *documentPlanner) operation(operation *Operation, path string) {
	if operation.GoName == "" || !token.IsIdentifier(operation.GoName) || token.Lookup(operation.GoName).IsKeyword() {
		p.unsupported("operation-name", path, fmt.Sprintf("invalid Go method name %q", operation.GoName))
	}
	for index, parameter := range operation.Parameters {
		p.parameter(parameter, fmt.Sprintf("%s/parameters/%d", path, index))
	}
	p.requestBody(operation.RequestBody, path+"/requestBody")
	responseMode := operationResponseBodyMode(operation)
	successClass := successResponseClass(operation.Responses)
	successes := 0
	for _, response := range operation.Responses {
		success := isSuccessResponseStatus(response.Status, successClass)
		if success {
			successes++
		}
		responsePath := path + "/responses/" + escapeJSONPointer(response.Status)
		if !success && p.schemaContainsUntaggedOneOf(response.Response.Schema, make(map[string]struct{})) {
			p.unsupported("schema-oneof-location", responsePath+"/content/schema", "untagged oneOf is supported only in JSON request and success response bodies")
		}
		if response.Response.Schema != nil && !isJSONMediaType(response.Response.ContentType) &&
			responseMode == responseBodyEncoded {
			p.unsupported(
				"media-type",
				responsePath+"/content",
				fmt.Sprintf("content type %q requires a non-JSON success response", response.Response.ContentType),
			)
		}
		p.response(response.Response, responsePath, responseMode != responseBodyEncoded)
	}
	if successes != 1 {
		p.unsupported("success-response-count", path+"/responses", fmt.Sprintf("must define exactly one 2xx response or, when absent, exactly one 3xx response, got %d", successes))
	}
}

func (p *documentPlanner) parameter(parameter Parameter, path string) {
	resolved, componentName, err := resolveParameterReference(parameter, p.document.Components)
	if err != nil {
		p.unsupported("parameter-reference", path, err.Error())
		return
	}
	if componentName != "" {
		p.used[componentName] = struct{}{}
	}
	if resolved.Schema == nil {
		p.unsupported("parameter-schema", path, "parameter has no schema")
	} else {
		if p.schemaContainsUntaggedOneOf(resolved.Schema, make(map[string]struct{})) {
			p.unsupported("schema-oneof-location", path+"/schema", "untagged oneOf is supported only in JSON bodies and component schemas")
		}
		p.schema(resolved.Schema, path+"/schema")
	}
	if resolved.In != "query" && resolved.AllowEmptyValue {
		p.unsupported("parameter-allow-empty-value", path, "allowEmptyValue is only supported for query parameters")
	}
	if strings.Contains(resolved.Name, ":") {
		p.unsupported("wire-name", path+"/name", "parameter wire names containing ':' are not renderable")
	}
	if resolved.In == "path" && !resolved.Required {
		p.unsupported("path-parameter-required", path, fmt.Sprintf("path parameter %q must be required", resolved.Name))
	}
	if resolved.In == "path" && (!token.IsIdentifier(resolved.Name) || token.Lookup(resolved.Name).IsKeyword()) {
		p.unsupported("path-parameter-name", path+"/name", fmt.Sprintf("path parameter %q is not a Loom wildcard identifier", resolved.Name))
	}
}

func (p *documentPlanner) requestBody(body *RequestBody, path string) {
	if body == nil {
		return
	}
	if body.Ref != "" {
		p.unsupported("request-body-reference", path, "request body references are not renderable")
		return
	}
	if len(body.ContentTypes) == 0 {
		p.content("", body.Schema, path, false)
		return
	}
	for _, contentType := range body.ContentTypes {
		contentPath := path + "/content/" + escapeJSONPointer(contentType)
		switch {
		case isJSONMediaType(contentType):
		case isMultipartMediaType(contentType), isFormMediaType(contentType):
		default:
			p.unsupported("media-type", contentPath, fmt.Sprintf("content type %q is not renderable", contentType))
		}
		if body.Schema == nil {
			p.unsupported("content-schema", contentPath+"/schema", "content type has no schema")
			continue
		}
		if !isJSONMediaType(contentType) && p.schemaContainsUntaggedOneOf(body.Schema, make(map[string]struct{})) {
			p.unsupported("schema-oneof-location", contentPath+"/schema", "untagged oneOf is supported only in JSON bodies and component schemas")
		}
		p.schema(body.Schema, contentPath+"/schema")
	}
}

func (p *documentPlanner) response(response Response, path string, allowNonJSON bool) {
	if response.Ref != "" {
		p.unsupported("response-reference", path, "response references are not renderable")
		return
	}
	p.content(response.ContentType, response.Schema, path, allowNonJSON)
	for _, named := range response.Headers {
		headerPath := path + "/headers/" + escapeJSONPointer(named.Name)
		if strings.Contains(named.Name, ":") {
			p.unsupported("wire-name", headerPath, "response header names containing ':' are not renderable")
		}
		if named.Header.Ref != "" {
			p.unsupported("header-reference", headerPath, "header references are not renderable")
			continue
		}
		if named.Header.Schema == nil {
			p.unsupported("header-schema", headerPath, fmt.Sprintf("header %q has no schema", named.Name))
			continue
		}
		if p.schemaContainsUntaggedOneOf(named.Header.Schema, make(map[string]struct{})) {
			p.unsupported("schema-oneof-location", headerPath+"/schema", "untagged oneOf is supported only in JSON bodies and component schemas")
		}
		p.schema(named.Header.Schema, headerPath+"/schema")
	}
}

func (p *documentPlanner) content(contentType string, schema *Schema, path string, allowNonJSON bool) {
	if schema == nil {
		if contentType != "" {
			p.unsupported("content-schema", path+"/content", "content type has no schema")
		}
		return
	}
	if contentType == "" || !allowNonJSON && !isJSONMediaType(contentType) {
		p.unsupported("media-type", path+"/content", fmt.Sprintf("content type %q is not renderable", contentType))
	}
	if !isJSONMediaType(contentType) && p.schemaContainsUntaggedOneOf(schema, make(map[string]struct{})) {
		p.unsupported("schema-oneof-location", path+"/content/schema", "untagged oneOf is supported only in JSON bodies and component schemas")
	}
	p.schema(schema, path+"/content/schema")
}

func (p *documentPlanner) schemaContainsUntaggedOneOf(schema *Schema, seen map[string]struct{}) bool {
	if schema == nil {
		return false
	}
	if len(schema.OneOf) > 0 {
		return true
	}
	if schema.Ref != "" {
		name := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		if name == schema.Ref {
			return false
		}
		if _, ok := seen[name]; ok {
			return false
		}
		seen[name] = struct{}{}
		named, ok := p.schemas[name]
		return ok && p.schemaContainsUntaggedOneOf(named.Schema, seen)
	}
	for _, base := range schema.Bases {
		if p.schemaContainsUntaggedOneOf(base, seen) {
			return true
		}
	}
	for _, property := range schema.Properties {
		if p.schemaContainsUntaggedOneOf(property.Schema, seen) {
			return true
		}
	}
	if p.schemaContainsUntaggedOneOf(schema.Items, seen) {
		return true
	}
	return schema.AdditionalProperties != nil &&
		p.schemaContainsUntaggedOneOf(schema.AdditionalProperties.Schema, seen)
}

func operationResponseBodyMode(operation *Operation) responseBodyMode {
	successClass := successResponseClass(operation.Responses)
	for _, response := range operation.Responses {
		if !isSuccessResponseStatus(response.Status, successClass) || response.Response.Schema == nil ||
			isJSONMediaType(response.Response.ContentType) {
			continue
		}
		if (operation.Method == "GET" || operation.Method == "HEAD") && response.Status == "200" &&
			!hasFileProtocolResponse(operation.Responses) {
			return responseBodyFile
		}
		return responseBodyStream
	}
	return responseBodyEncoded
}

func successResponseClass(responses []StatusResponse) byte {
	for _, response := range responses {
		if len(response.Status) == 3 && response.Status[0] == '2' {
			return '2'
		}
	}
	return '3'
}

func isSuccessResponseStatus(status string, successClass byte) bool {
	return len(status) == 3 && status[0] == successClass
}

func hasFileProtocolResponse(responses []StatusResponse) bool {
	for _, response := range responses {
		switch response.Status {
		case "206", "304", "412", "416":
			return true
		}
	}
	return false
}

func isMultipartMediaType(contentType string) bool {
	return normalizedMediaType(contentType) == "multipart/form-data"
}

func isFormMediaType(contentType string) bool {
	return normalizedMediaType(contentType) == "application/x-www-form-urlencoded"
}

func normalizedMediaType(contentType string) string {
	return strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
}

func (p *documentPlanner) schema(schema *Schema, path string) {
	if schema != nil && schema.unsupportedComposition {
		p.unsupported("schema-composition", path, "allOf, oneOf, anyOf, and not are not in the strict import subset")
		return
	}
	if schema != nil && len(schema.OneOf) > 0 {
		for index, branch := range schema.OneOf {
			if branch == nil || branch.Ref == "" {
				p.unsupported("schema-oneof-branch", fmt.Sprintf("%s/oneOf/%d", path, index), "untagged oneOf branches must reference concrete object components")
				continue
			}
			name := strings.TrimPrefix(branch.Ref, "#/components/schemas/")
			named, ok := p.schemas[name]
			if name == branch.Ref || !ok || named.Schema == nil || named.Schema.Ref != "" || named.Schema.Type != "object" {
				p.unsupported("schema-oneof-branch", fmt.Sprintf("%s/oneOf/%d", path, index), "untagged oneOf branches must reference concrete object components")
				continue
			}
			if !p.schemaIsFlatPrimitiveObject(named.Schema) {
				p.unsupported("schema-oneof-branch", fmt.Sprintf("%s/oneOf/%d", path, index), "untagged oneOf branches must be flat objects with primitive properties")
			}
		}
	}
	renderer := renderer{document: p.document, schemas: p.schemas}
	_, _, err := renderer.schemaExpression(schema, path)
	if err == nil {
		err = renderer.schemaBlock(schema, path, false)
	}
	if err != nil {
		p.unsupported("schema", path, strings.TrimPrefix(err.Error(), "render OpenAPI design: "))
	}
}

func (p *documentPlanner) schemaIsFlatPrimitiveObject(schema *Schema) bool {
	if schema == nil || schema.Type != "object" || len(schema.Bases) > 0 || schema.Items != nil || len(schema.OneOf) > 0 {
		return false
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		return false
	}
	for _, property := range schema.Properties {
		if !p.schemaIsPrimitive(property.Schema, make(map[string]struct{})) {
			return false
		}
	}
	return true
}

func (p *documentPlanner) schemaIsPrimitive(schema *Schema, seen map[string]struct{}) bool {
	if schema == nil {
		return false
	}
	if schema.Ref != "" {
		name := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		if name == schema.Ref {
			return false
		}
		if _, ok := seen[name]; ok {
			return false
		}
		seen[name] = struct{}{}
		named, ok := p.schemas[name]
		return ok && p.schemaIsPrimitive(named.Schema, seen)
	}
	if schema.Unconstrained {
		return true
	}
	switch schema.Type {
	case "boolean", "integer", "number", "string":
		return len(schema.Bases) == 0 && len(schema.OneOf) == 0 && len(schema.Properties) == 0 &&
			schema.Items == nil && schema.AdditionalProperties == nil
	default:
		return false
	}
}

func (p *documentPlanner) unsupported(code, path, message string) {
	p.diagnostics.add(code, path, message)
}

func parameterKey(parameter Parameter, components Components) (string, bool) {
	resolved, _, err := resolveParameterReference(parameter, components)
	if err != nil || resolved.Name == "" || resolved.In == "" {
		return "", false
	}
	return resolved.Name + "\x00" + resolved.In, true
}

func renderableComponentParameterName(name string) bool {
	return !strings.ContainsAny(name, "~/")
}

func resolveParameterReference(parameter Parameter, components Components) (Parameter, string, error) {
	if parameter.Ref == "" {
		return parameter, "", nil
	}
	name, err := localComponentReferenceName(parameter.Ref, "#/components/parameters/")
	if err != nil {
		return Parameter{}, "", fmt.Errorf("parameter reference %q %w", parameter.Ref, err)
	}
	for _, named := range components.Parameters {
		if named.Name != name {
			continue
		}
		if named.Parameter.Ref != "" {
			return Parameter{}, "", fmt.Errorf("parameter reference %q is nested and not renderable", parameter.Ref)
		}
		return named.Parameter, name, nil
	}
	return Parameter{}, "", fmt.Errorf("parameter reference %q does not resolve", parameter.Ref)
}

func localComponentReferenceName(ref, prefix string) (string, error) {
	segment := strings.TrimPrefix(ref, prefix)
	if segment == ref || segment == "" || strings.Contains(segment, "/") {
		return "", fmt.Errorf("has the wrong kind")
	}
	name, err := unescapeJSONPointerSegment(segment)
	if err != nil {
		return "", fmt.Errorf("has invalid JSON Pointer escaping")
	}
	return name, nil
}

func unescapeJSONPointerSegment(value string) (string, error) {
	var builder strings.Builder
	for index := 0; index < len(value); index++ {
		if value[index] != '~' {
			builder.WriteByte(value[index])
			continue
		}
		if index+1 == len(value) {
			return "", fmt.Errorf("trailing '~'")
		}
		index++
		switch value[index] {
		case '0':
			builder.WriteByte('~')
		case '1':
			builder.WriteByte('/')
		default:
			return "", fmt.Errorf("invalid escape")
		}
	}
	return builder.String(), nil
}
