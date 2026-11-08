package openapiimport

import (
	"fmt"
	"go/token"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/internal/openapiref"
)

type responseBodyMode uint8

const (
	responseBodyEncoded responseBodyMode = iota
	responseBodyStream
	responseBodyFile
)

// documentPlan is the render-ready result of applying the importer subset.
type documentPlan struct {
	document     *Document
	operations   []operationPlan
	diagnostics  Diagnostics
	schemas      map[string]NamedSchema
	errorSchemas map[string]struct{}
	references   *schemaReferencePlan
}

// planDocument applies the importer subset once for both Analyze and Render.
// Rendering retains defensive errors for manually-constructed Documents, but
// source documents are admitted only when this planner reports no diagnostics.
func planDocument(document *Document) documentPlan {
	if document == nil {
		return documentPlan{diagnostics: Diagnostics{{Code: "document", Path: "#", Message: "document is nil"}}}
	}
	if diagnostics := blockedDocumentDiagnostics(document); len(diagnostics) > 0 {
		return documentPlan{diagnostics: diagnostics}
	}
	// Preserve the source reference graph for name inventory; the resolved
	// occurrence model has already consumed reusable reference identities.
	nameSource := *document
	nameSource.Components.Schemas = slices.Clone(document.Components.Schemas)
	nameSource.Components.SecuritySchemes = slices.Clone(document.Components.SecuritySchemes)
	resolved, diagnostics := resolveReusableDocument(&nameSource)
	document = resolved
	planner := documentPlanner{
		diagnostics: diagnostics,
		document:    document,
		schemas:     make(map[string]NamedSchema, len(document.Components.Schemas)),
		used:        make(map[string]struct{}),
	}
	planner.index()
	if document.SecurityDefined {
		planner.securityRequirements(document.Security, "#/security")
	}
	if !renderableVersion(document.OpenAPIVersion) {
		planner.unsupported("openapi-version", "#", "only OpenAPI 3.0, 3.1, and 3.2 documents are renderable")
	}
	for _, named := range document.Components.Schemas {
		planner.schema(named.Schema, "#/components/schemas/"+escapeJSONPointer(named.Name), objectDeclaration)
	}
	for index := range document.Operations {
		op := &document.Operations[index]
		if op.SecurityDefined {
			planner.securityRequirements(op.Security, op.sourcePath+"/security")
		}
		path := op.sourcePath
		if path == "" {
			path = fmt.Sprintf("#/operations/%d", index)
		}
		planner.operation(op, path)
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
	if len(planner.diagnostics) > 0 {
		return documentPlan{diagnostics: planner.diagnostics.sorted()}
	}
	if err := allocateDesignNames(&nameSource); err != nil {
		return documentPlan{diagnostics: designNameDiagnostics(err)}
	}
	document.Components.Schemas = nameSource.Components.Schemas
	document.Components.SecuritySchemes = nameSource.Components.SecuritySchemes
	document.names = nameSource.names
	// Rebuild the schema index with the single allocator's final names.
	for _, named := range document.Components.Schemas {
		planner.schemas[named.Name] = named
	}
	return planner.renderPlan()
}

// renderPlan freezes names and reference decisions after subset admission.
// Error clones depend on final operation names; reference occurrences depend on
// those clones, so neither can be independently rediscovered during rendering.
func (p *documentPlanner) renderPlan() documentPlan {
	plan := documentPlan{document: p.document}
	operationRenderer := renderer{document: p.document, schemas: p.schemas}
	for index := range p.document.Operations {
		operation, err := operationRenderer.planOperation(&p.document.Operations[index], fmt.Sprintf("#/operations/%d", index))
		if err != nil {
			plan.diagnostics = Diagnostics{{Code: "render-plan", Path: fmt.Sprintf("#/operations/%d", index), Message: strings.TrimPrefix(err.Error(), "render OpenAPI design: ")}}
			return plan
		}
		plan.operations = append(plan.operations, operation)
	}
	disambiguateSharedErrorStatuses(p.document.Operations, plan.operations)
	declarations := renderer{
		document: p.document, schemas: p.schemas,
		errorSchemas: make(map[string]struct{}), operations: plan.operations,
	}
	if err := declarations.indexErrors(); err != nil {
		plan.diagnostics = Diagnostics{{Code: "render-plan", Path: "#", Message: err.Error()}}
		return plan
	}
	plan.schemas, plan.errorSchemas = declarations.schemas, declarations.errorSchemas
	plan.references, plan.diagnostics = planSchemaReferences(p.document, plan.schemas, plan.operations)
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
	p.parameterNamespaces(operation, path)
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
		if response.Response.Schema != nil && len(response.Response.Schema.MediaTypes) == 0 && !isJSONMediaType(response.Response.ContentType) &&
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
		p.schemaEncoding(resolved.Schema, path+"/schema", schemaEncodedValue, make(map[schemaEncodingOccurrence]bool))
		p.schema(resolved.Schema, path+"/schema", objectField)
		p.admitParameter(resolved, path)
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
		// resolveReusableDocument already reported this unresolved occurrence.
		return
	}
	if body.Schema != nil && len(body.Schema.MediaTypes) > 0 {
		p.mediaContracts(body.Schema, path)
		return
	}
	if len(body.ContentTypes) == 0 {
		p.content("", body.Schema, path, false, requestObjectRole(*body))
		return
	}
	for _, contentType := range body.ContentTypes {
		contentPath := path + "/content/" + escapeJSONPointer(contentType)
		mode, err := requestBodyModeFor(contentType, contentPath)
		if err != nil {
			p.unsupported("media-type", contentPath, fmt.Sprintf("content type %q is not renderable", contentType))
		}
		if body.Schema == nil {
			p.unsupported("content-schema", contentPath+"/schema", "content type has no schema")
			continue
		}
		if mode != requestBodyMultipart {
			p.schemaEncoding(body.Schema, contentPath+"/schema", schemaEncodedValue, make(map[schemaEncodingOccurrence]bool))
		}
		p.schema(body.Schema, contentPath+"/schema", requestObjectRole(*body))
		if isFormMediaType(contentType) || isMultipartMediaType(contentType) {
			renderer := renderer{schemas: p.schemas}
			if isFormMediaType(contentType) && !renderer.unionSchema(body.Schema) && len(body.Schema.MediaEncoding) == 0 {
				raw, err := renderer.formBodyRequiresRaw(body.Schema, contentPath+"/schema", make(map[string]struct{}))
				if err != nil {
					p.unsupported("media-schema", contentPath, err.Error())
					continue
				}
				if raw {
					continue
				}
			}
			p.mediaBranchSchema(body.Schema, contentType, contentPath, true, mediaSchemaContext{})
		}
	}
}

