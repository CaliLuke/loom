package ir

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"goa.design/goa/v3/codegen"
)

const (
	ParameterComponentRefPrefix   = "#/components/parameters/"
	HeaderComponentRefPrefix      = "#/components/headers/"
	RequestBodyComponentRefPrefix = "#/components/requestBodies/"
	ResponseComponentRefPrefix    = "#/components/responses/"
	ExampleComponentRefPrefix     = "#/components/examples/"
)

type (
	exampleUsage struct {
		ref  *ExampleRef
		base string
	}

	headerUsage struct {
		ref  *HeaderRef
		base string
	}

	requestBodyUsage struct {
		ref  *RequestBodyRef
		base string
	}

	responseUsage struct {
		ref    *ResponseRef
		base   string
		status string
	}

	componentUsage[V any, R any] struct {
		ref    R
		base   string
		prefix string
	}
)

func componentizeDocument(doc *Document) {
	if doc == nil {
		return
	}
	if doc.Components == nil {
		doc.Components = &Components{}
	}
	doc.Components.Examples = componentizeExamples(doc.Paths)
	doc.Components.Headers = componentizeHeaders(doc.Paths)
	doc.Components.RequestBodies = componentizeRequestBodies(doc.Paths, doc.Components.Schemas)
	doc.Components.Responses = componentizeResponses(doc.Paths, doc.Components.Schemas)
	doc.Components.Parameters = componentizeParameters(doc.Paths)
}

func componentizeParameters(paths map[string]*PathItem) map[string]*ParameterRef {
	inlineRefs := collectInlineParameters(paths)
	if len(inlineRefs) == 0 {
		return nil
	}

	counts := make(map[string]int)
	for _, ref := range inlineRefs {
		hash, err := parameterHash(ref.Value)
		if err != nil || hash == "" {
			continue
		}
		counts[hash]++
	}

	components := make(map[string]*ParameterRef)
	namesByHash := make(map[string]string)
	hashesByName := make(map[string]string)
	for _, ref := range inlineRefs {
		if ref == nil || ref.Value == nil || ref.Ref != "" {
			continue
		}
		hash, err := parameterHash(ref.Value)
		if err != nil || counts[hash] < 2 {
			continue
		}
		name, ok := namesByHash[hash]
		if !ok {
			name = uniqueParameterComponentName(ref.Value, hash, hashesByName)
			namesByHash[hash] = name
			hashesByName[name] = hash
			components[name] = &ParameterRef{Value: ref.Value}
		}
		ref.Ref = ParameterComponentRefPrefix + name
		ref.Value = nil
	}
	if len(components) == 0 {
		return nil
	}
	return components
}

func componentizeExamples(paths map[string]*PathItem) map[string]*ExampleRef {
	usages := collectExampleUsages(paths)
	return componentizeReusableRefs(
		usages,
		func(usage exampleUsage) componentUsage[Example, *ExampleRef] {
			return componentUsage[Example, *ExampleRef]{ref: usage.ref, base: usage.base, prefix: ExampleComponentRefPrefix}
		},
		func(ref *ExampleRef) *Example {
			if ref == nil {
				return nil
			}
			return ref.Value
		},
		func(ref *ExampleRef, path string) {
			ref.Ref = path
			ref.Value = nil
		},
		func(value *Example) *ExampleRef { return &ExampleRef{Value: value} },
	)
}

func componentizeHeaders(paths map[string]*PathItem) map[string]*HeaderRef {
	usages := collectHeaderUsages(paths)
	return componentizeReusableRefs(
		usages,
		func(usage headerUsage) componentUsage[Header, *HeaderRef] {
			return componentUsage[Header, *HeaderRef]{ref: usage.ref, base: usage.base, prefix: HeaderComponentRefPrefix}
		},
		func(ref *HeaderRef) *Header {
			if ref == nil {
				return nil
			}
			return ref.Value
		},
		func(ref *HeaderRef, path string) {
			ref.Ref = path
			ref.Value = nil
		},
		func(value *Header) *HeaderRef { return &HeaderRef{Value: value} },
	)
}

