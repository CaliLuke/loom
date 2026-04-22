package ir

import (
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
