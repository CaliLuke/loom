package openapiimport

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
)

type (
	descriptionGrammarChild struct {
		field, kind, mode string
	}
	descriptionGrammarWalker struct {
		doc             *document
		edges           []*edge
		hasSchemas      bool
		schemaPositions map[string]struct{}
		uriKeys         map[*yaml.Node]bool
	}
)

var descriptionGrammar = map[string][]descriptionGrammarChild{
	"openapi": {{"paths", "pathItems", "map"}, {"webhooks", "pathItems", "map"}},
	"pathItems": {
		{"parameters", "parameters", "array"}, {"get", "operation", ""},
		{"put", "operation", ""}, {"post", "operation", ""}, {"delete", "operation", ""},
		{"options", "operation", ""}, {"head", "operation", ""}, {"patch", "operation", ""},
		{"trace", "operation", ""}, {"query", "operation", ""}, {"additionalOperations", "operation", "map"},
	},
	"operation": {{"callbacks", "callbacks", "map"}, {"parameters", "parameters", "array"}, {"requestBody", "requestBodies", ""}, {"responses", "responses", "map"}},
	"schemas": {
		{"properties", "schemas", "map"}, {"patternProperties", "schemas", "map"},
		{"$defs", "schemas", "map"}, {"definitions", "schemas", "map"}, {"dependentSchemas", "schemas", "map"},
		{"oneOf", "schemas", "array"}, {"anyOf", "schemas", "array"}, {"allOf", "schemas", "array"}, {"prefixItems", "schemas", "array"},
		{"items", "schemas", ""}, {"additionalProperties", "schemas", ""}, {"unevaluatedProperties", "schemas", ""},
		{"unevaluatedItems", "schemas", ""}, {"contains", "schemas", ""}, {"propertyNames", "schemas", ""},
		{"not", "schemas", ""}, {"if", "schemas", ""}, {"then", "schemas", ""}, {"else", "schemas", ""},
	},
	"parameters":    {{"schema", "schemas", ""}, {"examples", "examples", "map"}, {"content", "media", "map"}},
	"headers":       {{"schema", "schemas", ""}, {"examples", "examples", "map"}, {"content", "media", "map"}},
	"media":         {{"encoding", "encoding", "map"}, {"schema", "schemas", ""}, {"examples", "examples", "map"}, {"content", "media", "map"}},
	"encoding":      {{"headers", "headers", "map"}},
	"requestBodies": {{"headers", "headers", "map"}, {"content", "media", "map"}},
	"responses":     {{"headers", "headers", "map"}, {"content", "media", "map"}},
}

// descriptionEdges visits grammar positions only. Defaults, examples, enums and
// extension values remain data, even when they contain a member named $ref.
func descriptionEdges(doc *document, pointer, kind string) ([]*edge, error) {
	node, err := descriptionNode(doc.root, pointer)
	if err != nil {
		return nil, err
	}
	walker := descriptionGrammarWalker{doc: doc}
	if err := walker.visit(node, pointer, kind); err != nil {
		return nil, fmt.Errorf("walk description grammar: %w", err)
	}
	return walker.edges, nil
}

// descriptionSchemaPositions derives target roles from the same grammar that
// discovers reference edges. Containers and literal data are never schema
// positions, even when they contain schema-shaped mappings. Cache only a
// successful walk of the immutable source document.
func descriptionSchemaPositions(doc *document) (map[string]struct{}, error) {
	if doc.schemaPositions != nil {
		return doc.schemaPositions, nil
	}
	walker := descriptionGrammarWalker{doc: doc, schemaPositions: make(map[string]struct{})}
	if err := walker.visit(doc.root, "", "openapi"); err != nil {
		return nil, err
	}
	doc.schemaPositions = walker.schemaPositions
	return doc.schemaPositions, nil
}

// descriptionURIKeys classifies raw source keys only after alias validation.
// Complete documents have one grammar root. Loose documents may occupy any
// supported object role; the same grammar discovers their URI key occurrences.
func descriptionURIKeys(root *yaml.Node) (map[*yaml.Node]bool, error) {
	keys := make(map[*yaml.Node]bool)
	walker := descriptionGrammarWalker{doc: &document{}, uriKeys: keys}
	fields, err := effectiveYAMLMapping(root, make(map[*yaml.Node]bool))
	if err != nil {
		return nil, err
	}
	if fields["openapi"] != nil || fields["swagger"] != nil || fields["paths"] != nil {
		err = walker.visit(root, "", "openapi")
	} else {
		kinds := []string{"callbacks"}
		for kind := range descriptionGrammar {
			kinds = append(kinds, kind)
		}
		slices.Sort(kinds)
		for _, kind := range kinds {
			if err = walker.visit(root, "", kind); err != nil {
				return nil, err
			}
		}
	}
	return keys, err
}

// sourceNode follows aliases already checked by parseDescriptionTree. Normal
// resolver walks receive canonical nodes; source diagnostic classification keeps
// raw key identities so display policy never changes duplicate-key matching.
func (w *descriptionGrammarWalker) sourceNode(node *yaml.Node) *yaml.Node {
	if w.uriKeys != nil {
		for node != nil && node.Kind == yaml.AliasNode {
			node = node.Alias
		}
	}
	return node
}

