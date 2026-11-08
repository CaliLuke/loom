package openapiimport

import (
	"encoding/json/jsontext"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/internal/httpencoding"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

type analyzer struct {
	bundle           *bundledDescription
	diagnostics      Diagnostics
	selection        Selection
	report           SelectionReport
	examples         map[string]Example
	schemas          map[string]*Schema
	version          string
	numbers          map[string]numericSource
	numericPositions map[numericPosition]numericSource
	unionPrepared    map[*Schema]bool
	unionSiblings    []oneOfSiblingCheck
}

var concreteHTTPStatus = regexp.MustCompile(`^[1-5][0-9]{2}$`)

func renderableVersion(version string) bool {
	return strings.HasPrefix(version, "3.0.") || strings.HasPrefix(version, "3.1.") || strings.HasPrefix(version, "3.2.")
}

func (a *analyzer) openAPI32() bool {
	return strings.HasPrefix(a.version, "3.2.")
}

func escapeJSONPointer(value string) string {
	return string(jsontext.Pointer("").AppendToken(value))[1:]
}

func (a *analyzer) document(source *v3.Document) *Document {
	document := &Document{OpenAPIVersion: source.Version, bundle: a.bundle}
	if source.Info != nil {
		document.Title = source.Info.Title
		document.Description = source.Info.Description
		document.APIVersion = source.Info.Version
		if source.Info.Summary != "" || source.Info.TermsOfService != "" || source.Info.Contact != nil || source.Info.License != nil {
			a.unsupported("info-metadata", "#/info", "summary, terms, contact, and license metadata are not in the strict import subset")
		}
		a.unsupportedExtensions("#/info", source.Info.Extensions)
	}
	if len(source.Servers) > 0 {
		a.unsupported("servers", "#/servers", "server definitions are not in the strict import subset")
	}
	if source.ExternalDocs != nil {
		a.unsupported("external-docs", "#/externalDocs", "external documentation is not in the strict import subset")
	}
	if source.JsonSchemaDialect != "" {
		a.unsupported("json-schema-dialect", "#/jsonSchemaDialect", "custom JSON Schema dialects are not in the strict import subset")
	}
	hasSelf := source.Self != ""
	if a.bundle != nil {
		_, err := descriptionNode(a.bundle.graph.registry.docs["entry"].root, "/$self")
		hasSelf = err == nil
	}
	if hasSelf {
		if a.openAPI32() {
			self := source.Self
			document.Self = &self
		} else {
			a.unsupported("versioned-field", "#/$self", "$self requires OpenAPI 3.2")
		}
	}
	document.Tags, document.TagMetadata = a.tags(source.Tags)
	if orderedmap.Len(source.Webhooks) > 0 {
		a.unsupported("webhooks", "#/webhooks", "webhooks are not in the strict import subset")
	}
	document.Extensions = a.extensions("#", source.Extensions)
	document.Components = a.components(source.Components)
	document.SecurityDefined = source.GoLow() != nil && !source.GoLow().Security.IsEmpty()
	if document.SecurityDefined {
		document.Security = a.projectSecurityKeys(source.Security, "#/security", document.Components)
	}
	document.Operations = a.operations(source.Paths, document.Components, document.Tags)
	a.validateOperationIdentities(document.Operations)
	assignOperationNames(document.Operations)
	a.prepareObjectConstructors(document)
	if err := allocateDesignNames(document); err != nil {
		// Admission diagnostics already explain invalid source constructs. Only
		// expose an additional planning failure when no fatal cause was found.
		if fatal, _ := a.diagnostics.Classify(true); len(fatal) == 0 {
			a.diagnostics = append(a.diagnostics, designNameDiagnostics(err)...)
		}
	}
	a.retainMemberExamples(document)
	return document
}

func (a *analyzer) components(source *v3.Components) Components {
	if source == nil {
		return Components{}
	}
	var result Components
	for name, example := range source.Examples.FromOldest() {
		if normalized, ok := a.componentExample(name, example, "#/components/examples/"+escapeJSONPointer(name)); ok {
			a.examples[name] = normalized
		}
	}
	result.Schemas = a.componentSchemas(source)
	securityDescriptions := a.securitySchemeDescriptions(source.SecuritySchemes)
	for name, scheme := range source.SecuritySchemes.FromOldest() {
		path := "#/components/securitySchemes/" + escapeJSONPointer(name)
		if normalized, ok := a.securityScheme(name, scheme, path, securityDescriptions[name]); ok {
			normalized.origin = a.bundle.declarationOrigin(path)
			if a.bundle != nil {
				normalized.URI = a.bundle.securityURIs[name]
			}
			result.SecuritySchemes = append(result.SecuritySchemes, normalized)
		}
	}
	for name, parameter := range source.Parameters.FromOldest() {
		result.Parameters = append(result.Parameters, NamedParameter{
			Name:      name,
			Parameter: a.parameter(parameter, "#/components/parameters/"+escapeJSONPointer(name)),
		})
	}
	for name, body := range source.RequestBodies.FromOldest() {
		result.RequestBodies = append(result.RequestBodies, NamedRequestBody{
			Name:        name,
			RequestBody: a.requestBody(body, "#/components/requestBodies/"+escapeJSONPointer(name)),
		})
	}
	for name, response := range source.Responses.FromOldest() {
		result.Responses = append(result.Responses, NamedResponse{
			Name:     name,
			Response: a.response(response, "#/components/responses/"+escapeJSONPointer(name)),
		})
	}
	for name, header := range source.Headers.FromOldest() {
		result.Headers = append(result.Headers, NamedHeader{
			Name:   name,
			Header: a.header(header, "#/components/headers/"+escapeJSONPointer(name)),
		})
	}
	if orderedmap.Len(source.Links) > 0 || orderedmap.Len(source.Callbacks) > 0 ||
		orderedmap.Len(source.PathItems) > 0 || orderedmap.Len(source.MediaTypes) > 0 {
		a.unsupported("component-kind", "#/components", "links, callbacks, path items, and media types are not in the strict import subset")
	}
	a.unsupportedExtensions("#/components", source.Extensions)
	return result
}

func (a *analyzer) componentSchemas(source *v3.Components) []NamedSchema {
	result := make([]NamedSchema, 0, orderedmap.Len(source.Schemas))
	for name, schema := range source.Schemas.FromOldest() {
		path := "#/components/schemas/" + escapeJSONPointer(name)
		a.schemas[name] = a.schema(schema, path)
		result = append(result, NamedSchema{Name: name, Schema: a.schemas[name], origin: a.bundle.declarationOrigin(path)})
	}
	return result
}

func (a *analyzer) operation(method, path string, inherited []*v3.Parameter, source *v3.Operation, pointer, inheritedPath string, components Components) Operation {
	operation := Operation{
		sourcePath: pointer,
		origin:     a.bundle.declarationOrigin(pointer),
		Method:     method, Path: path, OperationID: source.OperationId, Summary: source.Summary,
		Description: source.Description, Tags: append([]string(nil), source.Tags...),
		Deprecated: source.Deprecated != nil && *source.Deprecated,
		Extensions: a.extensions(pointer, source.Extensions),
	}
	if source.Security != nil {
		operation.SecurityDefined = true
		operation.Security = a.projectSecurityKeys(source.Security, pointer+"/security", components)
	}
	operation.Parameters = a.mergeParameters(inherited, source.Parameters, inheritedPath, pointer+"/parameters", components)
	if source.RequestBody != nil {
		body := a.requestBody(source.RequestBody, pointer+"/requestBody")
		operation.RequestBody = &body
	}
	if source.Responses != nil {
		for status, response := range source.Responses.Codes.FromOldest() {
			if !concreteHTTPStatus.MatchString(status) {
				a.unsupported("response-status", pointer+"/responses/"+escapeJSONPointer(status), "response status must be a concrete three-digit HTTP code")
			}
			operation.Responses = append(operation.Responses, StatusResponse{
				Status: status, Response: a.response(response, pointer+"/responses/"+escapeJSONPointer(status)),
			})
		}
		sort.Slice(operation.Responses, func(i, j int) bool { return operation.Responses[i].Status < operation.Responses[j].Status })
		if source.Responses.Default != nil {
			a.unsupported("default-response", pointer+"/responses/default", "default responses are not in the strict import subset")
		}
		a.unsupportedExtensions(pointer+"/responses", source.Responses.Extensions)
	}
	if orderedmap.Len(source.Callbacks) > 0 {
		a.unsupported("callbacks", pointer+"/callbacks", "callbacks are not in the strict import subset")
	}
	if len(source.Servers) > 0 {
		a.unsupported("servers", pointer+"/servers", "operation servers are not in the strict import subset")
	}
	if source.ExternalDocs != nil {
		a.unsupported("external-docs", pointer+"/externalDocs", "external documentation is not in the strict import subset")
	}
	return operation
}

func (a *analyzer) mergeParameters(inherited, operation []*v3.Parameter, inheritedPath, operationPath string, components Components) []Parameter {
	merged := make([]parameterOccurrence, 0, len(inherited)+len(operation))
	indices := make(map[string]int, len(inherited)+len(operation))
	appendParameter := func(parameter *v3.Parameter, path string) {
		occurrence := parameterOccurrence{parameter: a.parameter(parameter, path), path: path}
		key, ok := parameterKey(occurrence.parameter, components)
		if !ok {
			merged = append(merged, occurrence)
			return
		}
		if index, exists := indices[key]; exists {
			a.unsupported("duplicate-parameter", path, "parameters with the same name and location must not be repeated")
			merged[index] = occurrence
			return
		}
		indices[key] = len(merged)
		merged = append(merged, occurrence)
	}
	for index, parameter := range inherited {
		appendParameter(parameter, fmt.Sprintf("%s/%d", inheritedPath, index))
	}
	operationKeys := make(map[string]struct{}, len(operation))
	for index, parameter := range operation {
		path := fmt.Sprintf("%s/%d", operationPath, index)
		occurrence := parameterOccurrence{parameter: a.parameter(parameter, path), path: path}
		key, ok := parameterKey(occurrence.parameter, components)
		if !ok {
			merged = append(merged, occurrence)
			continue
		}
		if _, exists := operationKeys[key]; exists {
			a.unsupported("duplicate-parameter", occurrence.path, "parameters with the same name and location must not be repeated")
		}
		operationKeys[key] = struct{}{}
		if mergedIndex, exists := indices[key]; exists {
			merged[mergedIndex] = occurrence
			continue
		}
		indices[key] = len(merged)
		merged = append(merged, occurrence)
	}
	result := make([]Parameter, len(merged))
	for index := range merged {
		result[index] = merged[index].parameter
	}
	return result
}

func (a *analyzer) parameter(source *v3.Parameter, path string) Parameter {
	if source == nil {
		return Parameter{}
	}
	if reference := source.Reference; reference != "" {
		return a.parameterReference(source, reference)
	}
	if low := source.GoLow(); low != nil && low.IsReference() {
		return a.parameterReference(source, low.GetReference())
	}
	parameter := Parameter{
		Name: source.Name, In: source.In, Description: source.Description,
		Required: source.Required != nil && *source.Required, Deprecated: source.Deprecated, AllowEmptyValue: source.AllowEmptyValue,
		Style: source.Style, AllowReserved: source.AllowReserved,
		Schema: a.schema(source.Schema, path+"/schema"), Extensions: a.extensions(path, source.Extensions),
	}
	if !a.openAPI32() && source.Style == "cookie" {
		a.unsupported("versioned-field", path+"/style", "cookie parameter style requires OpenAPI 3.2")
	}
	if !a.openAPI32() && source.AllowReserved && source.In != "query" {
		a.unsupported("versioned-field", path+"/allowReserved", "allowReserved outside query parameters requires OpenAPI 3.2")
	}
	if source.In != "path" && source.In != "query" && source.In != "header" && source.In != "cookie" {
		a.unsupported("parameter-location", path+"/in", fmt.Sprintf("parameter location %q is not supported", source.In))
	}
	parameter.Serialization.Style, parameter.Serialization.Explode = source.Style, source.Explode
	if source.AllowReserved || source.GoLow() != nil && !source.GoLow().AllowReserved.IsEmpty() {
		parameter.Serialization.AllowReserved = &source.AllowReserved
	}
	if orderedmap.Len(source.Content) > 0 {
		if source.In != "query" || source.Schema != nil {
			a.unsupported("parameter-content", path+"/content", "content is supported only for named JSON query objects, without schema siblings")
		}
		media, schema, examples := a.content(source.Content, path+"/content")
		if len(examples) > 0 {
			a.unsupported("examples", path+"/content", "parameter content examples are not in the strict import subset")
		}
		parameter.Serialization.MediaType, parameter.Schema = media, schema
	}
	if location, ok := parameterLocation(parameter.In); ok {
		parameter.Serialization = httpencoding.NormalizeOpenAPIParameter(location, parameter.Serialization)
	}
	if source.Example != nil || orderedmap.Len(source.Examples) > 0 {
		a.unsupported("examples", path, "parameter examples are not in the strict import subset")
	}
	if parameter.Deprecated {
		a.unsupported("parameter-deprecated", path, "the HTTP DSL has no per-parameter deprecated marker; the flag is omitted from the rendered design")
	}
	return parameter
}

func (a *analyzer) requestBody(source *v3.RequestBody, path string) RequestBody {
	if source == nil {
		return RequestBody{}
	}
	reference := source.Reference
	var referenceNode *yaml.Node
	if low := source.GoLow(); low != nil {
		if reference == "" && low.IsReference() {
			reference = low.GetReference()
		}
		referenceNode = low.GetReferenceNode()
	}
	if reference != "" {
		description, summary := a.referenceAnnotations(referenceNode)
		return RequestBody{Ref: reference, ReferenceDescription: description, ReferenceSummary: summary}
	}
	contentTypes, schema, examples := a.requestContent(source.Content, path+"/content")
	body := RequestBody{
		Description: source.Description, Required: source.Required != nil && *source.Required,
		ContentTypes: contentTypes, Schema: schema, Examples: examples,
		Extensions: a.extensions(path, source.Extensions),
	}
	return body
}

func (a *analyzer) response(source *v3.Response, path string) Response {
	if source == nil {
		return Response{}
	}
	reference := source.Reference
	var referenceNode *yaml.Node
	if low := source.GoLow(); low != nil {
		if reference == "" && low.IsReference() {
			reference = low.GetReference()
		}
		referenceNode = low.GetReferenceNode()
	}
	if reference != "" {
		description, summary := a.referenceAnnotations(referenceNode)
		return Response{Ref: reference, ReferenceDescription: description, ReferenceSummary: summary}
	}
	contentType, schema, examples := a.content(source.Content, path+"/content")
	response := Response{
		Description: source.Description,
		Summary:     source.Summary,
		ContentType: contentType,
		Schema:      schema,
		Examples:    examples,
		Extensions:  a.extensions(path, source.Extensions),
	}
	if !a.openAPI32() && source.Summary != "" {
		a.unsupported("versioned-field", path+"/summary", "response summary requires OpenAPI 3.2")
	}
	for name, header := range source.Headers.FromOldest() {
		response.Headers = append(response.Headers, NamedHeader{
			Name: name, Header: a.header(header, path+"/headers/"+escapeJSONPointer(name)),
		})
	}
	sort.Slice(response.Headers, func(i, j int) bool { return response.Headers[i].Name < response.Headers[j].Name })
	if orderedmap.Len(source.Links) > 0 {
		a.unsupported("response-links", path+"/links", "response links are not in the strict import subset")
	}
	return response
}

func (a *analyzer) header(source *v3.Header, path string) Header {
	if source == nil {
		return Header{}
	}
	reference := source.Reference
	var referenceNode *yaml.Node
	if low := source.GoLow(); low != nil {
		if reference == "" && low.IsReference() {
			reference = low.GetReference()
		}
		referenceNode = low.GetReferenceNode()
	}
	if reference != "" {
		description, summary := a.referenceAnnotations(referenceNode)
		return Header{Ref: reference, ReferenceDescription: description, ReferenceSummary: summary}
	}
	header := Header{
		Description: source.Description, Required: source.Required,
		Deprecated: source.Deprecated, AllowReserved: source.AllowReserved,
		Schema: a.schema(source.Schema, path+"/schema"),
	}
	header.Serialization.Style = source.Style
	if source.Explode || source.GoLow() != nil && !source.GoLow().Explode.IsEmpty() {
		header.Serialization.Explode = &source.Explode
	}
	if source.AllowReserved || source.GoLow() != nil && !source.GoLow().AllowReserved.IsEmpty() {
		header.Serialization.AllowReserved = &source.AllowReserved
	}
	if !a.openAPI32() && source.AllowReserved {
		a.unsupported("versioned-field", path+"/allowReserved", "header allowReserved requires OpenAPI 3.2")
	}
	if orderedmap.Len(source.Content) > 0 {
		a.unsupported("header-content", path+"/content", "content-based headers are not in the strict import subset")
	}
	if source.AllowEmptyValue {
		a.unsupported("header-serialization", path, "custom header serialization is not in the strict import subset")
	}
	if source.Example != nil || orderedmap.Len(source.Examples) > 0 {
		a.unsupported("examples", path, "header examples are not in the strict import subset")
	}
	if header.Deprecated {
		a.unsupported("header-deprecated", path, "the HTTP DSL has no per-header deprecated marker; the flag is omitted from the rendered design")
	}
	a.unsupportedExtensions(path, source.Extensions)
	return header
}

func (a *analyzer) content(content *orderedmap.Map[string, *v3.MediaType], path string) (string, *Schema, []Example) {
	if orderedmap.Len(content) == 0 {
		return "", nil, nil
	}
	if orderedmap.Len(content) > 1 {
		_, schema, _ := a.requestContent(content, path)
		return "", schema, nil
	}
	for contentType, media := range content.FromOldest() {
		if media == nil {
			return contentType, nil, nil
		}
		mediaPath := path + "/" + escapeJSONPointer(contentType)
		a.mediaTypeParserGapDiagnostics(media, mediaPath)
		if media.ItemSchema != nil {
			a.unsupported("media-item-schema", mediaPath+"/itemSchema", "item schemas are not in the strict import subset")
		}
		if orderedmap.Len(media.Encoding) > 0 || orderedmap.Len(media.ItemEncoding) > 0 {
			a.unsupported("media-encoding", mediaPath, "media encodings are not in the strict import subset")
		}
		schema := a.schema(media.Schema, mediaPath+"/schema")
		examples := a.mediaExamples(media, schema, mediaPath)
		a.unsupportedExtensions(mediaPath, media.Extensions)
		return contentType, schema, examples
	}
	return "", nil, nil
}

func isJSONMediaType(contentType string) bool {
	baseType := normalizedMediaType(contentType)
	return baseType == "application/json" || strings.HasSuffix(baseType, "+json")
}

func (a *analyzer) schema(proxy *base.SchemaProxy, path string) *Schema {
	if proxy == nil {
		return nil
	}
	a.schemaUnknownKeywordDiagnostics(proxy, path)
	if proxy.IsReference() {
		return a.schemaReference(proxy, path)
	}
	if node := semanticSchemaNode(proxy); node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!bool" {
		var value bool
		if err := node.Decode(&value); err != nil {
			a.unsupported("schema-boolean", path, fmt.Sprintf("boolean schema could not be decoded: %v", err))
			return &Schema{}
		}
		return &Schema{Boolean: &value, Unconstrained: value}
	}
	source := proxy.Schema()
	if source == nil {
		a.unsupported("empty-schema", path, "schema could not be resolved")
		return &Schema{}
	}
	return a.schemaSource(source, path)
}

