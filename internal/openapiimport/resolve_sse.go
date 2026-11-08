package openapiimport

import (
	"maps"
	"slices"

	"github.com/pb33f/go-yaml"
)

type sseSchemaRoot struct {
	node *yaml.Node
	path string
}

// scanSSEOperations extends the existing reference fixed point. A reserved
// extension gains typed edges only after an actual response chain reaches an
// item-schema stream. No other vendor extension is traversed.
func (r *descriptionResolver) scanSSEOperations() error {
	for _, key := range slices.Sorted(maps.Keys(r.result.registry.docs)) {
		doc := r.result.registry.docs[key]
		for _, path := range slices.Sorted(maps.Keys(doc.operations)) {
			if doc.operations[path] {
				continue
			}
			node, err := descriptionNode(doc.root, path)
			if err != nil {
				return err
			}
			fields, err := sseNodeFields(node)
			if err != nil {
				return err
			}
			responses, err := sseNodeFields(fields["responses"])
			if err != nil {
				return err
			}
			guard := false
			for status, response := range responses {
				resolved, err := r.sseResponseNode(targetKey{doc.key, path + "/responses/" + escapeJSONPointer(status)}, response, make(map[targetKey]bool))
				if err != nil {
					return err
				}
				present, err := sseResponseHasItem(resolved)
				if err != nil {
					return err
				}
				guard = guard || present
			}
			if !guard {
				continue
			}
			roots, err := reservedSSESchemaRoots(node, path)
			if err != nil {
				return err
			}
			doc.operations[path] = true
			for _, root := range roots {
				if doc.typedRoots == nil {
					doc.typedRoots = make(map[string]bool)
				}
				doc.typedRoots[root.path] = true
				doc.schemaPositions = nil
				if err := r.scan(doc, root.path, "schemas"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *descriptionResolver) sseResponseNode(at targetKey, node *yaml.Node, seen map[targetKey]bool) (*yaml.Node, error) {
	if seen[at] {
		return nil, nil
	}
	seen[at] = true
	fields, err := sseNodeFields(node)
	if err != nil || fields["$ref"] == nil {
		return node, err
	}
	for _, edge := range r.result.edges {
		if edge.at.doc != at.doc || edge.at.pointer != at.pointer+"/$ref" || edge.kind != "responses" || edge.target == nil {
			continue
		}
		doc := r.result.registry.docs[edge.target.doc]
		target, err := descriptionNode(doc.root, edge.target.pointer)
		if err != nil {
			return nil, err
		}
		return r.sseResponseNode(*edge.target, target, seen)
	}
	return nil, nil
}

func sseResponseHasItem(node *yaml.Node) (bool, error) {
	fields, err := sseNodeFields(node)
	if err != nil {
		return false, err
	}
	content, err := sseNodeFields(fields["content"])
	if err != nil || len(content) != 1 {
		return false, err
	}
	for name, media := range content {
		if normalizedMediaType(name) != "text/event-stream" {
			continue
		}
		fields, err := sseNodeFields(media)
		return fields["itemSchema"] != nil, err
	}
	return false, nil
}

// reservedSSESchemaRoots also classifies URI-key diagnostics before loading.
// This fixed path inventory does not grant admission or interpret examples,
// defaults, mappings, or other extension values as references.
func reservedSSESchemaRoots(node *yaml.Node, path string) ([]sseSchemaRoot, error) {
	var err error
	for _, key := range []string{"x-loom-async", "messages"} {
		fields, fieldErr := sseNodeFields(node)
		if fieldErr != nil {
			return nil, fieldErr
		}
		node = fields[key]
		path += "/" + key
	}
	messages, err := sseNodeFields(node)
	if err != nil {
		return nil, err
	}
	var roots []sseSchemaRoot
	for _, direction := range []string{"inbound", "outbound"} {
		fields, err := sseNodeFields(messages[direction])
		if err != nil {
			return nil, err
		}
		if fields["schema"] != nil {
			roots = append(roots, sseSchemaRoot{fields["schema"], path + "/" + direction + "/schema"})
		}
	}
	return roots, nil
}

func sseNodeFields(node *yaml.Node) (map[string]*yaml.Node, error) {
	if node == nil {
		return nil, nil
	}
	for node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if node.Kind != yaml.MappingNode {
		return nil, nil
	}
	return effectiveYAMLMapping(node, make(map[*yaml.Node]bool))
}
