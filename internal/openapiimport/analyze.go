package openapiimport

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	yaml3 "gopkg.in/yaml.v3"
)

// ErrUnsupportedVersion indicates that the input is not an OpenAPI 3.0, 3.1, or 3.2
// document.
var ErrUnsupportedVersion = errors.New("unsupported OpenAPI version")

// Analyze parses source without writing to the filesystem and returns a
// normalized model plus every unsupported-feature diagnostic it can discover.
// Syntax, model-building, and version failures are returned as errors.
func Analyze(source []byte) (*Document, Diagnostics, error) {
	document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
	return document, diagnostics, err
}

// AnalyzeSelected parses source and retains operations that match selection.
// It returns tag counts and paths that do not match the requested tags.
func AnalyzeSelected(source []byte, selection Selection) (*Document, Diagnostics, SelectionReport, error) {
	document, diagnostics, report, err := analyzeSelectedDocument(source, selection)
	if err != nil || document == nil {
		return document, diagnostics, report, err
	}
	diagnostics = append(diagnostics, planDocument(document).diagnostics...)
	return document, diagnostics.sorted(), report, nil
}

func analyzeSelectedDocument(source []byte, selection Selection) (*Document, Diagnostics, SelectionReport, error) {
	if err := selection.Validate(); err != nil {
		return nil, nil, SelectionReport{}, err
	}
	parsed, err := libopenapi.NewDocument(source)
	if err != nil {
		return nil, nil, SelectionReport{}, fmt.Errorf("parse OpenAPI document: %w", err)
	}
	version := parsed.GetVersion()
	if !renderableVersion(version) {
		return nil, nil, SelectionReport{}, fmt.Errorf("%w %q; expected 3.0.x, 3.1.x, or 3.2.x", ErrUnsupportedVersion, version)
	}

	diagnostics, err := externalReferenceDiagnostics(source)
	if err != nil {
		return nil, nil, SelectionReport{}, fmt.Errorf("inspect OpenAPI references: %w", err)
	}
	if len(diagnostics) > 0 {
		return nil, diagnostics.sorted(), SelectionReport{}, nil
	}

	model, err := parsed.BuildV3Model()
	if err != nil {
		return nil, nil, SelectionReport{}, fmt.Errorf("build OpenAPI %s model: %w", version, err)
	}
	analyzer := analyzer{diagnostics: diagnostics, selection: selection}
	document := analyzer.document(&model.Model)
	if selection.Active() {
		closure := pruneComponents(document)
		analyzer.diagnostics = filterSelectionDiagnostics(analyzer.diagnostics, closure)
	}
	return document, analyzer.diagnostics.sorted(), analyzer.report, nil
}

type analyzer struct {
	diagnostics Diagnostics
	selection   Selection
	report      SelectionReport
}

var concreteHTTPStatus = regexp.MustCompile(`^[1-5][0-9]{2}$`)

func renderableVersion(version string) bool {
	return strings.HasPrefix(version, "3.0.") || strings.HasPrefix(version, "3.1.") || strings.HasPrefix(version, "3.2.")
}

func externalReferenceDiagnostics(source []byte) (Diagnostics, error) {
	var root yaml3.Node
	if err := yaml3.Unmarshal(source, &root); err != nil {
		return nil, err
	}
	var diagnostics Diagnostics
	walkReferences(&root, "#", &diagnostics)
	return diagnostics, nil
}

func walkReferences(node *yaml3.Node, path string, diagnostics *Diagnostics) {
	if node == nil {
		return
	}
	if node.Kind == yaml3.DocumentNode && len(node.Content) > 0 {
		walkReferences(node.Content[0], path, diagnostics)
		return
	}
	if node.Kind == yaml3.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			childPath := path + "/" + escapeJSONPointer(key.Value)
			if key.Value == "$ref" && value.Kind == yaml3.ScalarNode && !strings.HasPrefix(value.Value, "#/") {
				diagnostics.add("external-reference", childPath, fmt.Sprintf("external reference %q is not supported", value.Value))
			}
			walkReferences(value, childPath, diagnostics)
		}
		return
	}
	for i, child := range node.Content {
		walkReferences(child, fmt.Sprintf("%s/%d", path, i), diagnostics)
	}
}

func escapeJSONPointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func (a *analyzer) document(source *v3.Document) *Document {
	document := &Document{OpenAPIVersion: source.Version}
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
	if len(source.Security) > 0 {
		a.unsupported("security", "#/security", "security requirements are not in the strict import subset")
	}
	if source.ExternalDocs != nil {
		a.unsupported("external-docs", "#/externalDocs", "external documentation is not in the strict import subset")
	}
	if source.JsonSchemaDialect != "" || source.Self != "" {
		a.unsupported("document-identity", "#", "$self and jsonSchemaDialect are not in the strict import subset")
	}
	for _, tag := range source.Tags {
		if tag == nil {
			continue
		}
		document.Tags = append(document.Tags, tag.Name)
		if tag.Summary != "" || tag.Description != "" || tag.ExternalDocs != nil || tag.Parent != "" || tag.Kind != "" || orderedmap.Len(tag.Extensions) > 0 {
			a.unsupported("tag-metadata", "#/tags/"+escapeJSONPointer(tag.Name), "tag metadata beyond its name is not in the strict import subset")
		}
	}
	if orderedmap.Len(source.Webhooks) > 0 {
		a.unsupported("webhooks", "#/webhooks", "webhooks are not in the strict import subset")
	}
	document.Extensions = a.extensions("#", source.Extensions)
	document.Components = a.components(source.Components)
	document.Operations = a.operations(source.Paths, document.Components, document.Tags)
	assignOperationNames(document.Operations)
	return document
}

func (a *analyzer) components(source *v3.Components) Components {
	if source == nil {
		return Components{}
	}
	var result Components
	for name, schema := range source.Schemas.FromOldest() {
		result.Schemas = append(result.Schemas, NamedSchema{
			Name:   name,
			Schema: a.schema(schema, "#/components/schemas/"+escapeJSONPointer(name)),
		})
	}
	assignSchemaNames(result.Schemas)
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
	if orderedmap.Len(source.SecuritySchemes) > 0 {
		a.unsupported("security-schemes", "#/components/securitySchemes", "security schemes are not in the strict import subset")
	}
	if orderedmap.Len(source.Examples) > 0 || orderedmap.Len(source.Links) > 0 || orderedmap.Len(source.Callbacks) > 0 ||
		orderedmap.Len(source.PathItems) > 0 || orderedmap.Len(source.MediaTypes) > 0 {
		a.unsupported("component-kind", "#/components", "examples, links, callbacks, path items, and media types are not in the strict import subset")
	}
	a.unsupportedExtensions("#/components", source.Extensions)
	return result
}