func componentizeRequestBodies(paths map[string]*PathItem, schemas map[string]*Schema) map[string]*RequestBodyRef {
	usages := collectRequestBodyUsages(paths)
	return componentizeReusableRefs(
		usages,
		func(usage requestBodyUsage) componentUsage[RequestBody, *RequestBodyRef] {
			return componentUsage[RequestBody, *RequestBodyRef]{
				ref:    usage.ref,
				base:   requestBodyComponentBase(usage, schemas),
				prefix: RequestBodyComponentRefPrefix,
			}
		},
		func(ref *RequestBodyRef) *RequestBody {
			if ref == nil {
				return nil
			}
			return ref.Value
		},
		func(ref *RequestBodyRef, path string) {
			ref.Ref = path
			ref.Value = nil
		},
		func(value *RequestBody) *RequestBodyRef { return &RequestBodyRef{Value: value} },
	)
}

func componentizeResponses(paths map[string]*PathItem, schemas map[string]*Schema) map[string]*ResponseRef {
	usages := collectResponseUsages(paths)
	if len(usages) == 0 {
		return nil
	}

	counts := countReusableValues(usages, func(usage responseUsage) (string, error) {
		return responseHash(usage.ref, schemas)
	})

	components := make(map[string]*ResponseRef)
	namesByHash := make(map[string]string)
	hashesByName := make(map[string]string)
	for _, usage := range usages {
		if usage.ref == nil || usage.ref.Value == nil || usage.ref.Ref != "" {
			continue
		}
		hash, err := responseHash(usage.ref, schemas)
		if err != nil || counts[hash] < 2 {
			continue
		}
		name, ok := namesByHash[hash]
		if !ok {
			base := usage.base
			if standardErrorResponseComponentBase(usage.status) == "" {
				if inferred := reusableResponseComponentBase(usage.ref, usage.status, schemas); inferred != "" {
					base = inferred
				}
			}
			name = uniqueReusableComponentName(base, hash, hashesByName)
			namesByHash[hash] = name
			hashesByName[name] = hash
			components[name] = &ResponseRef{Value: usage.ref.Value}
		}
		usage.ref.Ref = ResponseComponentRefPrefix + name
		usage.ref.Value = nil
	}
	if len(components) == 0 {
		return nil
	}
	return components
}

func collectInlineParameters(paths map[string]*PathItem) []*ParameterRef {
	refs := make([]*ParameterRef, 0)
	for _, path := range orderedPathKeys(paths) {
		pathItem := paths[path]
		if pathItem == nil {
			continue
		}
		for _, op := range orderedOperations(pathItem) {
			refs = appendInlineParameterRefs(refs, op.Parameters)
		}
	}
	return refs
}

func appendInlineParameterRefs(dst []*ParameterRef, refs []*ParameterRef) []*ParameterRef {
	for _, ref := range refs {
		if ref == nil || ref.Value == nil || ref.Ref != "" {
			continue
		}
		dst = append(dst, ref)
	}
	return dst
}

func collectExampleUsages(paths map[string]*PathItem) []exampleUsage {
	usages := make([]exampleUsage, 0)
	for _, path := range orderedPathKeys(paths) {
		pathItem := paths[path]
		if pathItem == nil {
			continue
		}
		for _, operation := range orderedOperations(pathItem) {
			usages = append(usages, operationExampleUsages(operation)...)
		}
	}
	return compactExampleUsages(usages)
}

func operationExampleUsages(operation *Operation) []exampleUsage {
	if operation == nil {
		return nil
	}
	usages := make([]exampleUsage, 0)
	opName := componentNameFromOperation(operation.OperationID)
	for _, parameter := range operation.Parameters {
		usages = append(usages, parameterExampleUsages(parameter, opName+"Parameter")...)
	}
	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		for _, contentType := range orderedStringKeys(operation.RequestBody.Value.Content) {
			mediaType := operation.RequestBody.Value.Content[contentType]
			base := opName + mediaTypeComponentSuffix(contentType) + "RequestBody"
			usages = append(usages, mediaTypeExampleUsages(mediaType, base)...)
		}
	}
	for _, status := range orderedStringKeys(operation.Responses) {
		response := operation.Responses[status]
		if response == nil || response.Value == nil {
			continue
		}
		base := opName + responseStatusComponentSuffix(status) + "Response"
		for _, headerName := range orderedStringKeys(response.Value.Headers) {
			header := response.Value.Headers[headerName]
			usages = append(usages, headerExampleUsages(header, headerComponentBase(headerName))...)
		}
		for _, contentType := range orderedStringKeys(response.Value.Content) {
			mediaType := response.Value.Content[contentType]
			usages = append(usages, mediaTypeExampleUsages(mediaType, base+mediaTypeComponentSuffix(contentType))...)
		}
	}
	return usages
}

