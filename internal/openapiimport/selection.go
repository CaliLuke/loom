package openapiimport

import (
	"fmt"
	pathpkg "path"
	"sort"
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

type (
	// Selection identifies the union of OpenAPI operations to import.
	Selection struct {
		// Tags selects operations with at least one exact tag match.
		Tags []string
		// PathPrefixes selects operations whose path starts with a prefix.
		PathPrefixes []string
		// Paths selects operations whose path matches a path.Match pattern.
		Paths []string
	}

	// TagSummary reports the number of operations and paths assigned to a tag.
	TagSummary struct {
		// Name is the exact OpenAPI tag name.
		Name string
		// Operations is the number of operations assigned to the tag.
		Operations int
		// Paths is the number of distinct paths assigned to the tag.
		Paths int
	}

	// SelectionReport describes the available tags and paths excluded by tag filters.
	SelectionReport struct {
		// Tags contains deterministic operation and path counts by tag.
		Tags []TagSummary
		// UnclaimedPaths lists paths that do not use a requested tag.
		UnclaimedPaths []string
	}

	componentClosure struct {
		schemas       map[string]struct{}
		parameters    map[string]struct{}
		requestBodies map[string]struct{}
		responses     map[string]struct{}
		headers       map[string]struct{}
		tags          map[string]struct{}
	}

	componentPruner struct {
		closure       componentClosure
		schemas       map[string]*Schema
		parameters    map[string]Parameter
		requestBodies map[string]RequestBody
		responses     map[string]Response
		headers       map[string]Header
	}

	selectedOperation struct {
		method    string
		operation *v3.Operation
	}
)

// Active reports whether the selection contains an operation filter.
func (s Selection) Active() bool {
	return len(s.Tags) > 0 || len(s.PathPrefixes) > 0 || len(s.Paths) > 0
}

// Validate checks path patterns before the importer reads the document.
func (s Selection) Validate() error {
	for _, pattern := range s.Paths {
		if _, err := pathpkg.Match(pattern, ""); err != nil {
			return fmt.Errorf("invalid OpenAPI path pattern %q: %w", pattern, err)
		}
	}
	return nil
}

func (s Selection) matches(path string, tags []string) bool {
	if !s.Active() {
		return true
	}
	if s.matchesTag(tags) {
		return true
	}
	for _, prefix := range s.PathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, pattern := range s.Paths {
		if matched, _ := pathpkg.Match(pattern, path); matched {
			return true
		}
	}
	return false
}

func (s Selection) matchesTag(tags []string) bool {
	for _, selected := range s.Tags {
		for _, tag := range tags {
			if tag == selected {
				return true
			}
		}
	}
	return false
}

func (a *analyzer) operations(paths *v3.Paths, components Components, declaredTags []string) []Operation {
	if paths == nil {
		a.report = newSelectionReport(nil, nil, declaredTags, nil)
		return nil
	}
	a.extensions("#/paths", paths.Extensions)
	var operations []Operation
	tagPaths := make(map[string]map[string]struct{})
	tagOperations := make(map[string]int)
	var unclaimedPaths []string
	for path, item := range paths.PathItems.FromOldest() {
		base := "#/paths/" + escapeJSONPointer(path)
		var selected []selectedOperation
		tagClaimed := false
		for method, operation := range item.GetOperations().FromOldest() {
			operationTags := make(map[string]struct{}, len(operation.Tags))
			for _, tag := range operation.Tags {
				if _, seen := operationTags[tag]; seen {
					continue
				}
				operationTags[tag] = struct{}{}
				tagOperations[tag]++
				if tagPaths[tag] == nil {
					tagPaths[tag] = make(map[string]struct{})
				}
				tagPaths[tag][path] = struct{}{}
			}
			if a.selection.matchesTag(operation.Tags) {
				tagClaimed = true
			}
			if a.selection.matches(path, operation.Tags) {
				selected = append(selected, selectedOperation{method: method, operation: operation})
			}
		}
		if len(a.selection.Tags) > 0 && !tagClaimed {
			unclaimedPaths = append(unclaimedPaths, path)
		}
		if len(selected) == 0 {
			continue
		}
		if item.Reference != "" {
			a.unsupported("path-reference", base, "path item references are not in the strict import subset")
		}
		if item.Summary != "" || item.Description != "" {
			a.unsupported("path-metadata", base, "path item summaries and descriptions are not in the strict import subset")
		}
		if len(item.Servers) > 0 {
			a.unsupported("servers", base+"/servers", "path servers are not in the strict import subset")
		}
		a.extensions(base, item.Extensions)
		for _, selectedOperation := range selected {
			operations = append(operations, a.operation(
				strings.ToUpper(selectedOperation.method),
				path,
				item.Parameters,
				selectedOperation.operation,
				base+"/"+selectedOperation.method,
				base+"/parameters",
				components,
			))
		}
	}
	a.report = newSelectionReport(tagOperations, tagPaths, declaredTags, unclaimedPaths)
	return operations
}

func newSelectionReport(
	tagOperations map[string]int,
	tagPaths map[string]map[string]struct{},
	declaredTags, unclaimedPaths []string,
) SelectionReport {
	names := make([]string, 0, len(tagOperations)+len(declaredTags))
	seen := make(map[string]struct{}, len(tagOperations)+len(declaredTags))
	for name := range tagOperations {
		names = append(names, name)
		seen[name] = struct{}{}
	}
	for _, name := range declaredTags {
		if _, ok := seen[name]; ok {
			continue
		}
		names = append(names, name)
		seen[name] = struct{}{}
	}
	sort.Strings(names)
	tags := make([]TagSummary, 0, len(names))
	for _, name := range names {
		tags = append(tags, TagSummary{
			Name:       name,
			Operations: tagOperations[name],
			Paths:      len(tagPaths[name]),
		})
	}
	sort.Strings(unclaimedPaths)
	return SelectionReport{Tags: tags, UnclaimedPaths: unclaimedPaths}
}

func pruneComponents(document *Document) componentClosure {
	pruner := newComponentPruner(document.Components)
	for _, operation := range document.Operations {
		pruner.visitOperation(operation)
	}
	pruner.retain(document)
	return pruner.closure
}

func newComponentPruner(components Components) *componentPruner {
	pruner := &componentPruner{
		closure: componentClosure{
			schemas:       make(map[string]struct{}),
			parameters:    make(map[string]struct{}),
			requestBodies: make(map[string]struct{}),
			responses:     make(map[string]struct{}),
			headers:       make(map[string]struct{}),
		},
		schemas:       make(map[string]*Schema, len(components.Schemas)),
		parameters:    make(map[string]Parameter, len(components.Parameters)),
		requestBodies: make(map[string]RequestBody, len(components.RequestBodies)),
		responses:     make(map[string]Response, len(components.Responses)),
		headers:       make(map[string]Header, len(components.Headers)),
	}
	for _, named := range components.Schemas {
		pruner.schemas[named.Name] = named.Schema
	}
	for _, named := range components.Parameters {
		pruner.parameters[named.Name] = named.Parameter
	}
	for _, named := range components.RequestBodies {
		pruner.requestBodies[named.Name] = named.RequestBody
	}
	for _, named := range components.Responses {
		pruner.responses[named.Name] = named.Response
	}
	for _, named := range components.Headers {
		pruner.headers[named.Name] = named.Header
	}
	return pruner
}

func (p *componentPruner) visitOperation(operation Operation) {
	for _, parameter := range operation.Parameters {
		p.visitParameter(parameter)
	}
	if operation.RequestBody != nil {
		p.visitRequestBody(*operation.RequestBody)
	}
	for _, response := range operation.Responses {
		p.visitResponse(response.Response)
	}
}

func (p *componentPruner) visitSchema(schema *Schema) {
	if schema == nil {
		return
	}
	if name, ok := componentReferenceName(schema.Ref, "#/components/schemas/"); ok {
		if _, seen := p.closure.schemas[name]; seen {
			return
		}
		p.closure.schemas[name] = struct{}{}
		p.visitSchema(p.schemas[name])
		return
	}
	for _, property := range schema.Properties {
		p.visitSchema(property.Schema)
	}
	p.visitSchema(schema.Items)
	if schema.AdditionalProperties != nil {
		p.visitSchema(schema.AdditionalProperties.Schema)
	}
}

func (p *componentPruner) visitParameter(parameter Parameter) {
	if name, ok := componentReferenceName(parameter.Ref, "#/components/parameters/"); ok {
		if _, seen := p.closure.parameters[name]; seen {
			return
		}
		p.closure.parameters[name] = struct{}{}
		p.visitParameter(p.parameters[name])
		return
	}
	p.visitSchema(parameter.Schema)
}

func (p *componentPruner) visitRequestBody(body RequestBody) {
	if name, ok := componentReferenceName(body.Ref, "#/components/requestBodies/"); ok {
		if _, seen := p.closure.requestBodies[name]; seen {
			return
		}
		p.closure.requestBodies[name] = struct{}{}
		p.visitRequestBody(p.requestBodies[name])
		return
	}
	p.visitSchema(body.Schema)
}

func (p *componentPruner) visitHeader(header Header) {
	if name, ok := componentReferenceName(header.Ref, "#/components/headers/"); ok {
		if _, seen := p.closure.headers[name]; seen {
			return
		}
		p.closure.headers[name] = struct{}{}
		p.visitHeader(p.headers[name])
		return
	}
	p.visitSchema(header.Schema)
}

func (p *componentPruner) visitResponse(response Response) {
	if name, ok := componentReferenceName(response.Ref, "#/components/responses/"); ok {
		if _, seen := p.closure.responses[name]; seen {
			return
		}
		p.closure.responses[name] = struct{}{}
		p.visitResponse(p.responses[name])
		return
	}
	p.visitSchema(response.Schema)
	for _, named := range response.Headers {
		p.visitHeader(named.Header)
	}
}

func (p *componentPruner) retain(document *Document) {
	document.Components.Schemas = retainNamedSchemas(document.Components.Schemas, p.closure.schemas)
	assignSchemaNames(document.Components.Schemas)
	document.Components.Parameters = retainNamedParameters(document.Components.Parameters, p.closure.parameters)
	document.Components.RequestBodies = retainNamedRequestBodies(document.Components.RequestBodies, p.closure.requestBodies)
	document.Components.Responses = retainNamedResponses(document.Components.Responses, p.closure.responses)
	document.Components.Headers = retainNamedHeaders(document.Components.Headers, p.closure.headers)
	usedTags := make(map[string]struct{})
	for _, operation := range document.Operations {
		for _, tag := range operation.Tags {
			usedTags[tag] = struct{}{}
		}
	}
	document.Tags = retainStrings(document.Tags, usedTags)
	p.closure.tags = usedTags
}

func componentReferenceName(ref, prefix string) (string, bool) {
	if ref == "" {
		return "", false
	}
	name, err := localComponentReferenceName(ref, prefix)
	return name, err == nil
}

func filterSelectionDiagnostics(diagnostics Diagnostics, closure componentClosure) Diagnostics {
	filtered := make(Diagnostics, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Path == "#/components" || diagnostic.Path == "#/components/securitySchemes" {
			continue
		}
		if componentDiagnosticRetained(diagnostic.Path, "#/components/schemas/", closure.schemas) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/parameters/", closure.parameters) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/requestBodies/", closure.requestBodies) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/responses/", closure.responses) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/headers/", closure.headers) &&
			componentDiagnosticRetained(diagnostic.Path, "#/tags/", closure.tags) {
			filtered = append(filtered, diagnostic)
		}
	}
	return filtered
}

