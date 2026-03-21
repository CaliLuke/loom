package ir

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
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
		ref  *ResponseRef
		base string
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
	doc.Components.RequestBodies = componentizeRequestBodies(doc.Paths)
	doc.Components.Responses = componentizeResponses(doc.Paths)
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

func componentizeRequestBodies(paths map[string]*PathItem) map[string]*RequestBodyRef {
	usages := collectRequestBodyUsages(paths)
	return componentizeReusableRefs(
		usages,
		func(usage requestBodyUsage) componentUsage[RequestBody, *RequestBodyRef] {
			return componentUsage[RequestBody, *RequestBodyRef]{ref: usage.ref, base: usage.base, prefix: RequestBodyComponentRefPrefix}
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

func componentizeResponses(paths map[string]*PathItem) map[string]*ResponseRef {
	usages := collectResponseUsages(paths)
	return componentizeReusableRefs(
		usages,
		func(usage responseUsage) componentUsage[Response, *ResponseRef] {
			return componentUsage[Response, *ResponseRef]{ref: usage.ref, base: usage.base, prefix: ResponseComponentRefPrefix}
		},
		func(ref *ResponseRef) *Response {
			if ref == nil {
				return nil
			}
			return ref.Value
		},
		func(ref *ResponseRef, path string) {
			ref.Ref = path
			ref.Value = nil
		},
		func(value *Response) *ResponseRef { return &ResponseRef{Value: value} },
	)
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
					ref:  ref,
					base: componentNameFromOperation(operation.OperationID) + responseStatusComponentSuffix(status) + "Response",
				})
			}
		}
	}
	return usages
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
