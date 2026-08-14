package openapiimport

import (
	"fmt"
	"go/token"
	"strings"
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
		planner.unsupported("openapi-version", "#", "only OpenAPI 3.1 and 3.2 documents are renderable")
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
	return plan
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
	successes := 0
	for _, response := range operation.Responses {
		if strings.HasPrefix(response.Status, "2") {
			successes++
		}
		p.response(response.Response, path+"/responses/"+escapeJSONPointer(response.Status))
	}
	if successes != 1 {
		p.unsupported("success-response-count", path+"/responses", fmt.Sprintf("must define exactly one 2xx response, got %d", successes))
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
		p.schema(resolved.Schema, path+"/schema")
	}
	if resolved.Deprecated {
		p.unsupported("parameter-deprecated", path, "deprecated parameters are not renderable")
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
	p.content(body.ContentType, body.Schema, path)
}

func (p *documentPlanner) response(response Response, path string) {
	if response.Ref != "" {
		p.unsupported("response-reference", path, "response references are not renderable")
		return
	}
	p.content(response.ContentType, response.Schema, path)
	for _, named := range response.Headers {
		headerPath := path + "/headers/" + escapeJSONPointer(named.Name)
		if strings.Contains(named.Name, ":") {
			p.unsupported("wire-name", headerPath, "response header names containing ':' are not renderable")
		}
		if named.Header.Ref != "" {
			p.unsupported("header-reference", headerPath, "header references are not renderable")
			continue
		}
		if named.Header.Deprecated {
			p.unsupported("header-deprecated", headerPath, fmt.Sprintf("header %q is deprecated and not renderable", named.Name))
		}
		if named.Header.Schema == nil {
			p.unsupported("header-schema", headerPath, fmt.Sprintf("header %q has no schema", named.Name))
			continue
		}
		p.schema(named.Header.Schema, headerPath+"/schema")
	}
}

func (p *documentPlanner) content(contentType string, schema *Schema, path string) {
	if schema == nil {
		if contentType != "" {
			p.unsupported("content-schema", path+"/content", "content type has no schema")
		}
		return
	}
	if contentType == "" || !isJSONMediaType(contentType) {
		p.unsupported("media-type", path+"/content", fmt.Sprintf("content type %q is not renderable", contentType))
	}
	p.schema(schema, path+"/content/schema")
}

func (p *documentPlanner) schema(schema *Schema, path string) {
	renderer := renderer{document: p.document, schemas: p.schemas}
	_, _, err := renderer.schemaExpression(schema, path)
	if err == nil {
		err = renderer.schemaBlock(schema, path)
	}
	if err != nil {
		p.unsupported("schema", path, strings.TrimPrefix(err.Error(), "render OpenAPI design: "))
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