func componentDiagnosticRetained(path, prefix string, retained map[string]struct{}) bool {
	if !strings.HasPrefix(path, prefix) {
		return true
	}
	remainder := strings.TrimPrefix(path, prefix)
	segment := strings.SplitN(remainder, "/", 2)[0]
	name := strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
	_, ok := retained[name]
	return ok
}

func retainNamedSchemas(values []NamedSchema, retained map[string]struct{}) []NamedSchema {
	result := make([]NamedSchema, 0, len(retained))
	for _, value := range values {
		if _, ok := retained[value.Name]; ok {
			result = append(result, value)
		}
	}
	return result
}

func retainNamedParameters(values []NamedParameter, retained map[string]struct{}) []NamedParameter {
	result := make([]NamedParameter, 0, len(retained))
	for _, value := range values {
		if _, ok := retained[value.Name]; ok {
			result = append(result, value)
		}
	}
	return result
}

func retainNamedRequestBodies(values []NamedRequestBody, retained map[string]struct{}) []NamedRequestBody {
	result := make([]NamedRequestBody, 0, len(retained))
	for _, value := range values {
		if _, ok := retained[value.Name]; ok {
			result = append(result, value)
		}
	}
	return result
}

func retainNamedResponses(values []NamedResponse, retained map[string]struct{}) []NamedResponse {
	result := make([]NamedResponse, 0, len(retained))
	for _, value := range values {
		if _, ok := retained[value.Name]; ok {
			result = append(result, value)
		}
	}
	return result
}

func retainNamedHeaders(values []NamedHeader, retained map[string]struct{}) []NamedHeader {
	result := make([]NamedHeader, 0, len(retained))
	for _, value := range values {
		if _, ok := retained[value.Name]; ok {
			result = append(result, value)
		}
	}
	return result
}

func retainStrings(values []string, retained map[string]struct{}) []string {
	result := make([]string, 0, len(retained))
	for _, value := range values {
		if _, ok := retained[value]; ok {
			result = append(result, value)
		}
	}
	return result
}