func (p *documentPlanner) response(response Response, path string, allowNonJSON bool) {
	if response.Ref != "" {
		// resolveReusableDocument already reported this unresolved occurrence.
		return
	}
	if response.Schema != nil && len(response.Schema.MediaTypes) > 0 {
		p.mediaContracts(response.Schema, path)
	} else {
		p.content(response.ContentType, response.Schema, path, allowNonJSON, responseObjectRole(response, allowNonJSON))
	}
	for _, named := range response.Headers {
		headerPath := path + "/headers/" + escapeJSONPointer(named.Name)
		if strings.Contains(named.Name, ":") {
			p.unsupported("wire-name", headerPath, "response header names containing ':' are not renderable")
		}
		if named.Header.Ref != "" {
			// Reference resolution owns this diagnostic.
			continue
		}
		if named.Header.Schema == nil {
			p.unsupported("header-schema", headerPath, fmt.Sprintf("header %q has no schema", named.Name))
			continue
		}
		p.schemaEncoding(named.Header.Schema, headerPath+"/schema", schemaEncodedValue, make(map[schemaEncodingOccurrence]bool))
		p.schema(named.Header.Schema, headerPath+"/schema", objectField)
		p.admitParameter(Parameter{Name: named.Name, In: "header", Schema: named.Header.Schema, Serialization: named.Header.Serialization, AllowReserved: named.Header.AllowReserved}, headerPath)
	}
}

func (p *documentPlanner) content(
	contentType string,
	schema *Schema,
	path string,
	allowNonJSON bool,
	role objectConstructorRole,
) {
	if schema == nil {
		if contentType != "" {
			p.unsupported("content-schema", path+"/content", "content type has no schema")
		}
		return
	}
	encoding := schemaEncodedValue
	if allowNonJSON && contentType != "" && !isJSONMediaType(contentType) {
		encoding = schemaRawBody
	}
	p.schemaEncoding(schema, path+"/content/schema", encoding, make(map[schemaEncodingOccurrence]bool))
	if contentType == "" || !allowNonJSON && !isJSONMediaType(contentType) {
		p.unsupported("media-type", path+"/content", fmt.Sprintf("content type %q is not renderable", contentType))
	}
	p.schema(schema, path+"/content/schema", role)
}

