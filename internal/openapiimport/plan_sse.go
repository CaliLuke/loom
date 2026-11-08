package openapiimport

import (
	"fmt"
	"reflect"
	"slices"
)

// validateSSE uses the response classification owner before naming. A stream
// has one event contract; status and header occurrences remain independent.
func (p operationResponsePlan) validateSSE(path string, components Components) Diagnostics {
	var diagnostics Diagnostics
	comparison := documentPlanner{schemas: make(map[string]NamedSchema, len(components.Schemas))}
	for _, named := range components.Schemas {
		comparison.schemas[named.Name] = named
	}
	var first *Response
	firstPath := ""
	for _, group := range [][]plannedResponse{p.failures, p.informational} {
		for _, response := range group {
			if response.source.Response.stream != nil {
				diagnostics.add("sse-response", path+"/responses/"+response.source.Status, "SSE itemSchema requires a successful response")
			}
		}
	}
	for _, response := range p.successes {
		if response.source.Response.stream != nil {
			first = &response.source.Response
			firstPath = path + "/responses/" + response.source.Status
			break
		}
	}
	if first == nil {
		return diagnostics
	}
	declaration := ""
	for _, response := range p.successes {
		value := response.source.Response
		at := path + "/responses/" + response.source.Status
		if value.stream == nil || value.Schema == nil {
			diagnostics.add("sse-response", at, "every successful response must describe the common SSE event")
			continue
		}
		if response.selector.Code == 204 || response.selector.Code == 304 {
			diagnostics.add("sse-response", at, "SSE requires a body-bearing status")
		}
		conflictingDeclaration := declaration != "" && value.stream.declaration != "" && declaration != value.stream.declaration
		if declaration == "" {
			declaration = value.stream.declaration
		}
		if len(p.successes) > 1 && (conflictingDeclaration || !comparison.sameSSEGraph(first.Schema, value.Schema, make(map[sseSchemaPair]bool))) {
			diagnostics.add("sse-event", at, fmt.Sprintf("event occurrence differs from %s; multiple successes require equal complete event contracts and consistent declared identity", firstPath))
		}
	}
	return diagnostics
}

// sameSSEOccurrence compares normalized local contracts without following a
// declaration twice. Ref identity is compared, never inferred from samples or
// generated names. Diagnostic acquisition provenance is excluded; all retained
// schema facts (including ordered examples, exact numbers and presence) participate.
func sameSSEOccurrence(a, b *Schema) bool {
	return reflect.DeepEqual(sseComparableSchema(a, make(map[*Schema]*Schema)), sseComparableSchema(b, make(map[*Schema]*Schema)))
}

func sseComparableSchema(source *Schema, seen map[*Schema]*Schema) *Schema {
	if source == nil {
		return nil
	}
	if previous := seen[source]; previous != nil {
		return previous
	}
	result := sseComparableNode(source)
	seen[source] = &result
	result.Properties = append([]NamedProperty(nil), source.Properties...)
	for i := range result.Properties {
		result.Properties[i].Schema = sseComparableSchema(source.Properties[i].Schema, seen)
	}
	result.Items = sseComparableSchema(source.Items, seen)
	result.OneOf = make([]*Schema, len(source.OneOf))
	for i, branch := range source.OneOf {
		result.OneOf[i] = sseComparableSchema(branch, seen)
	}
	result.Bases = make([]*Schema, len(source.Bases))
	for i, base := range source.Bases {
		result.Bases[i] = sseComparableSchema(base, seen)
	}
	if source.AdditionalProperties != nil {
		properties := *source.AdditionalProperties
		properties.Schema = sseComparableSchema(properties.Schema, seen)
		result.AdditionalProperties = &properties
	}
	return &result
}

// sseComparableNode detaches comparison-only fields from diagnostic provenance.
// Example value state and authored annotations remain contract facts; membership
// owner pointers and source locations remain on the original graph only.
func sseComparableNode(source *Schema) Schema {
	result := *source
	result.sourcePath = ""
	result.Examples = slices.Clone(source.Examples)
	for index := range result.Examples {
		result.Examples[index].source = exampleSource{}
	}
	return result
}
