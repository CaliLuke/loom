package ir

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

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
	shouldForce func(componentUsage[V, R]) bool,
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
		if err != nil || (!shouldForce(usage) && counts[hash] < 2) {
			continue
		}
		name, ok := namesByHash[hash]
		if !ok {
			if explicit := explicitReusableComponentName(any(value)); explicit != "" {
				name = uniqueReusableComponentName(explicit, hash, hashesByName)
			} else {
				name = uniqueReusableComponentName(usage.base, hash, hashesByName)
			}
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

func valueOfExampleRef(ref *ExampleRef) *Example {
	if ref == nil {
		return nil
	}
	return ref.Value
}

func explicitReusableComponentName(value any) string {
	switch v := value.(type) {
	case *Example:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(v.ComponentName)
	default:
		return ""
	}
}

func parameterHash(parameter *Parameter) (string, error) {
	return hashReusableValue(parameter)
}

func requestBodyValue(ref *RequestBodyRef) *RequestBody {
	if ref == nil {
		return nil
	}
	return ref.Value
}

func shouldForceComponentizeParameter(parameter *Parameter) bool {
	return parameter != nil && strings.TrimSpace(parameter.ComponentName) != ""
}

func shouldForceComponentizeRequestBody(body *RequestBody) bool {
	return body != nil && strings.TrimSpace(body.ComponentName) != ""
}

func shouldForceComponentizeResponse(response *Response) bool {
	return response != nil && strings.TrimSpace(response.ComponentName) != ""
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
		Links:       cloneResponseLinkRefs(response.Links),
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

func cloneResponseLinkRefs(links map[string]*ResponseLinkRef) map[string]*ResponseLinkRef {
	if len(links) == 0 {
		return nil
	}
	cloned := make(map[string]*ResponseLinkRef, len(links))
	for name, link := range links {
		if link == nil {
			continue
		}
		current := *link
		if link.Value != nil {
			value := *link.Value
			if len(link.Value.Parameters) > 0 {
				value.Parameters = make(map[string]any, len(link.Value.Parameters))
				for key, parameter := range link.Value.Parameters {
					value.Parameters[key] = parameter
				}
			}
			value.Extensions = cloneResponseExtensions(link.Value.Extensions)
			current.Value = &value
		}
		cloned[name] = &current
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