func (a *analyzer) operation(method, path string, inherited []*v3.Parameter, source *v3.Operation, pointer, inheritedPath string, components Components) Operation {
	operation := Operation{
		Method: method, Path: path, OperationID: source.OperationId, Summary: source.Summary,
		Description: source.Description, Tags: append([]string(nil), source.Tags...),
		Deprecated: source.Deprecated != nil && *source.Deprecated,
		Extensions: a.extensions(pointer, source.Extensions),
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
	if len(source.Security) > 0 {
		a.unsupported("security", pointer+"/security", "operation security is not in the strict import subset")
	}
	if len(source.Servers) > 0 {
		a.unsupported("servers", pointer+"/servers", "operation servers are not in the strict import subset")
	}
	if source.ExternalDocs != nil {
		a.unsupported("external-docs", pointer+"/externalDocs", "external documentation is not in the strict import subset")
	}
	return operation
}

type parameterOccurrence struct {
	parameter Parameter
	path      string
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
		return Parameter{Ref: reference}
	}
	if low := source.GoLow(); low != nil && low.IsReference() {
		return Parameter{Ref: low.GetReference()}
	}
	parameter := Parameter{
		Name: source.Name, In: source.In, Description: source.Description,
		Required: source.Required != nil && *source.Required, Deprecated: source.Deprecated, AllowEmptyValue: source.AllowEmptyValue,
		Schema:     a.schema(source.Schema, path+"/schema"),
		Extensions: a.extensions(path, source.Extensions),
	}
	if source.In != "path" && source.In != "query" && source.In != "header" && source.In != "cookie" {
		a.unsupported("parameter-location", path+"/in", fmt.Sprintf("parameter location %q is not supported", source.In))
	}
	if orderedmap.Len(source.Content) > 0 {
		a.unsupported("parameter-content", path+"/content", "content-based parameters are not in the strict import subset")
	}
	if source.Style != "" || source.Explode != nil || source.AllowReserved {
		a.unsupported("parameter-serialization", path, "custom parameter serialization is not in the strict import subset")
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
	if reference := source.Reference; reference != "" {
		return RequestBody{Ref: reference}
	}
	if low := source.GoLow(); low != nil && low.IsReference() {
		return RequestBody{Ref: low.GetReference()}
	}
	contentType, schema, examples := a.content(source.Content, path+"/content")
	body := RequestBody{
		Description: source.Description, Required: source.Required != nil && *source.Required,
		ContentType: contentType, Schema: schema, Examples: examples,
		Extensions: a.extensions(path, source.Extensions),
	}
	return body
}

func (a *analyzer) response(source *v3.Response, path string) Response {
	if source == nil {
		return Response{}
	}
	if reference := source.Reference; reference != "" {
		return Response{Ref: reference}
	}
	if low := source.GoLow(); low != nil && low.IsReference() {
		return Response{Ref: low.GetReference()}
	}
	contentType, schema, examples := a.content(source.Content, path+"/content")
	response := Response{
		Description: source.Description,
		ContentType: contentType,
		Schema:      schema,
		Examples:    examples,
		Extensions:  a.extensions(path, source.Extensions),
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
	if source.Summary != "" {
		a.unsupported("response-summary", path+"/summary", "response summaries are not in the strict import subset")
	}
	return response
}

func (a *analyzer) header(source *v3.Header, path string) Header {
	if source == nil {
		return Header{}
	}
	if reference := source.Reference; reference != "" {
		return Header{Ref: reference}
	}
	if low := source.GoLow(); low != nil && low.IsReference() {
		return Header{Ref: low.GetReference()}
	}
	header := Header{
		Description: source.Description, Required: source.Required,
		Deprecated: source.Deprecated, Schema: a.schema(source.Schema, path+"/schema"),
	}
	if orderedmap.Len(source.Content) > 0 {
		a.unsupported("header-content", path+"/content", "content-based headers are not in the strict import subset")
	}
	if source.Style != "" || source.Explode || source.AllowEmptyValue || source.AllowReserved {
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
	if orderedmap.Len(content) != 1 {
		a.unsupported("multiple-media-types", path, "exactly one media type is supported")
		return "", nil, nil
	}
	for contentType, media := range content.FromOldest() {
		if media == nil {
			return contentType, nil, nil
		}
		mediaPath := path + "/" + escapeJSONPointer(contentType)
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
	if proxy.IsReference() {
		return &Schema{Ref: proxy.GetReference()}
	}
	source := proxy.Schema()
	if source == nil {
		a.unsupported("empty-schema", path, "schema could not be resolved")
		return &Schema{}
	}
	schema := newNormalizedSchema(source)
	supportedNullableComposition := a.schemaNullableAnyOf(schema, source, path)
	supportedAllOf := a.schemaAllOf(schema, source, path)
	a.schemaTypeAndProperties(schema, source, path)
	a.schemaCollections(schema, source, path)
	a.schemaEnum(schema, source, path)
	a.schemaExclusiveBounds(schema, source)
	a.schemaDefault(schema, source, path)
	a.schemaExamples(schema, source, path)
	a.schemaFormat(schema, path)
	a.schemaUnsupportedKeywords(schema, source, path, supportedAllOf, supportedNullableComposition)
	schema.Extensions = a.extensions(path, source.Extensions)
	return schema
}

func newNormalizedSchema(source *base.Schema) *Schema {
	return &Schema{
		Title: source.Title, Description: source.Description, Format: source.Format,
		Pattern: source.Pattern, Minimum: source.Minimum, Maximum: source.Maximum,
		MinLength: source.MinLength, MaxLength: source.MaxLength,
		MinItems: source.MinItems, MaxItems: source.MaxItems,
		Required:   append([]string(nil), source.Required...),
		Deprecated: source.Deprecated != nil && *source.Deprecated,
		ReadOnly:   source.ReadOnly != nil && *source.ReadOnly,
		WriteOnly:  source.WriteOnly != nil && *source.WriteOnly,
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
		if nullableType, ok := nullableUnionType(source.Type); ok {
			schema.Type = nullableType
			schema.Nullable = true
		} else {
			a.unsupported("schema-type-union", path+"/type", "only a two-member union containing one concrete type and null is supported")
		}
	}
	if source.Nullable != nil && *source.Nullable {
		schema.Nullable = true
	}
	if schema.Type == "" && orderedmap.Len(source.Properties) > 0 {
		schema.Type = "object"
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

func (a *analyzer) schemaEnum(schema *Schema, source *base.Schema, path string) {
	for _, value := range source.Enum {
		var decoded any
		if err := value.Decode(&decoded); err != nil {
			a.unsupported("enum-value", path+"/enum", fmt.Sprintf("enum value cannot be decoded: %v", err))
			continue
		}
		schema.Enum = append(schema.Enum, decoded)
	}
}

func (a *analyzer) schemaExclusiveBounds(schema *Schema, source *base.Schema) {
	if source.ExclusiveMinimum != nil {
		if source.ExclusiveMinimum.IsB() {
			value := source.ExclusiveMinimum.B
			schema.ExclusiveMinimum = &value
		} else if source.ExclusiveMinimum.A && source.Minimum != nil {
			value := *source.Minimum
			schema.ExclusiveMinimum = &value
			schema.Minimum = nil
		}
	}
	if source.ExclusiveMaximum != nil {
		if source.ExclusiveMaximum.IsB() {
			value := source.ExclusiveMaximum.B
			schema.ExclusiveMaximum = &value
		} else if source.ExclusiveMaximum.A && source.Maximum != nil {
			value := *source.Maximum
			schema.ExclusiveMaximum = &value
			schema.Maximum = nil
		}
	}
}

// schemaDefault decodes the JSON Schema default keyword and retains it when
// the schema is one of the scalar types Loom's Default DSL can express and
// the decoded value matches that type. Composite-type defaults (object,
// array) and type-mismatched defaults remain in the strict import subset's
// generic "schema-keyword" diagnostic rather than risk emitting a design that
// fails to evaluate.
func (a *analyzer) schemaDefault(schema *Schema, source *base.Schema, path string) {
	if source.Default == nil {
		return
	}
	var decoded any
	if err := source.Default.Decode(&decoded); err != nil {
		a.unsupported("schema-keyword", path, fmt.Sprintf("default value cannot be decoded: %v", err))
		return
	}
	if !defaultCompatibleWithType(schema.Type, decoded) {
		a.unsupported("schema-keyword", path, fmt.Sprintf("default value is not compatible with the strict import subset for schema type %q", schema.Type))
		return
	}
	schema.Default = &SchemaDefault{Value: decoded}
}

// defaultCompatibleWithType reports whether a decoded default value can be
// rendered as a Loom Default(...) call for the given normalized schema type.
// Only scalar types are supported; object and array defaults, and an
// explicit null default on any type, are left unsupported.
func defaultCompatibleWithType(schemaType string, value any) bool {
	if value == nil {
		return false
	}
	switch schemaType {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		switch value.(type) {
		case int, int64, uint64:
			return true
		}
		return false
	case "number":
		switch value.(type) {
		case int, int64, uint64, float32, float64:
			return true
		}
		return false
	default:
		return false
	}
}

// schemaFormat reports an unrecognized string, integer, or number format as a
// lossy-allowed diagnostic. An empty format is treated as absent (OpenAPI 3.1
// permits arbitrary format values, and JSON Schema requires that unknown
// formats never fail validation or processing), so it never reports here;
// render_schema.go falls back to the type's widest unformatted representation
// for anything not recognized.
func (a *analyzer) schemaFormat(schema *Schema, path string) {
	if schema.Format == "" {
		return
	}
	switch schema.Type {
	case "string":
		if schema.Format == "byte" || schema.Format == "binary" {
			return
		}
		if _, ok := stringFormatDSL(schema.Format); !ok {
			a.unsupported("schema-format", path, fmt.Sprintf("string format %q is not a recognized Loom format; rendering without a format validation", schema.Format))
		}
	case "integer":
		if schema.Format != "int32" && schema.Format != "int64" {
			a.unsupported("schema-format", path, fmt.Sprintf("integer format %q is not a recognized Loom format; rendering as an unformatted integer", schema.Format))
		}
	case "number":
		if schema.Format != "float" && schema.Format != "double" {
			a.unsupported("schema-format", path, fmt.Sprintf("number format %q is not a recognized Loom format; rendering as Float64", schema.Format))
		}
	}
}

func (a *analyzer) schemaUnsupportedKeywords(
	schema *Schema,
	source *base.Schema,
	path string,
	supportedAllOf bool,
	supportedNullableComposition bool,
) {
	if len(source.AllOf) > 0 && !supportedAllOf || len(source.OneOf) > 0 ||
		len(source.AnyOf) > 0 && !supportedNullableComposition || source.Not != nil {
		schema.unsupportedComposition = true
	}
	if len(source.PrefixItems) > 0 || source.Contains != nil || source.If != nil || source.Then != nil || source.Else != nil ||
		orderedmap.Len(source.DependentSchemas) > 0 || orderedmap.Len(source.DependentRequired) > 0 ||
		orderedmap.Len(source.PatternProperties) > 0 || source.PropertyNames != nil || source.UnevaluatedProperties != nil {
		a.unsupported("advanced-schema", path, "advanced JSON Schema applicators are not in the strict import subset")
	}
	if source.MultipleOf != nil || source.UniqueItems != nil || source.MaxProperties != nil || source.MinProperties != nil ||
		source.Const != nil ||
		source.ContentEncoding != "" || source.ContentMediaType != "" || source.XML != nil || source.ExternalDocs != nil {
		a.unsupported("schema-keyword", path, "one or more schema keywords are not in the strict import subset")
	}
	if source.DynamicRef != "" || source.Anchor != "" || source.DynamicAnchor != "" || source.SchemaTypeRef != "" ||
		source.Id != "" || source.Comment != "" || source.ContentSchema != nil || orderedmap.Len(source.Defs) > 0 ||
		orderedmap.Len(source.Vocabulary) > 0 || source.UnevaluatedItems != nil {
		a.unsupported("schema-resource", path, "JSON Schema resource and dialect keywords are not in the strict import subset")
	}
}

func (a *analyzer) unsupported(code, path, message string) {
	a.diagnostics.add(code, path, message)
}
