package openapiimport

import (
	"fmt"

	"github.com/pb33f/go-yaml"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// sseEvent retains the decoded payload occurrence separately from its fixed wire
// envelope. The source declaration and path survive occurrence-local comparison.
type sseEvent struct {
	path        string
	declaration string
}

func (a *analyzer) sseResponse(content *orderedmap.Map[string, *v3.MediaType], path string) (*Schema, *sseEvent) {
	present := false
	for contentType, media := range content.FromOldest() {
		present = present || normalizedMediaType(contentType) == "text/event-stream" && media != nil && media.ItemSchema != nil
	}
	if !present {
		return nil, nil
	}
	event := &sseEvent{path: path}
	if !a.openAPI32() {
		a.unsupported("versioned-field", path, "itemSchema requires OpenAPI 3.2")
		return nil, event
	}
	if orderedmap.Len(content) != 1 {
		a.unsupported("sse-envelope", path, "SSE itemSchema requires one text/event-stream representation")
		return nil, event
	}
	var payload *Schema
	for contentType, media := range content.FromOldest() {
		mediaPath := path + "/" + escapeJSONPointer(contentType)
		event.path = mediaPath + "/itemSchema/properties/data/contentSchema"
		if normalizedMediaType(contentType) != "text/event-stream" || media == nil || media.ItemSchema == nil {
			a.unsupported("sse-envelope", mediaPath, "itemSchema is supported only for text/event-stream")
			return nil, event
		}
		if media.Schema != nil || media.Example != nil || orderedmap.Len(media.Examples) > 0 || orderedmap.Len(media.Encoding) > 0 || orderedmap.Len(media.ItemEncoding) > 0 || orderedmap.Len(media.Extensions) > 0 {
			a.unsupported("sse-envelope", mediaPath, "SSE itemSchema cannot coexist with whole-body schemas, examples, encodings or media extensions")
			return nil, event
		}
		a.mediaTypeParserGapDiagnostics(media, mediaPath)
		if err := validateSSEEnvelope(semanticSchemaNode(media.ItemSchema)); err != nil {
			a.unsupported("sse-envelope", mediaPath+"/itemSchema", err.Error())
			return nil, event
		}
		envelope := media.ItemSchema.Schema()
		if envelope == nil {
			a.unsupported("sse-envelope", mediaPath+"/itemSchema", "SSE envelope could not be decoded")
			return nil, event
		}
		data := envelope.Properties.GetOrZero("data")
		if data == nil || data.Schema() == nil || data.Schema().ContentSchema == nil {
			a.unsupported("sse-envelope", mediaPath+"/itemSchema", "SSE data contentSchema could not be decoded")
			return nil, event
		}
		payload = a.schema(data.Schema().ContentSchema, event.path)
		if payload != nil {
			event.declaration = data.Schema().ContentSchema.GetReference()
		}
	}
	return payload, event
}

func validateSSEEnvelope(node *yaml.Node) error {
	fields, err := exactSSEFields(node, "type", "additionalProperties", "required", "properties")
	if err != nil {
		return fmt.Errorf("SSE envelope: %w", err)
	}
	if !sseString(fields["type"], "object") || fields["additionalProperties"].Tag != "!!bool" || fields["additionalProperties"].Value != "false" {
		return fmt.Errorf("SSE envelope must be an inline explicitly closed object")
	}
	required := fields["required"]
	if required.Kind != yaml.SequenceNode || len(required.Content) != 1 || !sseString(required.Content[0], "data") {
		return fmt.Errorf("SSE envelope must require only data")
	}
	properties, err := exactSSEFields(fields["properties"], "data")
	if err != nil {
		return fmt.Errorf("SSE envelope properties: %w", err)
	}
	data, err := exactSSEFields(properties["data"], "type", "contentMediaType", "contentSchema")
	if err != nil {
		return fmt.Errorf("SSE data: %w", err)
	}
	if !sseString(data["type"], "string") || !sseString(data["contentMediaType"], "application/json") {
		return fmt.Errorf("SSE data must be a string containing application/json")
	}
	return nil
}

func exactSSEFields(node *yaml.Node, names ...string) (map[string]*yaml.Node, error) {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected an inline object")
	}
	fields, err := effectiveYAMLMapping(node, make(map[*yaml.Node]bool))
	if err != nil {
		return nil, err
	}
	if len(fields) != len(names) {
		return nil, fmt.Errorf("unsupported or missing contract fields")
	}
	for _, name := range names {
		if fields[name] == nil {
			return nil, fmt.Errorf("missing contract field %s", name)
		}
	}
	return fields, nil
}

func sseString(node *yaml.Node, value string) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!str" && node.Value == value
}
