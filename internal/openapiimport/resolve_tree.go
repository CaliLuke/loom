package openapiimport

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/go-yaml/plugin/limit"
)

type duplicateDescriptionKey struct {
	key      *yaml.Node
	previous int
}

var (
	errSchemaResource = errors.New("schema resource identity or dialect is not supported")
	errSchemaDialect  = errors.New("custom JSON Schema dialects are not in the strict import subset")
)

// parseDescriptionTree preserves scalar tokens and tags while resolving YAML
// aliases and merge ownership inside the originating document.
func parseDescriptionTree(data []byte) (*yaml.Node, map[string]numericSource, error) {
	var parsed yaml.Node
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, nil, err
	}
	if len(parsed.Content) != 1 {
		return nil, nil, errors.New("missing source document")
	}
	// Node parsing retains aliases. Apply the parser's own default alias policy
	// before a source walk expands them; do not create an independent threshold.
	policy := limit.New()
	constructs, aliases := 0, 0
	var duplicate *duplicateDescriptionKey
	var validate func(*yaml.Node, bool, map[*yaml.Node]bool) error
	validate = func(node *yaml.Node, inAlias bool, active map[*yaml.Node]bool) error {
		if node == nil || active[node] {
			return errors.New("invalid or cyclic YAML alias")
		}
		constructs++
		if inAlias {
			aliases++
		}
		if err := policy.CheckAlias(aliases, constructs); err != nil {
			return err
		}
		active[node] = true
		defer delete(active, node)
		if node.Kind == yaml.MappingNode {
			if duplicate == nil {
				duplicate = duplicateDescriptionMapping(node)
			}
		}
		if node.Kind == yaml.AliasNode {
			return validate(node.Alias, true, active)
		}
		for _, child := range node.Content {
			if err := validate(child, inAlias, active); err != nil {
				return err
			}
		}
		return nil
	}
	if err := validate(parsed.Content[0], false, make(map[*yaml.Node]bool)); err != nil {
		return nil, nil, err
	}
	if duplicate != nil {
		keys, err := descriptionURIKeys(parsed.Content[0])
		if err != nil {
			return nil, nil, err
		}
		value := sourceDisplayText(duplicate.key.Value)
		if keys[duplicate.key] {
			value = sourceDisplayURI(duplicate.key.Value)
		}
		return nil, nil, fmt.Errorf("mapping key %q at line %d already defined at line %d", value, duplicate.key.Line, duplicate.previous)
	}
	root, err := effectiveDescriptionNode(parsed.Content[0])
	if err != nil {
		return nil, nil, err
	}
	numbers, err := numericSources(data)
	return root, numbers, err
}

// duplicateDescriptionMapping applies the parser's explicit-key identity rule
// before canonicalization can collapse duplicate keys. Node parsing does not
// run constructor checks; decoding values just for validation would lose the
// exact numeric domain this source tree is responsible for preserving. Merge
// source overlap is legal and is resolved separately by effectiveYAMLMapping.
func duplicateDescriptionMapping(node *yaml.Node) *duplicateDescriptionKey {
	type keyID struct {
		kind  yaml.Kind
		value string
	}
	seen := make(map[keyID]int)
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		id := keyID{key.Kind, key.Value}
		if line, exists := seen[id]; exists {
			return &duplicateDescriptionKey{key: key, previous: line}
		}
		seen[id] = key.Line
	}
	return nil
}

func effectiveDescriptionNode(node *yaml.Node) (*yaml.Node, error) {
	if node.Kind == yaml.AliasNode {
		return effectiveDescriptionNode(node.Alias)
	}
	copy := *node
	copy.Anchor, copy.Alias, copy.Content = "", nil, nil
	if node.Kind == yaml.MappingNode {
		members, err := orderedYAMLMapping(node)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			value, err := effectiveDescriptionNode(member.value)
			if err != nil {
				return nil, err
			}
			copy.Content = append(copy.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: member.name}, value)
		}
	} else {
		for _, child := range node.Content {
			value, err := effectiveDescriptionNode(child)
			if err != nil {
				return nil, err
			}
			copy.Content = append(copy.Content, value)
		}
	}
	return &copy, nil
}