func (a *analyzer) schemaSource(source *base.Schema, path string) *Schema {
	schema := newNormalizedSchema(source)
	a.schemaPattern(schema, path)
	a.schemaNumericBounds(schema, source, path)
	supportedNullableComposition := a.schemaNullableAnyOf(schema, source, path)
	supportedUntaggedOneOf := a.schemaUntaggedOneOf(schema, source, path)
	supportedAllOf := a.schemaAllOf(schema, source, path)
	if !supportedUntaggedOneOf {
		a.schemaTypeAndProperties(schema, source, path)
		a.schemaCollections(schema, source, path)
		a.schemaValueDomain(schema, source, path)
	}
	schema.sourceType = sourceEncodingType(source, make(map[*base.Schema]bool))
	a.schemaDefault(schema, source, path)
	a.schemaFormat(schema, path)
	a.schemaUnsupportedKeywords(
		schema,
		source,
		path,
		supportedAllOf,
		supportedNullableComposition,
		supportedUntaggedOneOf,
	)
	// Examples are members of the classified value domain.
	schema.Unconstrained = isUnconstrainedSchema(schema)
	a.schemaExamples(schema, source, path)
	a.schemaAnnotations(schema, source, path)
	return schema
}

func newNormalizedSchema(source *base.Schema) *Schema {
	return &Schema{
		Format:    source.Format,
		Pattern:   source.Pattern,
		MinLength: source.MinLength, MaxLength: source.MaxLength,
		MinItems: source.MinItems, MaxItems: source.MaxItems,
		Required: append([]string(nil), source.Required...),
	}
}

