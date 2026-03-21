package openapiv3

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/openapi"
)

const (
	exampleComponentRefPrefix     = "#/components/examples/"
	headerComponentRefPrefix      = "#/components/headers/"
	requestBodyComponentRefPrefix = "#/components/requestBodies/"
	responseComponentRefPrefix    = "#/components/responses/"
)

type (
	reusableComponents struct {
		Parameters    map[string]*ParameterRef
		Headers       map[string]*HeaderRef
		RequestBodies map[string]*RequestBodyRef
		Responses     map[string]*ResponseRef
		Examples      map[string]*ExampleRef
	}

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

func componentizeReusableComponents(paths map[string]*PathItem) reusableComponents {
	examples := componentizeExamples(paths)
	headers := componentizeHeaders(paths)
	requestBodies := componentizeRequestBodies(paths)
	responses := componentizeResponses(paths)
	parameters := componentizeParameters(paths)

	return reusableComponents{
		Parameters:    parameters,
		Headers:       headers,
		RequestBodies: requestBodies,
		Responses:     responses,
		Examples:      examples,
	}
}

func componentizeExamples(paths map[string]*PathItem) map[string]*ExampleRef {
	usages := collectExampleUsages(paths)
	return componentizeReusableRefs(
		usages,
		func(usage exampleUsage) componentUsage[Example, *ExampleRef] {
			return componentUsage[Example, *ExampleRef]{
				ref:    usage.ref,
				base:   usage.base,
				prefix: exampleComponentRefPrefix,
			}
		},
		func(ref *ExampleRef) *Example {
			if ref == nil {
				return nil
			}
			return ref.Value
		},
		func(ref *ExampleRef, refPath string) {
			ref.Ref = refPath
			ref.Value = nil
		},
		func(value *Example) *ExampleRef {
			return &ExampleRef{Value: value}
		},
	)
}

func componentizeHeaders(paths map[string]*PathItem) map[string]*HeaderRef {
	usages := collectHeaderUsages(paths)
	return componentizeReusableRefs(
		usages,
		func(usage headerUsage) componentUsage[Header, *HeaderRef] {
			return componentUsage[Header, *HeaderRef]{
				ref:    usage.ref,
				base:   usage.base,
				prefix: headerComponentRefPrefix,
			}
		},
		func(ref *HeaderRef) *Header {
			if ref == nil {
				return nil
			}
			return ref.Value
		},
		func(ref *HeaderRef, refPath string) {
			ref.Ref = refPath
			ref.Value = nil
		},
		func(value *Header) *HeaderRef {
			return &HeaderRef{Value: value}
		},
	)
}

func componentizeRequestBodies(paths map[string]*PathItem) map[string]*RequestBodyRef {
	usages := collectRequestBodyUsages(paths)
	return componentizeReusableRefs(
		usages,
		func(usage requestBodyUsage) componentUsage[RequestBody, *RequestBodyRef] {
			return componentUsage[RequestBody, *RequestBodyRef]{
				ref:    usage.ref,
				base:   usage.base,
				prefix: requestBodyComponentRefPrefix,
			}
		},
		func(ref *RequestBodyRef) *RequestBody {
			if ref == nil {
				return nil
			}
			return ref.Value
		},
		func(ref *RequestBodyRef, refPath string) {
			ref.Ref = refPath
			ref.Value = nil
		},
		func(value *RequestBody) *RequestBodyRef {
			return &RequestBodyRef{Value: value}
		},
	)
}

func componentizeResponses(paths map[string]*PathItem) map[string]*ResponseRef {
	usages := collectResponseUsages(paths)
	return componentizeReusableRefs(
		usages,
		func(usage responseUsage) componentUsage[Response, *ResponseRef] {
			return componentUsage[Response, *ResponseRef]{
				ref:    usage.ref,
				base:   usage.base,
				prefix: responseComponentRefPrefix,
			}
		},
		func(ref *ResponseRef) *Response {
			if ref == nil {
				return nil
			}
			return ref.Value
		},
		func(ref *ResponseRef, refPath string) {
			ref.Ref = refPath
			ref.Value = nil
		},
		func(value *Response) *ResponseRef {
			return &ResponseRef{Value: value}
		},
	)
}

func collectReusableComponentSchemaRefs(reusable reusableComponents, addRef func(string)) {
	for _, parameter := range reusable.Parameters {
		if parameter == nil || parameter.Value == nil {
			continue
		}
		collectSchemaRefs(parameter.Value.Schema, addRef)
	}
	for _, header := range reusable.Headers {
		if header == nil || header.Value == nil {
			continue
		}
		collectSchemaRefs(header.Value.Schema, addRef)
		collectMediaTypeSchemaRefs(header.Value.Content, addRef)
	}
	for _, requestBody := range reusable.RequestBodies {
		if requestBody == nil || requestBody.Value == nil {
			continue
		}
		collectMediaTypeSchemaRefs(requestBody.Value.Content, addRef)
	}
	for _, response := range reusable.Responses {
		if response == nil || response.Value == nil {
			continue
		}
		for _, header := range response.Value.Headers {
			if header == nil || header.Value == nil {
				continue
			}
			collectSchemaRefs(header.Value.Schema, addRef)
			collectMediaTypeSchemaRefs(header.Value.Content, addRef)
		}
		collectMediaTypeSchemaRefs(response.Value.Content, addRef)
	}
}

func collectMediaTypeSchemaRefs(mediaTypes map[string]*MediaType, addRef func(string)) {
	for _, mediaType := range mediaTypes {
		if mediaType == nil {
			continue
		}
		collectSchemaRefs(mediaType.Schema, addRef)
	}
}

func operationTagNames(endpointMeta, serviceMeta expr.MetaExpr, serviceName string) []string {
	tagNames := openapi.TagNamesFromExpr(endpointMeta)
	if len(tagNames) > 0 {
		return tagNames
	}
	tagNames = openapi.TagNamesFromExpr(serviceMeta)
	if len(tagNames) > 0 {
		return tagNames
	}
	return []string{serviceName}
}

func orderedPathKeys(paths map[string]*PathItem) []string {
	keys := make([]string, 0, len(paths))
	for path := range paths {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	return keys
}

func appendOperationRefs(paths map[string]*PathItem, fn func(path string, pathItem *PathItem, method string, operation *Operation)) {
	for _, path := range orderedPathKeys(paths) {
		pathItem := paths[path]
		if pathItem == nil {
			continue
		}
		for _, op := range orderedOperations(pathItem) {
			fn(path, pathItem, op.method, op.operation)
		}
	}
}

type orderedOperation struct {
	method    string
	operation *Operation
}

func orderedOperations(pathItem *PathItem) []orderedOperation {
	ops := []orderedOperation{
		{method: "connect", operation: pathItem.Connect},
		{method: "delete", operation: pathItem.Delete},
		{method: "get", operation: pathItem.Get},
		{method: "head", operation: pathItem.Head},
		{method: "options", operation: pathItem.Options},
		{method: "patch", operation: pathItem.Patch},
		{method: "post", operation: pathItem.Post},
		{method: "put", operation: pathItem.Put},
		{method: "trace", operation: pathItem.Trace},
	}
	return slices.DeleteFunc(ops, func(op orderedOperation) bool {
		return op.operation == nil
	})
}

func collectExampleUsages(paths map[string]*PathItem) []exampleUsage {
	usages := make([]exampleUsage, 0)
	appendOperationRefs(paths, func(_ string, pathItem *PathItem, _ string, operation *Operation) {
		usages = append(usages, pathItemExampleUsages(pathItem)...)
		usages = append(usages, operationExampleUsages(operation)...)
	})
	return compactExampleUsages(usages)
}

func pathItemExampleUsages(pathItem *PathItem) []exampleUsage {
	if pathItem == nil {
		return nil
	}
	usages := make([]exampleUsage, 0)
	for _, parameter := range pathItem.Parameters {
		usages = append(usages, parameterExampleUsages(parameter, "Parameter")...)
	}
	return usages
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
	appendOperationRefs(paths, func(_ string, _ *PathItem, _ string, operation *Operation) {
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
				usages = append(usages, headerUsage{
					ref:  ref,
					base: headerComponentBase(headerName),
				})
			}
		}
	})
	return usages
}

func collectRequestBodyUsages(paths map[string]*PathItem) []requestBodyUsage {
	usages := make([]requestBodyUsage, 0)
	appendOperationRefs(paths, func(_ string, _ *PathItem, _ string, operation *Operation) {
		if operation.RequestBody == nil || operation.RequestBody.Value == nil || operation.RequestBody.Ref != "" {
			return
		}
		usages = append(usages, requestBodyUsage{
			ref:  operation.RequestBody,
			base: componentNameFromOperation(operation.OperationID) + "RequestBody",
		})
	})
	return usages
}

func collectResponseUsages(paths map[string]*PathItem) []responseUsage {
	usages := make([]responseUsage, 0)
	appendOperationRefs(paths, func(_ string, _ *PathItem, _ string, operation *Operation) {
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
	})
	return usages
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
	if trimmed == "" {
		return "Default"
	}
	if trimmed == "default" {
		return "Default"
	}
	if isDigits(trimmed) {
		return "Status" + trimmed
	}
	return codegen.Goify(trimmed, true)
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}