func operationResponseBodyMode(operation *Operation) responseBodyMode {
	successClass := successResponseClass(operation.Responses)
	for _, response := range operation.Responses {
		if !isSuccessResponseStatus(response.Status, successClass) || response.Response.Schema == nil ||
			isJSONMediaType(response.Response.ContentType) || len(response.Response.Schema.MediaTypes) > 0 {
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

func (p *documentPlanner) schema(schema *Schema, path string, role objectConstructorRole) {
	if !p.schemaAdmission(schema, path, role) {
		return
	}
	if schema != nil && len(schema.OneOf) > 0 && len(schema.MediaTypes) == 0 {
		p.admitUntaggedOneOf(schema, path)
	}
	renderer := renderer{document: p.document, schemas: p.schemas}
	_, _, err := renderer.schemaExpression(schema, path)
	if err == nil {
		err = renderer.schemaBlock(schema, path, false, nil)
	}
	if err != nil {
		p.unsupported("schema", path, strings.TrimPrefix(err.Error(), "render OpenAPI design: "))
	}
}

// schemaAdmission checks retained source distinctions before rendering can
// reduce them. It also visits inline occurrences that the renderer would
// otherwise turn into attributes or collection elements.
func (p *documentPlanner) schemaAdmission(schema *Schema, path string, role objectConstructorRole) bool {
	if schema == nil {
		return true
	}
	admitted := true
	if objectNeedsName(schema, role) {
		p.unsupported("schema-object-constructor", path, "inline object requires a named type in this constructor; analyze source schemas before rendering")
		admitted = false
	}
	for _, diagnostic := range []*Diagnostic{schema.valueDomainError, schema.referenceError, schema.patternError, schema.compositionError} {
		if diagnostic != nil {
			p.unsupported(diagnostic.Code, diagnostic.Path, diagnostic.Message)
			return false
		}
	}
	if schema.Boolean != nil && (!*schema.Boolean || strings.HasPrefix(p.document.OpenAPIVersion, "3.0.")) {
		p.unsupported("schema-boolean", path, "only true boolean schemas in OpenAPI 3.1 and 3.2 are in the strict import subset")
		admitted = false
	}
	if schema.LegacyNullable != nil && !strings.HasPrefix(p.document.OpenAPIVersion, "3.0.") {
		p.unsupported("schema-nullable-version", path+"/nullable", "nullable is an OpenAPI 3.0 keyword; use a supported null type union in OpenAPI 3.1 and 3.2")
		admitted = false
	}
	if len(schema.TypeUnion) > 0 && strings.HasPrefix(p.document.OpenAPIVersion, "3.0.") {
		p.unsupported("schema-type-union", path+"/type", "type arrays require the OpenAPI 3.1 or 3.2 schema dialect")
		admitted = false
	}
	if schema.unsupportedComposition {
		p.unsupported("schema-composition", path, "this composition cannot be normalized without losing schema information")
		admitted = false
	}
	for _, property := range schema.Properties {
		if !p.schemaAdmission(property.Schema, path+"/properties/"+escapeJSONPointer(property.Name), objectField) {
			admitted = false
		}
	}
	if !p.schemaAdmission(schema.Items, path+"/items", objectArrayItem) {
		admitted = false
	}
	if schema.AdditionalProperties != nil && !p.schemaAdmission(schema.AdditionalProperties.Schema, path+"/additionalProperties", objectMapValue) {
		admitted = false
	}
	for index, branch := range schema.OneOf {
		if !p.schemaAdmission(branch, fmt.Sprintf("%s/oneOf/%d", path, index), objectUnionBranch) {
			admitted = false
		}
	}
	for index, base := range schema.Bases {
		if !p.schemaAdmission(base, fmt.Sprintf("%s/bases/%d", path, index), objectDeclaration) {
			admitted = false
		}
	}
	return admitted
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
	name, err := openapiref.Name(parameter.Ref, "#/components/parameters/")
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
		resolved := named.Parameter
		if parameter.ReferenceDescription != nil {
			resolved.Description = *parameter.ReferenceDescription
		}
		return resolved, name, nil
	}
	return Parameter{}, "", fmt.Errorf("parameter reference %q does not resolve", parameter.Ref)
}
