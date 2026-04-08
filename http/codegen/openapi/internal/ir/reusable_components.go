package ir

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
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
		if err != nil || (!shouldForceComponentizeParameter(ref.Value) && counts[hash] < 2) {
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
		func(usage componentUsage[Example, *ExampleRef]) bool {
			example := valueOfExampleRef(usage.ref)
			return example != nil && strings.TrimSpace(example.ComponentName) != ""
		},
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
		func(componentUsage[Header, *HeaderRef]) bool { return false },
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
		func(usage componentUsage[RequestBody, *RequestBodyRef]) bool {
			return shouldForceComponentizeRequestBody(requestBodyValue(usage.ref))
		},
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
		if err != nil || (!shouldForceComponentizeResponse(usage.ref.Value) && counts[hash] < 2) {
			continue
		}
		name, ok := namesByHash[hash]
		if !ok {
			base := usage.base
			if inferred := reusableResponseComponentBase(usage.ref, usage.status, schemas); inferred != "" {
				base = inferred
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
	if strings.TrimSpace(ref.Value.ComponentName) != "" {
		return strings.TrimSpace(ref.Value.ComponentName)
	}
	if base := reusableErrorResponseComponentBase(ref.Value, status, schemas); base != "" {
		return base
	}
	if base := standardErrorResponseComponentBase(status); base != "" {
		return base
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

func reusableErrorResponseComponentBase(response *Response, status string, schemas map[string]*Schema) string {
	if response == nil || !isErrorSchemaResponse(response, schemas) {
		return ""
	}
	if code := responseErrorCode(response.Description); code != "" {
		if generic := genericErrorCodeForStatus(status); generic != "" && code == generic {
			return standardErrorResponseComponentBase(status)
		}
		return semanticErrorComponentBase(code)
	}
	return ""
}

func isErrorSchemaResponse(response *Response, schemas map[string]*Schema) bool {
	if response == nil || len(response.Content) != 1 {
		return false
	}
	contentType := orderedStringKeys(response.Content)[0]
	mediaType := response.Content[contentType]
	schemaName, ok := componentSchemaNameFromMediaType(mediaType, schemas)
	return ok && schemaName == "Error"
}

func responseErrorCode(description string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return ""
	}
	code, _, ok := strings.Cut(desc, ":")
	if !ok {
		return ""
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	for _, r := range code {
		if !(r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return ""
		}
	}
	return code
}

func semanticErrorComponentBase(code string) string {
	base := codegen.Goify(code, true)
	if base == "" {
		return "Error"
	}
	if strings.HasSuffix(base, "Error") {
		return base
	}
	return base + "Error"
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

func genericErrorCodeForStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "400":
		return "bad_request"
	case "401":
		return "unauthorized"
	case "403":
		return "forbidden"
	case "404":
		return "not_found"
	case "409":
		return "conflict"
	case "422":
		return "unprocessable_entity"
	case "429":
		return "rate_limited"
	default:
		return ""
	}
}