func descriptionNode(root *yaml.Node, pointer string) (*yaml.Node, error) {
	if !jsontext.Pointer(pointer).IsValid() {
		return nil, errors.New("invalid target JSON Pointer")
	}
	node := root
	for token := range jsontext.Pointer(pointer).Tokens() {
		if node == nil {
			return nil, errors.New("reference pointer does not resolve")
		}
		switch node.Kind {
		case yaml.MappingNode:
			var next *yaml.Node
			for i := 0; i+1 < len(node.Content); i += 2 {
				if node.Content[i].Value == token {
					next = node.Content[i+1]
					break
				}
			}
			node = next
		case yaml.SequenceNode:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(node.Content) || strconv.Itoa(index) != token {
				return nil, errors.New("reference array pointer does not resolve")
			}
			node = node.Content[index]
		default:
			return nil, errors.New("reference pointer traverses a scalar")
		}
	}
	if node == nil {
		return nil, errors.New("reference pointer does not resolve")
	}
	return node, nil
}

func (r *descriptionResolver) admitTarget(doc *document, target targetKey, kind string) error {
	if err := admitOpenAPITarget(doc, target.pointer, kind, r.result.registry.docs["entry"].version); err != nil {
		return err
	}
	node, err := descriptionNode(doc.root, target.pointer)
	if err != nil {
		return err
	}
	if node.Kind != yaml.MappingNode && !(kind == "schemas" && node.Kind == yaml.ScalarNode && node.Tag == "!!bool") {
		return errors.New("reference target has the wrong object kind")
	}
	if err := admitTargetDialect(doc, node, target.pointer, kind); err != nil {
		return err
	}
	if kind != "schemas" {
		return nil
	}
	return admitSchemaResource(doc, node, target.pointer)
}

// admitOpenAPITarget checks version compatibility and typed target locations.
// Loose documents defer their shape check to the referenced node itself.
func admitOpenAPITarget(doc *document, pointer, kind, entryVersion string) error {
	if doc.version == "" {
		return nil
	}
	version := strings.Split(doc.version, ".")
	parent := strings.Split(entryVersion, ".")
	if len(version) < 2 || len(parent) < 2 || version[0] != parent[0] || version[1] != parent[1] {
		return errors.New("referenced OpenAPI document version mismatch")
	}
	if kind == "schemas" {
		positions, err := descriptionSchemaPositions(doc)
		if err != nil {
			return err
		}
		if _, schema := positions[pointer]; !schema {
			return errors.New("target does not match the required OpenAPI object kind")
		}
		return nil
	}
	parts := strings.Split(pointer, "/")
	component := len(parts) == 4 && parts[1] == "components" && parts[2] == kind
	pathItem := kind == "pathItems" && len(parts) == 3 && (parts[1] == "paths" || parts[1] == "webhooks")
	if !component && !pathItem {
		return errors.New("target does not match the required OpenAPI object kind")
	}
	return nil
}

// admitTargetDialect applies a source dialect to schema-bearing targets only.
// Grammar traversal keeps examples and other literal data outside that policy.
func admitTargetDialect(doc *document, node *yaml.Node, pointer, kind string) error {
	if doc.key == "entry" || supportedSchemaDialect(doc.version, doc.dialect) {
		return nil
	}
	walker := descriptionGrammarWalker{doc: doc}
	if err := walker.visit(node, pointer, kind); err != nil {
		return err
	}
	if walker.hasSchemas {
		return errSchemaDialect
	}
	return nil
}