func descriptionComponentKind(kind string) bool {
	switch kind {
	case "schemas", "parameters", "requestBodies", "responses", "headers", "examples", "securitySchemes":
		return true
	default:
		return false
	}
}

func (w *descriptionGrammarWalker) add(class, kind, at string, value *yaml.Node) {
	e := &edge{class: class, kind: kind, at: targetKey{w.doc.key, at}, authored: value.Value}
	if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
		e.invalid = true
		e.failure = &Diagnostic{Code: "external-reference", Path: "#" + at, Message: "reference must be a string"}
	}
	w.edges = append(w.edges, e)
}

func (w *descriptionGrammarWalker) visit(node *yaml.Node, path, kind string) error {
	node = w.sourceNode(node)
	if node != nil && kind == "schemas" {
		w.hasSchemas = true
		if w.schemaPositions != nil {
			w.schemaPositions[path] = struct{}{}
		}
	}
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	fields, err := effectiveYAMLMapping(node, make(map[*yaml.Node]bool))
	if err != nil {
		return err
	}
	if ref := fields["$ref"]; ref != nil && (descriptionComponentKind(kind) || kind == "pathItems") {
		class := "component"
		if kind == "pathItems" {
			class = "pathItem"
		}
		w.add(class, kind, path+"/$ref", ref)
	}
	switch kind {
	case "openapi":
		if err := w.components(fields["components"], path+"/components"); err != nil {
			return err
		}
		if err := w.security(fields["security"], path+"/security"); err != nil {
			return err
		}
	case "operation":
		if err := w.security(fields["security"], path+"/security"); err != nil {
			return err
		}
	case "callbacks":
		return w.members(node, path, "pathItems", true)
	case "schemas":
		w.discriminator(fields["discriminator"], path+"/discriminator")
	}
	for _, child := range descriptionGrammar[kind] {
		if err := w.child(fields[child.field], path+"/"+escapeJSONPointer(child.field), child); err != nil {
			return err
		}
	}
	return nil
}

func (w *descriptionGrammarWalker) child(node *yaml.Node, path string, child descriptionGrammarChild) error {
	node = w.sourceNode(node)
	if node == nil {
		return nil
	}
	switch child.mode {
	case "map":
		skipExtensions := child.field == "paths" || child.field == "webhooks" || child.field == "responses" || child.field == "additionalOperations"
		return w.members(node, path, child.kind, skipExtensions)
	case "array":
		if node.Kind == yaml.SequenceNode {
			for i, item := range node.Content {
				if err := w.visit(item, path+"/"+strconv.Itoa(i), child.kind); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return w.visit(node, path, child.kind)
	}
}

func (w *descriptionGrammarWalker) members(node *yaml.Node, path, kind string, skipExtensions bool) error {
	node = w.sourceNode(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	members, err := orderedYAMLMapping(node)
	if err != nil {
		return err
	}
	for _, member := range members {
		name := member.name
		if skipExtensions && (name == "$ref" || strings.HasPrefix(name, "x-")) {
			continue
		}
		if err := w.visit(member.value, path+"/"+escapeJSONPointer(name), kind); err != nil {
			return err
		}
	}
	return nil
}

func (w *descriptionGrammarWalker) components(node *yaml.Node, path string) error {
	node = w.sourceNode(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	members, err := orderedYAMLMapping(node)
	if err != nil {
		return err
	}
	for _, member := range members {
		kind := member.name
		if !descriptionComponentKind(kind) && kind != "pathItems" && kind != "callbacks" {
			continue
		}
		if err := w.members(member.value, path+"/"+kind, kind, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *descriptionGrammarWalker) security(node *yaml.Node, path string) error {
	node = w.sourceNode(node)
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	for i, requirement := range node.Content {
		requirement = w.sourceNode(requirement)
		if requirement.Kind != yaml.MappingNode {
			continue
		}
		if w.uriKeys != nil {
			if err := w.sourceSecurityKeys(requirement); err != nil {
				return err
			}
		}
		for j := 0; j+1 < len(requirement.Content); j += 2 {
			key := requirement.Content[j]
			w.add("securityKey", "securitySchemes", path+"/"+strconv.Itoa(i)+"/"+escapeJSONPointer(key.Value), key)
		}
	}
	return nil
}

func (w *descriptionGrammarWalker) discriminator(node *yaml.Node, path string) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		name, value := node.Content[i].Value, node.Content[i+1]
		if name == "defaultMapping" {
			w.add("discriminatorMapping", "schemas", path+"/defaultMapping", value)
		}
		if name == "mapping" && value.Kind == yaml.MappingNode {
			for j := 0; j+1 < len(value.Content); j += 2 {
				w.add("discriminatorMapping", "schemas", path+"/mapping/"+escapeJSONPointer(value.Content[j].Value), value.Content[j+1])
			}
		}
	}
}

// sourceSecurityKeys retains key node identity across aliases and merge sources.
// Duplicate matching is still performed on raw keys before canonicalization.
func (w *descriptionGrammarWalker) sourceSecurityKeys(node *yaml.Node) error {
	node = w.sourceNode(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if key.Tag != "!!merge" {
			w.uriKeys[key] = true
			continue
		}
		sources, err := yamlMergeSources(node.Content[i+1])
		if err != nil {
			return err
		}
		for _, source := range sources {
			if err := w.sourceSecurityKeys(source); err != nil {
				return err
			}
		}
	}
	return nil
}
