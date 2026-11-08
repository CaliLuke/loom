package openapiimport

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type sseAsyncSchema struct {
	value *Schema
	path  string
}

type sseSchemaPair struct{ left, right *Schema }

func (p *documentPlanner) sseAsync(operation *Operation, responses operationResponsePlan, path string) {
	value, present := operation.Extensions["x-loom-async"]
	if !present {
		return
	}
	at := path + "/x-loom-async"
	if responses.mode != responseBodySSE {
		p.unsupported("sse-async", at, "reserved derived extension requires a supported itemSchema SSE contract")
		return
	}
	if err := validateSSEAsyncValue(value, operation, responses); err != nil {
		p.unsupported("sse-async", at, err.Error())
	}
	// Reserved roots are comparison-only. The ordinary event owns native
	// representation admission; a matching expansion creates no declaration.
	var outbound *Schema
	for _, schema := range operation.asyncSchemas {
		if strings.HasSuffix(schema.path, "/outbound/schema") {
			outbound = schema.value
		}
	}
	event := responses.successes[0].source
	if outbound == nil || !p.sameSSEGraph(event.Response.Schema, outbound, make(map[sseSchemaPair]bool)) {
		eventPath := responseSchemaPath(path+"/responses/"+escapeJSONPointer(event.Status), event.Response)
		p.unsupported("sse-async", at+"/messages/outbound/schema", fmt.Sprintf("derived message does not equal the complete event contract at %s, including occurrence annotations and examples", eventPath))
	}
}

func validateSSEAsyncValue(value any, operation *Operation, responses operationResponsePlan) error {
	contract, ok := exactSSEObject(value, "transport", "direction", "handshake", "messages")
	if !ok || contract["transport"] != "sse" || contract["direction"] != "server" {
		return fmt.Errorf("reserved contract requires exactly transport sse, direction server, handshake and messages")
	}
	handshake, ok := exactSSEObject(contract["handshake"], "path", "request", "response")
	if !ok || handshake["path"] != operation.Path {
		return fmt.Errorf("handshake path must equal the operation route")
	}
	request, ok := exactSSEObject(handshake["request"], "method")
	if !ok || request["method"] != operation.Method {
		return fmt.Errorf("handshake method must equal the operation method")
	}
	if fields, ok := handshake["response"].(map[string]any); ok {
		if _, legacy := fields["status"]; legacy {
			return fmt.Errorf("scalar status is obsolete and requires regeneration; use canonical statuses")
		}
	}
	response, ok := exactSSEObject(handshake["response"], "statuses", "contentType")
	if !ok || response["contentType"] != "text/event-stream" {
		return fmt.Errorf("handshake response requires exactly statuses and text/event-stream")
	}
	keys := make([]string, 0, len(responses.successes))
	for _, response := range responses.successes {
		keys = append(keys, response.selector.OpenAPIKey())
	}
	slices.Sort(keys)
	statuses, ok := response["statuses"].([]any)
	if !ok || len(statuses) != len(keys) || len(keys) == 0 {
		return fmt.Errorf("statuses must contain every successful canonical key once in lexical order")
	}
	for i, key := range keys {
		if statuses[i] != key {
			return fmt.Errorf("statuses must contain every successful canonical key once in lexical order")
		}
	}
	messages, ok := exactSSEObject(contract["messages"], "outbound")
	if !ok {
		return fmt.Errorf("only the outbound server message is supported")
	}
	outbound, ok := exactSSEObject(messages["outbound"], "contentType", "schema", "sse")
	if !ok || outbound["contentType"] != "application/json" {
		return fmt.Errorf("outbound message requires application/json, schema and null SSE mappings")
	}
	mappings, ok := exactSSEObject(outbound["sse"], "requestIDField", "dataField", "idField", "eventField", "retryField")
	if !ok {
		return fmt.Errorf("exactly the five null SSE mappings are supported; projections are unsupported")
	}
	for _, value := range mappings {
		if value != nil {
			return fmt.Errorf("all SSE mappings must be null")
		}
	}
	return nil
}

func exactSSEObject(value any, keys ...string) (map[string]any, bool) {
	object, ok := value.(map[string]any)
	if !ok || len(object) != len(keys) {
		return nil, false
	}
	for _, key := range keys {
		if _, present := object[key]; !present {
			return nil, false
		}
	}
	return object, true
}

// sameSSEGraph compares normalized typed edges with a pair guard. Bare
// declaration references may pair with inline expansions, but distinct references
// never compare as one declaration. Occurrence overlays stay intact and
// must compare as retained contracts, never by matching example values alone.
func (p *documentPlanner) sameSSEGraph(a, b *Schema, seen map[sseSchemaPair]bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Ref != "" && b.Ref != "" {
		return a.Ref == b.Ref && sameSSEOccurrence(a, b)
	}
	originalA, originalB := a, b
	var ok bool
	if a, ok = p.sseReferenceContract(a, make(map[*Schema]bool)); !ok {
		return false
	}
	if b, ok = p.sseReferenceContract(b, make(map[*Schema]bool)); !ok {
		return false
	}
	pair := sseSchemaPair{originalA, originalB}
	if seen[pair] {
		return true
	}
	seen[pair] = true
	left, right := sseComparableNode(a), sseComparableNode(b)
	left.Items, right.Items = nil, nil
	left.Properties, right.Properties = nil, nil
	left.OneOf, right.OneOf = nil, nil
	left.Bases, right.Bases = nil, nil
	left.AdditionalProperties, right.AdditionalProperties = nil, nil
	if !reflect.DeepEqual(left, right) || len(a.Properties) != len(b.Properties) || len(a.OneOf) != len(b.OneOf) || len(a.Bases) != len(b.Bases) {
		return false
	}
	if !p.sameSSEGraph(a.Items, b.Items, seen) {
		return false
	}
	for i, property := range a.Properties {
		if property.Name != b.Properties[i].Name || !p.sameSSEGraph(property.Schema, b.Properties[i].Schema, seen) {
			return false
		}
	}
	for i, branch := range a.OneOf {
		if !p.sameSSEGraph(branch, b.OneOf[i], seen) {
			return false
		}
	}
	for i, base := range a.Bases {
		if !p.sameSSEGraph(base, b.Bases[i], seen) {
			return false
		}
	}
	if a.AdditionalProperties == nil || b.AdditionalProperties == nil {
		return a.AdditionalProperties == b.AdditionalProperties
	}
	return reflect.DeepEqual(a.AdditionalProperties.Allowed, b.AdditionalProperties.Allowed) && p.sameSSEGraph(a.AdditionalProperties.Schema, b.AdditionalProperties.Schema, seen)
}