func parameterExampleUsages(ref *ParameterRef, prefix string) []exampleUsage {
	if ref == nil || ref.Value == nil {
		return nil
	}
	base := prefix + codegen.Goify(parameterComponentSuffix(ref.Value.Name), true)
	return namedExampleUsages(ref.Value.Examples, base)
}

func mediaTypeExampleUsages(mediaType *MediaType, base string) []exampleUsage {
	if mediaType == nil {
		return nil
	}
	return namedExampleUsages(mediaType.Examples, base)
}

func headerExampleUsages(ref *HeaderRef, base string) []exampleUsage {
	if ref == nil || ref.Value == nil {
		return nil
	}
	return namedExampleUsages(ref.Value.Examples, base)
}

func namedExampleUsages(examples map[string]*ExampleRef, base string) []exampleUsage {
	if len(examples) == 0 {
		return nil
	}
	usages := make([]exampleUsage, 0, len(examples))
	for _, name := range orderedStringKeys(examples) {
		ref := examples[name]
		if ref == nil || ref.Value == nil || ref.Ref != "" {
			continue
		}
		usages = append(usages, exampleUsage{
			ref:  ref,
			base: base + codegen.Goify(name, true) + "Example",
		})
	}
	return usages
}

func compactExampleUsages(usages []exampleUsage) []exampleUsage {
	if len(usages) == 0 {
		return nil
	}
	seen := make(map[*ExampleRef]struct{}, len(usages))
	compacted := make([]exampleUsage, 0, len(usages))
	for _, usage := range usages {
		if usage.ref == nil {
			continue
		}
		if _, ok := seen[usage.ref]; ok {
			continue
		}
		seen[usage.ref] = struct{}{}
		compacted = append(compacted, usage)
	}
	return compacted
}

func collectHeaderUsages(paths map[string]*PathItem) []headerUsage {
	usages := make([]headerUsage, 0)
	for _, path := range orderedPathKeys(paths) {
		pathItem := paths[path]
		if pathItem == nil {
			continue
		}
		for _, operation := range orderedOperations(pathItem) {
			for _, status := range orderedStringKeys(operation.Responses) {
				response := operation.Responses[status]
				if response == nil || response.Value == nil {
					continue
				}
				for _, headerName := range orderedStringKeys(response.Value.Headers) {
					ref := response.Value.Headers[headerName]
					if ref == nil || ref.Value == nil || ref.Ref != "" {
						continue
					}
					usages = append(usages, headerUsage{ref: ref, base: headerComponentBase(headerName)})
				}
			}
		}
	}
	return usages
}

func collectRequestBodyUsages(paths map[string]*PathItem) []requestBodyUsage {
	usages := make([]requestBodyUsage, 0)
	for _, path := range orderedPathKeys(paths) {
		pathItem := paths[path]
		if pathItem == nil {
			continue
		}
		for _, operation := range orderedOperations(pathItem) {
			if operation.RequestBody == nil || operation.RequestBody.Value == nil || operation.RequestBody.Ref != "" {
				continue
			}
			usages = append(usages, requestBodyUsage{
				ref:  operation.RequestBody,
				base: componentNameFromOperation(operation.OperationID) + "RequestBody",
			})
		}
	}
	return usages
}

