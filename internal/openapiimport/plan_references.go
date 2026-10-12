package openapiimport

import (
	"fmt"
	"sort"
)

type (
	// schemaReferencePlan is frozen before rendering. Occurrence keys retain the
	// declaration owner: cloned error declarations may reuse a source schema.
	schemaReferencePlan struct {
		lookups map[referenceOccurrenceKey]string
	}
	referenceOccurrenceKey struct {
		owner string
		path  string
	}
	referenceOccurrence struct {
		key         referenceOccurrenceKey
		target      string
		name        string
		constructor string
		eligible    bool
	}
	referencePlanner struct {
		schemas     map[string]NamedSchema
		occurrences []referenceOccurrence
		diagnostics Diagnostics
	}
)

func planSchemaReferences(document *Document, schemas map[string]NamedSchema, operations []operationPlan) (*schemaReferencePlan, Diagnostics) {
	p := referencePlanner{schemas: schemas}
	for _, named := range document.Components.Schemas {
		owner := componentDeclaration(named.Name)
		p.schema(named.Schema, owner, owner, "Type", false)
	}
	for operationIndex, operation := range operations {
		for _, response := range operation.failures {
			if !response.cloneErrorType {
				continue
			}
			path := errorCloneSchemaPath(operationIndex, response.status)
			p.schema(response.response.Schema, errorCloneDeclaration(operationIndex, response.status), path, "Extend", false)
		}
	}
	if len(p.diagnostics) > 0 {
		return nil, p.diagnostics.sorted()
	}
	plan := &schemaReferencePlan{lookups: make(map[referenceOccurrenceKey]string)}
	components := referenceComponents(p.occurrences)
	remaining := make([]referenceOccurrence, 0, len(p.occurrences))
	for _, occurrence := range p.occurrences {
		if occurrence.eligible && components[occurrence.key.owner] == components[occurrence.target] {
			plan.lookups[occurrence.key] = occurrence.name
		} else {
			remaining = append(remaining, occurrence)
		}
	}
	components = referenceComponents(remaining)
	for _, occurrence := range remaining {
		if components[occurrence.key.owner] == components[occurrence.target] {
			p.diagnostics.add("schema-reference-initialization", occurrence.key.path,
				fmt.Sprintf("%s requires a concrete reference from %s to %s in an initializer cycle; this recursive constructor graph is outside the import subset", occurrence.constructor, occurrence.key.owner, occurrence.target))
		}
	}
	return plan, p.diagnostics.sorted()
}

func componentDeclaration(name string) string {
	return "#/components/schemas/" + escapeJSONPointer(name)
}

func errorCloneSchemaPath(operation int, status string) string {
	return fmt.Sprintf("#/operations/%d/responses/%s/content/schema", operation, escapeJSONPointer(status))
}

func errorCloneDeclaration(operation int, status string) string {
	return errorCloneSchemaPath(operation, status) + "/error-clone"
}

func (p *referencePlanner) schema(schema *Schema, owner, path, constructor string, delayed bool) {
	if schema == nil {
		return
	}
	if schema.Ref != "" {
		name, err := localComponentReferenceName(schema.Ref, "#/components/schemas/")
		if err != nil {
			p.diagnostics.add("schema-reference", path, err.Error())
			return
		}
		if _, ok := p.schemas[name]; !ok {
			p.diagnostics.add("schema-reference", path, fmt.Sprintf("component %q does not resolve", name))
			return
		}
		p.occurrences = append(p.occurrences, referenceOccurrence{
			key: referenceOccurrenceKey{owner: owner, path: path}, target: componentDeclaration(name),
			name: name, constructor: constructor, eligible: delayed || constructor == "OneOf",
		})
		return
	}
	for index, branch := range schema.OneOf {
		p.schema(branch, owner, fmt.Sprintf("%s/oneOf/%d", path, index), "OneOf", delayed)
	}
	if schema.Items != nil {
		p.schema(schema.Items, owner, path+"/items", "ArrayOf", delayed)
	}
	if schema.AdditionalProperties != nil {
		p.schema(schema.AdditionalProperties.Schema, owner, path+"/additionalProperties", "MapOf", delayed)
	}
	for index, base := range schema.Bases {
		p.schema(base, owner, fmt.Sprintf("%s/bases/%d", path, index), "Extend", false)
	}
	for _, property := range schema.Properties {
		p.schema(property.Schema, owner, path+"/properties/"+escapeJSONPointer(property.Name), "Attribute", true)
	}
}

// referenceComponents assigns strongly connected component identities without
// collapsing occurrences. Parallel mandatory and eligible edges remain distinct
// until the selected occurrences have been removed.
func referenceComponents(occurrences []referenceOccurrence) map[string]int {
	edges := make(map[string][]string)
	for _, occurrence := range occurrences {
		edges[occurrence.key.owner] = append(edges[occurrence.key.owner], occurrence.target)
		if _, exists := edges[occurrence.target]; !exists {
			edges[occurrence.target] = nil
		}
	}
	nodes := make([]string, 0, len(edges))
	for node := range edges {
		nodes = append(nodes, node)
		sort.Strings(edges[node])
	}
	sort.Strings(nodes)
	indices, low, active := make(map[string]int), make(map[string]int), make(map[string]bool)
	components := make(map[string]int)
	var stack []string
	next, component := 1, 0
	var visit func(string)
	visit = func(node string) {
		indices[node], low[node] = next, next
		next++
		stack = append(stack, node)
		active[node] = true
		for _, target := range edges[node] {
			if indices[target] == 0 {
				visit(target)
				low[node] = min(low[node], low[target])
			} else if active[target] {
				low[node] = min(low[node], indices[target])
			}
		}
		if low[node] != indices[node] {
			return
		}
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			active[last] = false
			components[last] = component
			if last == node {
				break
			}
		}
		component++
	}
	for _, node := range nodes {
		if indices[node] == 0 {
			visit(node)
		}
	}
	return components
}
