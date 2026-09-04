package openapi

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// MarshalJSON produces the JSON resulting from encoding an object composed of
// the fields in v (which must me a struct) and the keys in extensions.
func MarshalJSON(v any, extensions map[string]any) ([]byte, error) {
	marshaled, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	if len(extensions) == 0 {
		return marshaled, nil
	}
	fields := make(map[string]jsontext.Value)
	if err := json.Unmarshal(marshaled, &fields); err != nil {
		return nil, err
	}
	for name, extension := range extensions {
		value, marshalErr := json.Marshal(extension, json.Deterministic(true))
		if marshalErr != nil {
			return nil, marshalErr
		}
		fields[name] = value
	}
	merged, err := json.Marshal(fields, json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	return merged, nil
}

// MarshalYAML produces the YAML resulting from encoding an object composed of
// the fields in v (which must be a struct) and the keys in extensions.
func MarshalYAML(v any, extensions map[string]any) (any, error) {
	if len(extensions) == 0 {
		return v, nil
	}
	marshaled, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(marshaled, &document); err != nil {
		return nil, err
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("openapi: YAML extension target must encode as an object")
	}
	mapping := document.Content[0]
	fields := make(map[string]*yaml.Node, len(mapping.Content)/2+len(extensions))
	for index := 0; index < len(mapping.Content); index += 2 {
		fields[mapping.Content[index].Value] = mapping.Content[index+1]
	}
	for name, extension := range extensions {
		value, encodeErr := encodeYAMLNode(extension)
		if encodeErr != nil {
			return nil, encodeErr
		}
		fields[name] = value
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	mapping.Content = mapping.Content[:0]
	for _, name := range names {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name},
			fields[name],
		)
	}
	return mapping, nil
}

func encodeYAMLNode(value any) (*yaml.Node, error) {
	if raw, ok := value.(jsontext.Value); ok {
		var document yaml.Node
		if err := yaml.Unmarshal(raw, &document); err != nil {
			return nil, err
		}
		if len(document.Content) != 1 {
			return nil, fmt.Errorf("openapi: JSON extension must contain one value")
		}
		normalizeJSONYAMLNode(document.Content[0])
		return document.Content[0], nil
	}
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return nil, err
	}
	return &node, nil
}

func normalizeJSONYAMLNode(node *yaml.Node) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode || node.Tag == "!!str" {
		node.Style = 0
	}
	for _, child := range node.Content {
		normalizeJSONYAMLNode(child)
	}
	if node.Kind == yaml.MappingNode {
		type pair struct{ key, value *yaml.Node }
		pairs := make([]pair, 0, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			pairs = append(pairs, pair{key: node.Content[index], value: node.Content[index+1]})
		}
		sort.SliceStable(pairs, func(left, right int) bool {
			return pairs[left].key.Value < pairs[right].key.Value
		})
		node.Content = node.Content[:0]
		for _, entry := range pairs {
			node.Content = append(node.Content, entry.key, entry.value)
		}
	}
}