func collectResponseUsages(paths map[string]*PathItem) []responseUsage {
	usages := make([]responseUsage, 0)
	for _, path := range orderedPathKeys(paths) {
		pathItem := paths[path]
		if pathItem == nil {
			continue
		}
		for _, operation := range orderedOperations(pathItem) {
			for _, status := range orderedStringKeys(operation.Responses) {
				ref := operation.Responses[status]
				if ref == nil || ref.Value == nil || ref.Ref != "" {
					continue
				}
				usages = append(usages, responseUsage{
					ref:    ref,
					base:   responseComponentBase(operation.OperationID, status),
					status: status,
				})
			}
		}
	}
	return usages
}

func responseComponentBase(operationID, status string) string {
	if base := standardErrorResponseComponentBase(status); base != "" {
		return base
	}
	return componentNameFromOperation(operationID) + responseStatusComponentSuffix(status) + "Response"
}

func requestBodyComponentBase(usage requestBodyUsage, schemas map[string]*Schema) string {
	if usage.ref != nil && usage.ref.Value != nil && strings.TrimSpace(usage.ref.Value.ComponentName) != "" {
		return strings.TrimSpace(usage.ref.Value.ComponentName)
	}
	if base := reusableRequestBodyComponentBase(usage.ref, schemas); base != "" {
		return base
	}
	return usage.base
}

func reusableRequestBodyComponentBase(ref *RequestBodyRef, schemas map[string]*Schema) string {
	if ref == nil || ref.Value == nil || len(ref.Value.Content) != 1 {
		return ""
	}
	contentType := orderedStringKeys(ref.Value.Content)[0]
	mediaType := ref.Value.Content[contentType]
	schemaName, ok := componentSchemaNameFromMediaType(mediaType, schemas)
	if !ok {
		return ""
	}
	suffix := mediaTypeComponentSuffix(contentType)
	if strings.HasSuffix(schemaName, "RequestBody") {
		if suffix == "" {
			return schemaName
		}
		return schemaName + suffix
	}
	return schemaName + suffix + "RequestBody"
}

func reusableResponseComponentBase(ref *ResponseRef, status string, schemas map[string]*Schema) string {
	if ref == nil || ref.Value == nil {
		return ""
	}
	if base := genericEmptyResponseComponentBase(ref.Value, status); base != "" {
		return base
	}
	if len(ref.Value.Content) != 1 {
		return ""
	}
	contentType := orderedStringKeys(ref.Value.Content)[0]
	mediaType := ref.Value.Content[contentType]
	schemaName, ok := componentSchemaNameFromMediaType(mediaType, schemas)
	if !ok {
		return ""
	}
	return schemaName + mediaTypeComponentSuffix(contentType) + responseStatusComponentSuffix(status) + "Response"
}

func genericEmptyResponseComponentBase(response *Response, status string) string {
	if response == nil || len(response.Content) != 0 || len(response.Headers) != 0 {
		return ""
	}
	if !isDefaultResponseDescription(response.Description, status) {
		return ""
	}
	text := strings.TrimSpace(http.StatusText(statusCodeValue(status)))
	if text == "" {
		return ""
	}
	return codegen.Goify(text, true) + "Response"
}

func isDefaultResponseDescription(description, status string) bool {
	code := statusCodeValue(status)
	if code == 0 {
		return false
	}
	return description == fmt.Sprintf("%s response.", http.StatusText(code))
}

func statusCodeValue(status string) int {
	trimmed := strings.TrimSpace(status)
	if !isDigits(trimmed) {
		return 0
	}
	code, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0
	}
	return code
}

func componentSchemaNameFromMediaType(mediaType *MediaType, schemas map[string]*Schema) (string, bool) {
	if mediaType == nil || mediaType.Schema == nil {
		return "", false
	}
	name, ok := schemaComponentName(mediaType.Schema.Ref)
	if !ok {
		return "", false
	}
	return canonicalComponentSchemaName(name, schemas), true
}

func canonicalComponentSchemaName(name string, schemas map[string]*Schema) string {
	base, ok := duplicateAliasBase(name)
	if !ok || schemas[base] == nil || schemas[name] == nil {
		return name
	}
	cache := map[string]string{}
	if schemaHashByName(base, schemas, cache, map[string]struct{}{}) == schemaHashByName(name, schemas, cache, map[string]struct{}{}) {
		return base
	}
	return name
}

