package openapiimport

import (
	"encoding/json"
	"fmt"
	"go/format"
	"go/token"
	"sort"
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
	plan := planDocument(document)
	if len(plan.diagnostics) > 0 {
		return nil, fmt.Errorf("render OpenAPI design: cannot preserve the input contract:\n%s", plan.diagnostics.Error())
	}
	r := renderer{
		document:     document,
		schemas:      make(map[string]NamedSchema),
		errorSchemas: make(map[string]struct{}),
		operations:   plan.operations,
	}
	if err := r.index(); err != nil {
		return nil, err
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
	for _, scheme := range document.Components.SecuritySchemes {
		if err := r.securityScheme(scheme); err != nil {
			return nil, err
		}
	}
	if err := r.api(); err != nil {
		return nil, err
	}
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
	document     *Document
	schemas      map[string]NamedSchema
	errorSchemas map[string]struct{}
	operations   []operationPlan
	builder      strings.Builder
	indent       int
}

func (r *renderer) namedSchema(named NamedSchema) error {
	path := "#/components/schemas/" + escapeJSONPointer(named.Name)
	expression, object, err := r.schemaExpression(named.Schema, path)
	if err != nil {
		return err
	}
	name := "Imported" + named.GoName
	_, errorType := r.errorSchemas[named.Name]
	switch {
	case object:
		r.open("var %s = Type(%q, func()", name, named.Name)
		r.line("Meta(%q, %q)", "openapi:typename:canonical", "true")
		if err := r.schemaBlock(named.Schema, path, errorType); err != nil {
			return err
		}
		r.close()
	case r.hasSchemaBlock(named.Schema):
		r.open("var %s = Type(%q, %s, func()", name, named.Name, expression)
		r.line("Meta(%q, %q)", "openapi:typename:canonical", "true")
		if err := r.schemaBlock(named.Schema, path, errorType); err != nil {
			return err
		}
		r.close()
	default:
		r.open("var %s = Type(%q, %s, func()", name, named.Name, expression)
		r.line("Meta(%q, %q)", "openapi:typename:canonical", "true")
		r.close()
	}
	r.line("")
	return nil
}

func (r *renderer) api() error {
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
	r.line("Meta(%q, %q)", "openapi:example", "false")
	if err := r.emitExtensions("", r.document.Extensions); err != nil {
		return err
	}
	if strings.HasPrefix(r.document.OpenAPIVersion, "3.0.") || strings.HasPrefix(r.document.OpenAPIVersion, "3.1.") {
		r.line("Meta(%q, %q)", "openapi:version", "3.1")
	}
	for _, tag := range r.importedTags() {
		r.line("Meta(%q)", "openapi:tag:"+tag)
	}
	if r.document.SecurityDefined {
		if err := r.securityRequirements(r.document.Security); err != nil {
			return err
		}
	}
	consumes, produces := r.importedMediaTypes()
	if len(consumes) > 0 || len(produces) > 0 {
		r.open("HTTP(func()")
		if len(consumes) > 0 {
			r.quotedCall("Consumes", consumes)
		}
		if len(produces) > 0 {
			r.quotedCall("Produces", produces)
		}
		r.close()
	}
	r.close()
	r.line("")
	return nil
}

func (r *renderer) importedTags() []string {
	seen := make(map[string]struct{})
	tags := make([]string, 0, len(r.document.Tags))
	add := func(tag string) {
		if _, ok := seen[tag]; ok {
			return
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	for _, tag := range r.document.Tags {
		add(tag)
	}
	for _, operation := range r.document.Operations {
		for _, tag := range operation.Tags {
			add(tag)
		}
	}
	return tags
}

func (r *renderer) service() error {
	serviceName := codegen.Goify(r.document.Title, true)
	if serviceName == "" {
		serviceName = "ImportedAPI"
	}
	r.open("var _ = Service(%q, func()", serviceName)
	for i := range r.document.Operations {
		if err := r.operation(&r.document.Operations[i], r.operations[i], i); err != nil {
			return err
		}
	}
	r.close()
	return nil
}

func (r *renderer) operation(operation *Operation, plan operationPlan, index int) error {
	path := fmt.Sprintf("#/operations/%d", index)

	r.open("Method(%q, func()", operation.GoName)
	if err := r.operationMetadata(operation); err != nil {
		return err
	}
	if operation.SecurityDefined {
		if len(operation.Security) == 0 {
			r.line("NoSecurity()")
		} else if err := r.securityRequirements(operation.Security); err != nil {
			return err
		}
	}
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
	response   responseBodyMode
}

type requestBodyMode uint8

const (
	requestBodyJSON requestBodyMode = iota
	requestBodyMultipart
	requestBodyForm
	requestBodyRaw
)

func (r *renderer) planOperation(operation *Operation, path string) (operationPlan, error) {
	if operation.GoName == "" || !token.IsIdentifier(operation.GoName) || token.Lookup(operation.GoName).IsKeyword() {
		return operationPlan{}, fmt.Errorf("render OpenAPI design: %s has invalid Go method name %q", path, operation.GoName)
	}
	responseMode := operationResponseBodyMode(operation)
	success, failures, err := r.operationResponses(operation, responseMode, path)
	if err != nil {
		return operationPlan{}, err
	}
	parameters, err := r.parameters(operation.Parameters, path+"/parameters")
	if err != nil {
		return operationPlan{}, err
	}
	parameters, err = r.securityParameters(operation, parameters, path+"/security")
	if err != nil {
		return operationPlan{}, err
	}
	body, err := r.requestBody(operation.RequestBody, path+"/requestBody")
	if err != nil {
		return operationPlan{}, err
	}
	return operationPlan{
		success: success, failures: failures, parameters: parameters, body: body, response: responseMode,
	}, nil
}

func (r *renderer) operationMetadata(operation *Operation) error {
	if operation.Description != "" {
		r.line("Description(%q)", operation.Description)
	}
	r.line("Meta(%q, %q)", "openapi:operationId", operation.OperationID)
	r.line("Meta(%q, %q)", "openapi:summary", operation.Summary)
	return r.emitExtensions("", operation.Extensions)
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
	for _, tag := range operation.Tags {
		r.line("Meta(%q)", "openapi:tag:"+tag)
	}
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
		switch plan.body.mode {
		case requestBodyMultipart:
			r.line("MultipartRequest()")
		case requestBodyForm:
			r.line("FormRequest()")
		case requestBodyRaw:
			r.line("SkipRequestBodyEncodeDecode()")
			if err := r.openAPIRequestBody(plan.body.body, path+"/requestBody"); err != nil {
				return err
			}
		default:
			r.line("Body(%q)", plan.body.field)
		}
	}
	switch plan.response {
	case responseBodyStream:
		r.line("SkipResponseBodyEncodeDecode()")
	case responseBodyFile:
		r.line("FileResponse()")
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
	parameter      Parameter
	field          string
	componentName  string
	securityScheme string
}

type renderedBody struct {
	body  RequestBody
	field string
	mode  requestBodyMode
}

type renderedResponse struct {
	status   string
	response Response
	headers  []renderedHeader
	body     string
	rawBody  bool
}

type renderedHeader struct {
	name   string
	field  string
	header Header
}

type renderedMetadata struct {
	name  string
	value string
}

func (r *renderer) parameters(source []Parameter, path string) ([]renderedParameter, error) {
	used := make(map[string]int)
	result := make([]renderedParameter, 0, len(source))
	for i, parameter := range source {
		resolved, componentName, err := r.resolveParameter(parameter, fmt.Sprintf("%s/%d", path, i))
		if err != nil {
			return nil, err
		}
		if resolved.Schema == nil {
			return nil, fmt.Errorf("render OpenAPI design: %s/%d has no schema", path, i)
		}
		// A deprecated parameter has no faithful Loom DSL representation
		// (there is no per-parameter deprecated marker); the flag is
		// intentionally dropped here. Analyze reports this omission as the
		// lossy-allowed "parameter-deprecated" diagnostic.
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
		result = append(result, renderedParameter{parameter: resolved, field: field, componentName: componentName})
	}
	return result, nil
}

func (r *renderer) payload(parameters []renderedParameter, body *renderedBody, path string) error {
	hasBody := body != nil && body.mode != requestBodyRaw
	if len(parameters) == 0 && !hasBody {
		return nil
	}
	r.open("Payload(func()")
	var required []string
	for i, parameter := range parameters {
		if parameter.securityScheme != "" {
			r.line("APIKey(%q, %q, String)", parameter.securityScheme, parameter.field)
			continue
		}
		metadata, err := parameterMetadata(parameter)
		if err != nil {
			return err
		}
		if err := r.attribute(parameter.field, parameter.parameter.Schema, parameter.parameter.Description, fmt.Sprintf("%s/parameters/%d", path, i), metadata...); err != nil {
			return err
		}
		if parameter.parameter.Required {
			required = append(required, parameter.field)
		}
	}
	if hasBody {
		bodyRequired, err := r.payloadBody(body, path+"/requestBody/schema")
		if err != nil {
			return err
		}
		if bodyRequired {
			required = append(required, body.field)
		}
	}
	if len(required) > 0 {
		r.quotedCall("Required", required)
	}
	r.close()
	return nil
}

func parameterMetadata(parameter renderedParameter) ([]renderedMetadata, error) {
	var metadata []renderedMetadata
	if parameter.componentName != "" {
		metadata = append(metadata, renderedMetadata{name: "openapi:component:parameter", value: parameter.componentName})
	}
	if parameter.parameter.In == "query" {
		metadata = append(metadata, renderedMetadata{name: "openapi:allowEmptyValue", value: strconv.FormatBool(parameter.parameter.AllowEmptyValue)})
	}
	extensions, err := renderedExtensions("parameter", parameter.parameter.Extensions)
	return append(metadata, extensions...), err
}

func (r *renderer) payloadBody(body *renderedBody, path string) (bool, error) {
	schema := schemaWithExamples(body.body.Schema, body.body.Examples)
	if body.mode != requestBodyJSON {
		if err := r.emitExtensions("requestBody", body.body.Extensions); err != nil {
			return false, err
		}
		if body.body.Description != "" {
			r.line("Meta(%q, %q)", "openapi:description:requestBody", body.body.Description)
		}
		return false, r.renderRequestTransportBody(schema, path)
	}
	metadata, err := renderedExtensions("requestBody", body.body.Extensions)
	if err != nil {
		return false, err
	}
	if body.body.Description != "" {
		metadata = append(metadata, renderedMetadata{name: "openapi:description:requestBody", value: body.body.Description})
	}
	return body.body.Required, r.attribute(body.field, schema, "", path, metadata...)
}

func (r *renderer) result(response renderedResponse, path string) error {
	return r.responseType("Result", "", response, path)
}

func (r *renderer) errorDefinition(response renderedResponse, path string) error {
	return r.responseType("Error", "Status"+response.status, response, path)
}

func (r *renderer) responseType(call, name string, response renderedResponse, path string) error {
	responseSchema := schemaWithExamples(response.response.Schema, response.response.Examples)
	prefix := call + "("
	if name != "" {
		prefix += strconv.Quote(name) + ", "
	}
	if len(response.headers) == 0 {
		return r.responseTypeWithoutHeaders(call, prefix, response, responseSchema, path+"/content/schema")
	}
	r.open("%sfunc()", prefix)
	for _, header := range response.headers {
		if err := r.attribute(header.field, header.header.Schema, header.header.Description, path+"/headers/"+escapeJSONPointer(header.name)); err != nil {
			return err
		}
	}
	if response.response.Schema != nil && !response.rawBody {
		if err := r.attribute(response.body, responseSchema, "", path+"/content/schema"); err != nil {
			return err
		}
	}
	var required []string
	for _, header := range response.headers {
		if header.header.Required {
			required = append(required, header.field)
		}
	}
	if response.response.Schema != nil && !response.rawBody {
		required = append(required, response.body)
	}
	if len(required) > 0 {
		r.quotedCall("Required", required)
	}
	r.close()
	return nil
}

func (r *renderer) responseTypeWithoutHeaders(call, prefix string, response renderedResponse, schema *Schema, path string) error {
	if response.response.Schema == nil || response.rawBody {
		r.line("%sEmpty)", prefix)
		return nil
	}
	expression, object, err := r.schemaExpression(schema, path)
	if err != nil {
		return err
	}
	if object {
		r.open("%sfunc()", prefix)
		if err := r.emitNullableGoType(schema, path); err != nil {
			return err
		}
		if err := r.schemaBlock(schema, path, call == "Error"); err != nil {
			return err
		}
		r.close()
		return nil
	}
	if r.hasSchemaBlock(schema) || r.effectiveNullable(schema) {
		r.open("%s%s, func()", prefix, expression)
		if err := r.emitNullableGoType(schema, path); err != nil {
			return err
		}
		if err := r.validationBlock(schema, path); err != nil {
			return err
		}
		r.close()
		return nil
	}
	r.line("%s%s)", prefix, expression)
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
	needsBlock := failure || response.response.Description != "" || response.response.ContentType != "" ||
		len(response.headers) > 0 || len(response.response.Extensions) > 0
	if !needsBlock {
		r.line("%s)", prefix)
		return nil
	}
	r.open("%s, func()", prefix)
	if response.response.Description != "" {
		r.line("Description(%q)", response.response.Description)
	}
	if failure {
		r.line("Meta(%q, %q)", "openapi:description:errorName", "false")
	}
	if err := r.emitExtensions("response", response.response.Extensions); err != nil {
		return err
	}
	if response.response.ContentType != "" {
		r.line("ContentType(%q)", response.response.ContentType)
	}
	if response.rawBody {
		if err := r.openAPIBody(schemaWithExamples(response.response.Schema, response.response.Examples), path+"/content/schema"); err != nil {
			return err
		}
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

func schemaWithExamples(schema *Schema, examples []Example) *Schema {
	if schema == nil || len(examples) == 0 {
		return schema
	}
	combined := *schema
	combined.Examples = append(append([]Example(nil), schema.Examples...), examples...)
	return &combined
}

func renderedExtensions(scope string, extensions map[string]any) ([]renderedMetadata, error) {
	names := make([]string, 0, len(extensions))
	for name := range extensions {
		names = append(names, name)
	}
	sort.Strings(names)
	metadata := make([]renderedMetadata, 0, len(names))
	prefix := "openapi:extension:"
	if scope != "" {
		prefix = "openapi:" + scope + ":extension:"
	}
	for _, name := range names {
		value, err := json.Marshal(extensions[name])
		if err != nil {
			return nil, fmt.Errorf("render OpenAPI design: vendor extension %q: %w", name, err)
		}
		metadata = append(metadata, renderedMetadata{name: prefix + name, value: string(value)})
	}
	return metadata, nil
}

func (r *renderer) emitExtensions(scope string, extensions map[string]any) error {
	metadata, err := renderedExtensions(scope, extensions)
	if err != nil {
		return err
	}
	for _, meta := range metadata {
		r.line("Meta(%q, %q)", meta.name, meta.value)
	}
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