func (a *analyzer) schemaTypeAndProperties(schema *Schema, source *base.Schema, path string) {
	if len(source.Type) == 1 {
		if source.Type[0] == "null" {
			a.unsupported("schema-type", path+"/type", "a null-only schema is not in the strict import subset")
		} else {
			schema.Type = source.Type[0]
		}
	} else if len(source.Type) > 1 {
		schema.TypeUnion = append([]string(nil), source.Type...)
		if nullableType, ok := nullableUnionType(source.Type); ok {
			schema.Type = nullableType
			schema.Nullable = true
		} else {
			a.unsupported("schema-type-union", path+"/type", "only a two-member union containing one concrete type and null is supported")
		}
	}
	if source.Nullable != nil {
		value := *source.Nullable
		schema.LegacyNullable = &value
		if strings.HasPrefix(a.version, "3.0.") && value {
			schema.Nullable = true
		}
	}
	if schema.Type != "" && schema.Type != "object" && schema.Type != "array" && schema.Type != "string" &&
		schema.Type != "integer" && schema.Type != "number" && schema.Type != "boolean" {
		a.unsupported("schema-type", path+"/type", fmt.Sprintf("schema type %q is not supported", schema.Type))
	}
	for name, property := range source.Properties.FromOldest() {
		schema.Properties = append(schema.Properties, NamedProperty{
			Name: name, Schema: a.schema(property, path+"/properties/"+escapeJSONPointer(name)),
		})
	}
	sort.Slice(schema.Properties, func(i, j int) bool { return schema.Properties[i].Name < schema.Properties[j].Name })
	sort.Strings(schema.Required)
}

func (a *analyzer) schemaCollections(schema *Schema, source *base.Schema, path string) {
	if source.Items != nil {
		if source.Items.IsA() {
			schema.Items = a.schema(source.Items.A, path+"/items")
		} else {
			a.unsupported("boolean-items", path+"/items", "boolean items schemas are not in the strict import subset")
		}
	}
	if source.AdditionalProperties != nil {
		schema.AdditionalProperties = &AdditionalProperties{}
		if source.AdditionalProperties.IsA() {
			schema.AdditionalProperties.Schema = a.schema(source.AdditionalProperties.A, path+"/additionalProperties")
		} else {
			allowed := source.AdditionalProperties.B
			schema.AdditionalProperties.Allowed = &allowed
		}
	}
}

func (a *analyzer) unsupported(code, path, message string) {
	a.diagnostics.add(code, path, message)
}