func standardErrorResponseComponentBase(status string) string {
	switch strings.TrimSpace(status) {
	case "400":
		return "BadRequestError"
	case "401":
		return "UnauthorizedError"
	case "403":
		return "ForbiddenError"
	case "404":
		return "NotFoundError"
	case "409":
		return "ConflictError"
	case "422":
		return "UnprocessableEntityError"
	case "429":
		return "TooManyRequestsError"
	default:
		return ""
	}
}

func orderedPathKeys(paths map[string]*PathItem) []string {
	keys := make([]string, 0, len(paths))
	for path := range paths {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	return keys
}

func orderedOperations(pathItem *PathItem) []*Operation {
	if pathItem == nil {
		return nil
	}
	methods := []string{"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT", "TRACE"}
	operations := make([]*Operation, 0, len(methods))
	for _, method := range methods {
		operation := pathItem.Operations[method]
		if operation != nil {
			operations = append(operations, operation)
		}
	}
	return operations
}

func orderedStringKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func countReusableValues[T any](usages []T, hashFn func(T) (string, error)) map[string]int {
	counts := make(map[string]int)
	for _, usage := range usages {
		hash, err := hashFn(usage)
		if err != nil || hash == "" {
			continue
		}
		counts[hash]++
	}
	return counts
}

func componentizeReusableRefs[U any, V any, R any](
	usages []U,
	mapUsage func(U) componentUsage[V, R],
	valueOf func(R) *V,
	setRef func(R, string),
	newRef func(*V) R,
) map[string]R {
	if len(usages) == 0 {
		return nil
	}
	mapped := make([]componentUsage[V, R], 0, len(usages))
	for _, usage := range usages {
		current := mapUsage(usage)
		if valueOf(current.ref) == nil {
			continue
		}
		mapped = append(mapped, current)
	}
	if len(mapped) == 0 {
		return nil
	}
	counts := countReusableValues(mapped, func(usage componentUsage[V, R]) (string, error) {
		return hashReusableValue(valueOf(usage.ref))
	})
	components := make(map[string]R)
	namesByHash := make(map[string]string)
	hashesByName := make(map[string]string)
	for _, usage := range mapped {
		value := valueOf(usage.ref)
		hash, err := hashReusableValue(value)
		if err != nil || counts[hash] < 2 {
			continue
		}
		name, ok := namesByHash[hash]
		if !ok {
			name = uniqueReusableComponentName(usage.base, hash, hashesByName)
			namesByHash[hash] = name
			hashesByName[name] = hash
			components[name] = newRef(value)
		}
		setRef(usage.ref, usage.prefix+name)
	}
	if len(components) == 0 {
		return nil
	}
	return components
}

func parameterHash(parameter *Parameter) (string, error) {
	return hashReusableValue(parameter)
}

func responseHash(ref *ResponseRef, schemas map[string]*Schema) (string, error) {
	if ref == nil || ref.Value == nil {
		return "", nil
	}
	normalized := cloneResponseForHash(ref.Value, schemas)
	return hashReusableValue(normalized)
}

func cloneResponseHeaderRefs(headers map[string]*HeaderRef) map[string]*HeaderRef {
	if len(headers) == 0 {
		return nil
	}
	cloned := make(map[string]*HeaderRef, len(headers))
	for name, header := range headers {
		if header == nil {
			continue
		}
		current := *header
		cloned[name] = &current
	}
	return cloned
}

func cloneResponseExampleRefs(examples map[string]*ExampleRef) map[string]*ExampleRef {
	if len(examples) == 0 {
		return nil
	}
	cloned := make(map[string]*ExampleRef, len(examples))
	for name, example := range examples {
		if example == nil {
			continue
		}
		current := *example
		cloned[name] = &current
	}
	return cloned
}

func cloneResponseExtensions(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneResponseForHash(response *Response, schemas map[string]*Schema) *Response {
	if response == nil {
		return nil
	}
	cloned := &Response{
		Description: response.Description,
		Headers:     cloneResponseHeaderRefs(response.Headers),
		Extensions:  cloneResponseExtensions(response.Extensions),
	}
	if len(response.Content) > 0 {
		cloned.Content = make(map[string]*MediaType, len(response.Content))
		for contentType, mediaType := range response.Content {
			cloned.Content[contentType] = cloneMediaTypeForHash(mediaType, schemas, map[string]string{}, map[string]struct{}{})
		}
	}
	return cloned
}

func cloneMediaTypeForHash(mediaType *MediaType, schemas map[string]*Schema, cache map[string]string, stack map[string]struct{}) *MediaType {
	if mediaType == nil {
		return nil
	}
	return &MediaType{
		Schema:     normalizeSchemaForHash(mediaType.Schema, schemas, cache, stack),
		Example:    mediaType.Example,
		Examples:   cloneResponseExampleRefs(mediaType.Examples),
		Extensions: cloneResponseExtensions(mediaType.Extensions),
	}
}

func normalizeSchemaForHash(schema *Schema, schemas map[string]*Schema, cache map[string]string, stack map[string]struct{}) *Schema {
	if schema == nil {
		return nil
	}
	if refName, ok := schemaComponentName(schema.Ref); ok {
		return &Schema{Ref: canonicalResponseSchemaRef(refName, schemas, cache)}
	}

	cloned := *schema
	cloned.Items = normalizeSchemaForHash(schema.Items, schemas, cache, stack)
	cloned.Properties = normalizeSchemaMapForHash(schema.Properties, schemas, cache, stack)
	cloned.Defs = normalizeSchemaMapForHash(schema.Defs, schemas, cache, stack)
	cloned.AnyOf = normalizeSchemaSliceForHash(schema.AnyOf, schemas, cache, stack)
	cloned.OneOf = normalizeSchemaSliceForHash(schema.OneOf, schemas, cache, stack)
	cloned.AdditionalProperties = normalizeBoolOrSchemaForHash(schema.AdditionalProperties, schemas, cache, stack)
	cloned.UnevaluatedProperties = normalizeBoolOrSchemaForHash(schema.UnevaluatedProperties, schemas, cache, stack)
	cloned.Discriminator = normalizeDiscriminatorForHash(schema.Discriminator, schemas, cache, stack)
	cloned.Links = normalizeLinksForHash(schema.Links, schemas, cache, stack)
	return &cloned
}

func normalizeSchemaMapForHash(schemasMap map[string]*Schema, schemas map[string]*Schema, cache map[string]string, stack map[string]struct{}) map[string]*Schema {
	if len(schemasMap) == 0 {
		return nil
	}
	cloned := make(map[string]*Schema, len(schemasMap))
	for name, schema := range schemasMap {
		cloned[name] = normalizeSchemaForHash(schema, schemas, cache, stack)
	}
	return cloned
}

func normalizeSchemaSliceForHash(values []*Schema, schemas map[string]*Schema, cache map[string]string, stack map[string]struct{}) []*Schema {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]*Schema, len(values))
	for i, schema := range values {
		cloned[i] = normalizeSchemaForHash(schema, schemas, cache, stack)
	}
	return cloned
}

