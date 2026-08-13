package openapiimport

import (
	"fmt"
	"go/format"
	"go/token"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

// Options configures deterministic Loom design rendering.
type Options struct {
	// PackageName is the Go package name used by the rendered design. It
	// defaults to design.
	PackageName string
}

// Render renders document as a deterministic, gofmt-formatted Loom design
// source file. Render does not write files or mutate document. It returns an
// error rather than silently omit normalized constructs it cannot faithfully
// express with the Loom DSL.
func Render(document *Document, options Options) ([]byte, error) {
	if document == nil {
		return nil, fmt.Errorf("render OpenAPI design: document is nil")
	}
	packageName := options.PackageName
	if packageName == "" {
		packageName = "design"
	}
	if !token.IsIdentifier(packageName) || token.Lookup(packageName).IsKeyword() {
		return nil, fmt.Errorf("render OpenAPI design: package name %q is not a Go identifier", packageName)
	}
	r := renderer{document: document, schemas: make(map[string]NamedSchema)}
	if err := r.index(); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(document.OpenAPIVersion, "3.1.") && !strings.HasPrefix(document.OpenAPIVersion, "3.2.") {
		return nil, fmt.Errorf("render OpenAPI design: cannot target OpenAPI %s; Loom renders 3.1 or 3.2", document.OpenAPIVersion)
	}

	r.line("package %s", packageName)
	r.line("")
	r.line("import . %q", "github.com/CaliLuke/loom/dsl")
	r.line("")
	for _, schema := range document.Components.Schemas {
		if err := r.namedSchema(schema); err != nil {
			return nil, err
		}
	}
	r.api()
	if err := r.service(); err != nil {
		return nil, err
	}

	source, err := format.Source([]byte(r.builder.String()))
	if err != nil {
		return nil, fmt.Errorf("format rendered OpenAPI design: %w", err)
	}
	return source, nil
}

type renderer struct {
	document *Document
	schemas  map[string]NamedSchema
	builder  strings.Builder
	indent   int
}

func (r *renderer) index() error {
	for _, schema := range r.document.Components.Schemas {
		if schema.Name == "" || schema.GoName == "" {
			return fmt.Errorf("render OpenAPI design: component schema names must not be empty")
		}
		identifier := "Imported" + schema.GoName
		if !token.IsIdentifier(identifier) || token.Lookup(identifier).IsKeyword() {
			return fmt.Errorf("render OpenAPI design: component schema %q has invalid Go name %q", schema.Name, schema.GoName)
		}
		if _, exists := r.schemas[schema.Name]; exists {
			return fmt.Errorf("render OpenAPI design: component schema %q is defined more than once", schema.Name)
		}
		r.schemas[schema.Name] = schema
	}
	return nil
}

func (r *renderer) namedSchema(named NamedSchema) error {
	path := "#/components/schemas/" + escapeJSONPointer(named.Name)
	expression, object, err := r.schemaExpression(named.Schema, path)
	if err != nil {
		return err
	}
	name := "Imported" + named.GoName
	switch {
	case object:
		r.open("var %s = Type(%q, func()", name, named.Name)
		if err := r.schemaBlock(named.Schema, path); err != nil {
			return err
		}
		r.close()
	case r.hasSchemaBlock(named.Schema):
		r.open("var %s = Type(%q, %s, func()", name, named.Name, expression)
		if err := r.schemaBlock(named.Schema, path); err != nil {
			return err
		}
		r.close()
	default:
		r.line("var %s = Type(%q, %s)", name, named.Name, expression)
	}
	r.line("")
	return nil
}

func (r *renderer) api() {
	name := r.document.Title
	if name == "" {
		name = "Imported API"
	}
	r.open("var _ = API(%q, func()", name)
	if r.document.Title != "" {
		r.line("Title(%q)", r.document.Title)
	}
	if r.document.Description != "" {
		r.line("Description(%q)", r.document.Description)
	}
	if r.document.APIVersion != "" {
		r.line("Version(%q)", r.document.APIVersion)
	}
	if strings.HasPrefix(r.document.OpenAPIVersion, "3.1.") {
		r.line("Meta(%q, %q)", "openapi:version", "3.1")
	}
	r.close()
	r.line("")
}

func (r *renderer) service() error {
	serviceName := codegen.Goify(r.document.Title, true)
	if serviceName == "" {
		serviceName = "ImportedAPI"
	}
	r.open("var _ = Service(%q, func()", serviceName)
	for i := range r.document.Operations {
		if err := r.operation(&r.document.Operations[i], i); err != nil {
			return err
		}
	}
	r.close()
	return nil
}

func (r *renderer) operation(operation *Operation, index int) error {
	path := fmt.Sprintf("#/operations/%d", index)
	plan, err := r.planOperation(operation, path)
	if err != nil {
		return err
	}

	r.open("Method(%q, func()", operation.GoName)
	r.operationMetadata(operation)
	if err := r.payload(plan.parameters, plan.body, path+"/payload"); err != nil {
		return err
	}
	if err := r.result(plan.success, path+"/responses/"+plan.success.status); err != nil {
		return err
	}
	if err := r.errorDefinitions(plan.failures, path); err != nil {
		return err
	}
	if err := r.operationHTTP(operation, plan, path); err != nil {
		return err
	}
	r.close()
	return nil
}

type operationPlan struct {
	success    renderedResponse
	failures   []renderedResponse
	parameters []renderedParameter
	body       *renderedBody
}

func (r *renderer) planOperation(operation *Operation, path string) (operationPlan, error) {
	if operation.GoName == "" || !token.IsIdentifier(operation.GoName) || token.Lookup(operation.GoName).IsKeyword() {
		return operationPlan{}, fmt.Errorf("render OpenAPI design: %s has invalid Go method name %q", path, operation.GoName)
	}
	success, failures, err := r.operationResponses(operation, path)
	if err != nil {
		return operationPlan{}, err
	}
	parameters, err := r.parameters(operation.Parameters, path+"/parameters")
	if err != nil {
		return operationPlan{}, err
	}
	body, err := r.requestBody(operation.RequestBody, path+"/requestBody")
	if err != nil {
		return operationPlan{}, err
	}
	return operationPlan{success: success, failures: failures, parameters: parameters, body: body}, nil
}

func (r *renderer) operationMetadata(operation *Operation) {
	if operation.Description != "" {
		r.line("Description(%q)", operation.Description)
	}
	if operation.OperationID != "" {
		r.line("Meta(%q, %q)", "openapi:operationId", operation.OperationID)
	}
	if operation.Summary != "" {
		r.line("Meta(%q, %q)", "openapi:summary", operation.Summary)
	}
	for _, tag := range operation.Tags {
		r.line("Meta(%q)", "openapi:tag:"+tag)
	}
}

func (r *renderer) errorDefinitions(responses []renderedResponse, path string) error {
	for _, response := range responses {
		if err := r.errorDefinition(response, path+"/responses/"+response.status); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) operationHTTP(operation *Operation, plan operationPlan, path string) error {
	r.open("HTTP(func()")
	if err := r.route(operation); err != nil {
		return err
	}
	if operation.Deprecated {
		r.line("Deprecated()")
	}
	for _, parameter := range plan.parameters {
		if err := r.parameterMapping(parameter, path+"/parameters"); err != nil {
			return err
		}
	}
	if plan.body != nil {
		r.line("Body(%q)", plan.body.field)
	}
	if err := r.responseMapping(plan.success, false, path+"/responses/"+plan.success.status); err != nil {
		return err
	}
	for _, response := range plan.failures {
		if err := r.responseMapping(response, true, path+"/responses/"+response.status); err != nil {
			return err
		}
	}
	r.close()
	return nil
}

type renderedParameter struct {
	parameter Parameter
	field     string
}

type renderedBody struct {
	body  RequestBody
	field string
}

type renderedResponse struct {
	status   string
	response Response
	headers  []renderedHeader
	body     string
}

type renderedHeader struct {
	name   string
	field  string
	header Header
}

func (r *renderer) parameters(source []Parameter, path string) ([]renderedParameter, error) {
	used := make(map[string]int)
	result := make([]renderedParameter, 0, len(source))
	for i, parameter := range source {
		resolved, err := r.resolveParameter(parameter, fmt.Sprintf("%s/%d", path, i))
		if err != nil {
			return nil, err
		}
		if resolved.Schema == nil {
			return nil, fmt.Errorf("render OpenAPI design: %s/%d has no schema", path, i)
		}
		if resolved.Deprecated {
			return nil, fmt.Errorf("render OpenAPI design: %s/%d deprecated parameters are not renderable", path, i)
		}
		if resolved.In == "path" && !resolved.Required {
			return nil, fmt.Errorf("render OpenAPI design: %s/%d path parameter %q must be required", path, i, resolved.Name)
		}
		if resolved.In == "path" && (!token.IsIdentifier(resolved.Name) || token.Lookup(resolved.Name).IsKeyword()) {
			return nil, fmt.Errorf("render OpenAPI design: %s/%d path parameter %q is not a Loom wildcard identifier", path, i, resolved.Name)
		}
		field := uniqueName(codegen.Goify(resolved.Name, false), used)
		if field == "" {
			return nil, fmt.Errorf("render OpenAPI design: %s/%d parameter %q has no usable field name", path, i, resolved.Name)
		}
		result = append(result, renderedParameter{parameter: resolved, field: field})
	}
	return result, nil
}

func (r *renderer) requestBody(source *RequestBody, path string) (*renderedBody, error) {
	if source == nil {
		return nil, nil
	}
	body := *source
	if body.Ref != "" {
		return nil, fmt.Errorf("render OpenAPI design: %s request body references are not renderable", path)
	}
	if body.Schema == nil {
		return nil, fmt.Errorf("render OpenAPI design: %s has no schema", path)
	}
	if body.ContentType == "" || !isJSONMediaType(body.ContentType) {
		return nil, fmt.Errorf("render OpenAPI design: %s content type %q is not renderable", path, body.ContentType)
	}
	return &renderedBody{body: body, field: "body"}, nil
}

func (r *renderer) operationResponses(operation *Operation, path string) (renderedResponse, []renderedResponse, error) {
	var success []renderedResponse
	var failures []renderedResponse
	for _, source := range operation.Responses {
		response, err := r.renderedResponse(source, path+"/responses/"+source.Status)
		if err != nil {
			return renderedResponse{}, nil, err
		}
		if strings.HasPrefix(source.Status, "2") {
			success = append(success, response)
		} else {
			failures = append(failures, response)
		}
	}
	if len(success) != 1 {
		return renderedResponse{}, nil, fmt.Errorf("render OpenAPI design: %s must define exactly one 2xx response, got %d", path, len(success))
	}
	return success[0], failures, nil
}

func (r *renderer) renderedResponse(source StatusResponse, path string) (renderedResponse, error) {
	if source.Response.Ref != "" {
		return renderedResponse{}, fmt.Errorf("render OpenAPI design: %s response references are not renderable", path)
	}
	if source.Response.Schema == nil && source.Response.ContentType != "" {
		return renderedResponse{}, fmt.Errorf("render OpenAPI design: %s has content type but no schema", path)
	}
	if source.Response.Schema != nil && (source.Response.ContentType == "" || !isJSONMediaType(source.Response.ContentType)) {
		return renderedResponse{}, fmt.Errorf("render OpenAPI design: %s content type %q is not renderable", path, source.Response.ContentType)
	}
	used := make(map[string]int)
	headers := make([]renderedHeader, 0, len(source.Response.Headers))
	for _, named := range source.Response.Headers {
		header, err := r.resolveHeader(named.Header, path+"/headers/"+escapeJSONPointer(named.Name))
		if err != nil {
			return renderedResponse{}, err
		}
		if header.Deprecated {
			return renderedResponse{}, fmt.Errorf("render OpenAPI design: %s header %q is deprecated and not renderable", path, named.Name)
		}
		if header.Schema == nil {
			return renderedResponse{}, fmt.Errorf("render OpenAPI design: %s header %q has no schema", path, named.Name)
		}
		field := uniqueName(codegen.Goify(named.Name, false), used)
		headers = append(headers, renderedHeader{name: named.Name, field: field, header: header})
	}
	bodyField := ""
	if source.Response.Schema != nil {
		bodyField = uniqueName("body", used)
	}
	return renderedResponse{status: source.Status, response: source.Response, headers: headers, body: bodyField}, nil
}

func (r *renderer) payload(parameters []renderedParameter, body *renderedBody, path string) error {
	if len(parameters) == 0 && body == nil {
		return nil
	}
	r.open("Payload(func()")
	var required []string
	for i, parameter := range parameters {
		if err := r.attribute(parameter.field, parameter.parameter.Schema, parameter.parameter.Description, fmt.Sprintf("%s/parameters/%d", path, i)); err != nil {
			return err
		}
		if parameter.parameter.Required {
			required = append(required, parameter.field)
		}
	}
	if body != nil {
		if err := r.attribute(body.field, body.body.Schema, "", path+"/requestBody/schema"); err != nil {
			return err
		}
		if body.body.Required {
			required = append(required, body.field)
		}
		if body.body.Description != "" {
			r.line("Meta(%q, %q)", "openapi:description:requestBody", body.body.Description)
		}
	}
	if len(required) > 0 {
		r.quotedCall("Required", required)
	}
	r.close()
	return nil
}

func (r *renderer) result(response renderedResponse, path string) error {
	return r.responseType("Result", "", response, path)
}

func (r *renderer) errorDefinition(response renderedResponse, path string) error {
	return r.responseType("Error", "Status"+response.status, response, path)
}

func (r *renderer) responseType(call, name string, response renderedResponse, path string) error {
	prefix := call + "("
	if name != "" {
		prefix += strconv.Quote(name) + ", "
	}
	if len(response.headers) == 0 {
		if response.response.Schema == nil {
			r.line("%sEmpty)", prefix)
			return nil
		}
		expression, object, err := r.schemaExpression(response.response.Schema, path+"/content/schema")
		if err != nil {
			return err
		}
		if object {
			r.open("%sfunc()", prefix)
			if err := r.schemaBlock(response.response.Schema, path+"/content/schema"); err != nil {
				return err
			}
			r.close()
			return nil
		}
		r.line("%s%s)", prefix, expression)
		return nil
	}
	r.open("%sfunc()", prefix)
	for _, header := range response.headers {
		if err := r.attribute(header.field, header.header.Schema, header.header.Description, path+"/headers/"+escapeJSONPointer(header.name)); err != nil {
			return err
		}
	}
	if response.response.Schema != nil {
		if err := r.attribute(response.body, response.response.Schema, "", path+"/content/schema"); err != nil {
			return err
		}
	}
	var required []string
	for _, header := range response.headers {
		if header.header.Required {
			required = append(required, header.field)
		}
	}
	if response.response.Schema != nil {
		required = append(required, response.body)
	}
	if len(required) > 0 {
		r.quotedCall("Required", required)
	}
	r.close()
	return nil
}

func (r *renderer) route(operation *Operation) error {
	switch operation.Method {
	case "GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS", "TRACE", "CONNECT", "PATCH", "QUERY":
		r.line("%s(%q)", operation.Method, operation.Path)
	default:
		r.line("Route(%q, %q)", operation.Method, operation.Path)
	}
	return nil
}

func (r *renderer) parameterMapping(parameter renderedParameter, path string) error {
	name := parameter.field
	if parameter.parameter.In != "path" && name != parameter.parameter.Name {
		name += ":" + parameter.parameter.Name
	}
	switch parameter.parameter.In {
	case "path", "query":
		r.line("Param(%q)", name)
	case "header":
		r.line("Header(%q)", name)
	case "cookie":
		r.line("Cookie(%q)", name)
	default:
		return fmt.Errorf("render OpenAPI design: %s parameter location %q is not renderable", path, parameter.parameter.In)
	}
	return nil
}

func (r *renderer) responseMapping(response renderedResponse, failure bool, path string) error {
	status, err := strconv.Atoi(response.status)
	if err != nil {
		return fmt.Errorf("render OpenAPI design: %s status %q is not numeric", path, response.status)
	}
	prefix := "Response("
	if failure {
		prefix += strconv.Quote("Status"+response.status) + ", "
	}
	prefix += strconv.Itoa(status)
	needsBlock := response.response.Description != "" || response.response.ContentType != "" ||
		len(response.headers) > 0 || response.response.Schema != nil && len(response.headers) > 0
	if !needsBlock {
		r.line("%s)", prefix)
		return nil
	}
	r.open("%s, func()", prefix)
	if response.response.Description != "" {
		r.line("Description(%q)", response.response.Description)
	}
	if response.response.ContentType != "" {
		r.line("ContentType(%q)", response.response.ContentType)
	}
	for _, header := range response.headers {
		name := header.field
		if name != header.name {
			name += ":" + header.name
		}
		r.line("Header(%q)", name)
	}
	if response.body != "" && len(response.headers) > 0 {
		r.line("Body(%q)", response.body)
	}
	r.close()
	return nil
}

func (r *renderer) open(format string, args ...any) {
	r.line(format+" {", args...)
	r.indent++
}

func (r *renderer) close() {
	r.indent--
	r.line("})")
}

func (r *renderer) line(format string, args ...any) {
	if format != "" {
		r.builder.WriteString(strings.Repeat("\t", r.indent))
		r.builder.WriteString(fmt.Sprintf(format, args...))
	}
	r.builder.WriteByte('\n')
}