// admitSchemaResource rejects resource boundaries along schema ancestry and
// dynamic references at the target, preserving loose-document grammar contexts.
func admitSchemaResource(doc *document, node *yaml.Node, targetPointer string) error {
	contexts := schemaPointerContexts(targetPointer)
	pointer := ""
	for {
		ancestor, err := descriptionNode(doc.root, pointer)
		if err != nil {
			return err
		}
		if (pointer == targetPointer || contexts[pointer] == "schema") && descriptionResourceKeyword(ancestor) ||
			contexts[pointer] == "" && doc.version == "" && descriptionResourceDeclaration(ancestor) {
			return errSchemaResource
		}
		if pointer == targetPointer {
			break
		}
		rest := strings.TrimPrefix(targetPointer, pointer+"/")
		next, _, _ := strings.Cut(rest, "/")
		pointer += "/" + next
	}
	if descriptionDynamicReference(node) {
		return fmt.Errorf("%w: $dynamicRef", errSchemaResource)
	}
	return nil
}

// schemaPointerContexts finds the longest grammar-valid schema ancestry of a
// target in a loose document. Forward parsing distinguishes a keyword from a
// property with the same name; property dictionaries are never Schema Objects.
func schemaPointerContexts(pointer string) map[string]string {
	if pointer == "" {
		return map[string]string{"": "schema"}
	}
	tokens := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	for start := range len(tokens) {
		if contexts, ok := schemaPointerChain(tokens, start); ok {
			return contexts
		}
	}
	return map[string]string{pointer: "schema"}
}

func schemaPointerChain(tokens []string, start int) (map[string]string, bool) {
	contexts := make(map[string]string)
	if start > 0 {
		contexts["/"+strings.Join(tokens[:start], "/")] = "schema"
	}
	for position := start; position < len(tokens); {
		var child *descriptionGrammarChild
		for i := range descriptionGrammar["schemas"] {
			candidate := &descriptionGrammar["schemas"][i]
			if candidate.field == tokens[position] {
				child = candidate
				break
			}
		}
		if child == nil {
			return nil, false
		}
		position++
		if child.mode != "" {
			if position == len(tokens) {
				return nil, false
			}
			if child.mode == "array" {
				index, err := strconv.Atoi(tokens[position])
				if err != nil || index < 0 || strconv.Itoa(index) != tokens[position] {
					return nil, false
				}
			}
			contexts["/"+strings.Join(tokens[:position], "/")] = "container"
			position++
		}
		contexts["/"+strings.Join(tokens[:position], "/")] = "schema"
	}
	return contexts, true
}

// A loose document can declare a schema resource outside the target's known
// grammar ancestry. Only string declarations identify that boundary: a component
// dictionary entry named $id with a schema value is ordinary data.
func descriptionResourceDeclaration(node *yaml.Node) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i+1].Kind != yaml.ScalarNode || node.Content[i+1].Tag != "!!str" {
			continue
		}
		switch node.Content[i].Value {
		case "$id", "$anchor", "$dynamicAnchor", "$schema":
			return true
		}
	}
	return false
}

func descriptionResourceKeyword(node *yaml.Node) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		switch node.Content[i].Value {
		case "$id", "$anchor", "$dynamicAnchor", "$schema":
			return true
		}
	}
	return false
}

func descriptionDynamicReference(node *yaml.Node) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		name, value := node.Content[i].Value, node.Content[i+1]
		if name == "$dynamicRef" {
			return true
		}
		switch name {
		case "properties", "patternProperties", "$defs", "definitions", "dependentSchemas":
			for j := 1; j < len(value.Content); j += 2 {
				if descriptionDynamicReference(value.Content[j]) {
					return true
				}
			}
		case "oneOf", "anyOf", "allOf", "prefixItems":
			for _, child := range value.Content {
				if descriptionDynamicReference(child) {
					return true
				}
			}
		case "items", "additionalProperties", "unevaluatedProperties", "unevaluatedItems", "not", "contains", "if", "then", "else", "propertyNames":
			if descriptionDynamicReference(value) {
				return true
			}
		}
	}
	return false
}