func normalizeBoolOrSchemaForHash(value *BoolOrSchema, schemas map[string]*Schema, cache map[string]string, stack map[string]struct{}) *BoolOrSchema {
	if value == nil {
		return nil
	}
	cloned := *value
	cloned.Schema = normalizeSchemaForHash(value.Schema, schemas, cache, stack)
	return &cloned
}

func normalizeDiscriminatorForHash(discriminator *Discriminator, schemas map[string]*Schema, cache map[string]string, stack map[string]struct{}) *Discriminator {
	if discriminator == nil {
		return nil
	}
	cloned := *discriminator
	if len(discriminator.Mapping) > 0 {
		cloned.Mapping = make(map[string]string, len(discriminator.Mapping))
		for key, ref := range discriminator.Mapping {
			if refName, ok := schemaComponentName(ref); ok {
				cloned.Mapping[key] = "schema:" + schemaHashByName(refName, schemas, cache, stack)
				continue
			}
			cloned.Mapping[key] = ref
		}
	}
	return &cloned
}

func normalizeLinksForHash(links []*Link, schemas map[string]*Schema, cache map[string]string, stack map[string]struct{}) []*Link {
	if len(links) == 0 {
		return nil
	}
	cloned := make([]*Link, len(links))
	for i, link := range links {
		if link == nil {
			continue
		}
		current := *link
		current.Schema = normalizeSchemaForHash(link.Schema, schemas, cache, stack)
		current.TargetSchema = normalizeSchemaForHash(link.TargetSchema, schemas, cache, stack)
		cloned[i] = &current
	}
	return cloned
}

func schemaHashByName(name string, schemas map[string]*Schema, cache map[string]string, stack map[string]struct{}) string {
	if name == "" {
		return "missing"
	}
	if hash, ok := cache[name]; ok {
		return hash
	}
	if _, ok := stack[name]; ok {
		return "recursive"
	}
	schema := schemas[name]
	if schema == nil {
		return "missing:" + name
	}
	stack[name] = struct{}{}
	normalized := normalizeSchemaForHash(schema, schemas, cache, stack)
	delete(stack, name)
	hash, err := hashReusableValue(normalized)
	if err != nil || hash == "" {
		hash = "unhashable:" + name
	}
	cache[name] = hash
	return hash
}

func canonicalResponseSchemaRef(name string, schemas map[string]*Schema, cache map[string]string) string {
	if name == "" {
		return ""
	}
	base, ok := duplicateAliasBase(name)
	if !ok || schemas[base] == nil || schemas[name] == nil {
		return "#/components/schemas/" + name
	}
	if schemaHashByName(base, schemas, cache, map[string]struct{}{}) == schemaHashByName(name, schemas, cache, map[string]struct{}{}) {
		return "#/components/schemas/" + base
	}
	return "#/components/schemas/" + name
}

func duplicateAliasBase(name string) (string, bool) {
	idx := strings.LastIndex(name, "_")
	if idx <= 0 || idx == len(name)-1 {
		return "", false
	}
	suffix := name[idx+1:]
	if !isDigits(suffix) {
		return "", false
	}
	return name[:idx], true
}

func schemaComponentName(ref string) (string, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	return strings.TrimPrefix(ref, prefix), true
}

func hashReusableValue(value any) (string, error) {
	if value == nil {
		return "", nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:]), nil
}

func uniqueParameterComponentName(parameter *Parameter, hash string, hashesByName map[string]string) string {
	base := ParameterComponentName(parameter)
	if existingHash, ok := hashesByName[base]; !ok || existingHash == hash {
		return base
	}
	return fmt.Sprintf("%s_%s", base, hash[:8])
}

func ParameterComponentName(parameter *Parameter) string {
	if parameter == nil {
		return "Parameter"
	}
	if strings.TrimSpace(parameter.ComponentName) != "" {
		return strings.TrimSpace(parameter.ComponentName)
	}
	base := codegen.Goify(parameter.In, true) + codegen.Goify(parameterComponentSuffix(parameter.Name), true)
	if base == "" {
		return "Parameter"
	}
	return base
}

func uniqueReusableComponentName(base, hash string, hashesByName map[string]string) string {
	if existingHash, ok := hashesByName[base]; !ok || existingHash == hash {
		return base
	}
	return fmt.Sprintf("%s_%s", base, hash[:8])
}

func componentNameFromOperation(operationID string) string {
	base := codegen.Goify(canonicalOperationIDComponent(operationID), true)
	if base == "" {
		return "Operation"
	}
	return base
}

func headerComponentBase(name string) string {
	base := codegen.Goify(strings.TrimSpace(name), true)
	if base == "" {
		base = "Header"
	}
	if !strings.HasSuffix(base, "Header") {
		base += "Header"
	}
	return base
}

func mediaTypeComponentSuffix(contentType string) string {
	trimmed := strings.TrimSpace(contentType)
	if trimmed == "" || trimmed == "application/json" {
		return ""
	}
	return codegen.Goify(trimmed, true)
}

func responseStatusComponentSuffix(status string) string {
	trimmed := strings.TrimSpace(status)
	switch {
	case trimmed == "", trimmed == "default":
		return "Default"
	case isDigits(trimmed):
		return "Status" + trimmed
	default:
		return codegen.Goify(trimmed, true)
	}
}

func parameterComponentSuffix(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Parameter"
	}
	return trimmed
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}
