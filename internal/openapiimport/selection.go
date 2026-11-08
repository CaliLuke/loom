package openapiimport

import (
	"encoding/json/jsontext"
	"fmt"
	pathpkg "path"
	"slices"
	"sort"
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"

	"github.com/CaliLuke/loom/internal/openapiref"
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
		// UnresolvedPaths lists Path Items whose operation and tag membership is unknown.
		UnresolvedPaths []string
	}

	componentClosure struct {
		namingError     error
		examples        map[string]struct{}
		securitySchemes map[string]struct{}
		schemas         map[string]struct{}
		parameters      map[string]struct{}
		requestBodies   map[string]struct{}
		responses       map[string]struct{}
		headers         map[string]struct{}
		tags            map[string]struct{}
	}

	componentPruner struct {
		weakChildren      map[*Discriminator][]string
		schemaOccurrences map[*Schema]struct{}
		// closure owns retained identity; bodies records components whose
		// normalized body and raw source were walked, keyed by kind and name.
		closure       componentClosure
		bodies        map[string]struct{}
		constructors  map[*Schema]string
		schemas       map[string]*Schema
		parameters    map[string]Parameter
		requestBodies map[string]RequestBody
		responses     map[string]Response
		headers       map[string]Header
		// bundle supplies raw source references and their finalized component
		// index; sources records the source prefixes this pruner walked.
		bundle  *bundledDescription
		sources map[string]bool
	}

	selectedOperation struct {
		method     string
		sourcePath string
		operation  *v3.Operation
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
	a.unsupportedExtensions("#/paths", paths.Extensions)
	var operations []Operation
	tagPaths := make(map[string]map[string]struct{})
	tagOperations := make(map[string]int)
	var unclaimedPaths []string
	for path, item := range paths.PathItems.FromOldest() {
		if a.bundle != nil && slices.Contains(a.bundle.unresolvedPaths, path) {
			continue
		}
		base := "#/paths/" + escapeJSONPointer(path)
		selected, tagClaimed := a.selectPathOperations(path, item, tagPaths, tagOperations)
		if len(a.selection.Tags) > 0 && !tagClaimed {
			unclaimedPaths = append(unclaimedPaths, path)
		}
		if len(selected) == 0 {
			continue
		}
		if item.Summary != "" || item.Description != "" {
			a.unsupported("path-metadata", base, "path item summaries and descriptions are not in the strict import subset")
		}
		if len(item.Servers) > 0 {
			a.unsupported("servers", base+"/servers", "path servers are not in the strict import subset")
		}
		a.unsupportedExtensions(base, item.Extensions)
		for _, selectedOperation := range selected {
			operations = append(operations, a.operation(
				strings.ToUpper(selectedOperation.method),
				path,
				item.Parameters,
				selectedOperation.operation,
				selectedOperation.sourcePath,
				base+"/parameters",
				components,
			))
		}
	}
	a.report = newSelectionReport(tagOperations, tagPaths, declaredTags, unclaimedPaths)
	return operations
}

func (a *analyzer) selectPathOperations(path string, item *v3.PathItem, tagPaths map[string]map[string]struct{}, tagOperations map[string]int) ([]selectedOperation, bool) {
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
			sourcePath := "#/paths/" + escapeJSONPointer(path) + "/" + escapeJSONPointer(method)
			if item.AdditionalOperations != nil && item.AdditionalOperations.GetOrZero(method) == operation {
				sourcePath = "#/paths/" + escapeJSONPointer(path) + "/additionalOperations/" + escapeJSONPointer(method)
			}
			selected = append(selected, selectedOperation{method: method, sourcePath: sourcePath, operation: operation})
		}
	}
	return selected, tagClaimed
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

// pruneComponents retains the closure of the walk that weak-child projection
// converged on; the selected roots are not walked a second time.
func pruneComponents(document *Document) componentClosure {
	pruner := projectWeakChildren(document)
	if !slices.ContainsFunc(document.Operations, func(operation Operation) bool {
		return !operation.SecurityDefined
	}) {
		document.Security, document.SecurityDefined = nil, false
	}
	pruner.retain(document)
	document.retainedClosure = &pruner.closure
	return pruner.closure
}

// walkSelectedRoots visits every selected operation, its source-only grammar
// dependencies and effective security under the supplied weak ParentHint edges.
func walkSelectedRoots(document *Document, weakChildren map[*Discriminator][]string) *componentPruner {
	pruner := newDocumentPruner(document)
	pruner.weakChildren = weakChildren
	for _, operation := range document.Operations {
		pruner.visitOperation(operation)
		if document.bundle != nil {
			pruner.visitSourceOperation(document.bundle, operation)
		}
		if operation.SecurityDefined {
			pruner.visitSecurity(operation.Security)
		} else if document.SecurityDefined {
			pruner.visitSecurity(document.Security)
			if document.bundle != nil {
				pruner.visitSourcePrefix(document.bundle, "#/security", pruner.sources)
			}
		}
	}
	return pruner
}

func newComponentPruner(components Components) *componentPruner {
	pruner := &componentPruner{
		schemaOccurrences: make(map[*Schema]struct{}),
		bodies:            make(map[string]struct{}),
		sources:           make(map[string]bool),
		closure: componentClosure{
			examples: make(map[string]struct{}), securitySchemes: make(map[string]struct{}),
			schemas:       make(map[string]struct{}),
			parameters:    make(map[string]struct{}),
			requestBodies: make(map[string]struct{}),
			responses:     make(map[string]struct{}),
			headers:       make(map[string]struct{}),
		},
		constructors:  make(map[*Schema]string),
		schemas:       make(map[string]*Schema, len(components.Schemas)),
		parameters:    make(map[string]Parameter, len(components.Parameters)),
		requestBodies: make(map[string]RequestBody, len(components.RequestBodies)),
		responses:     make(map[string]Response, len(components.Responses)),
		headers:       make(map[string]Header, len(components.Headers)),
	}
	for _, named := range components.Schemas {
		pruner.schemas[named.Name] = named.Schema
		if named.constructionOnly {
			pruner.constructors[named.Schema] = named.Name
		}
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

// newDocumentPruner walks a document's components together with its bundled raw
// source. Selection, child qualification and occurrence inventories all use it,
// so every retained component contributes both closures.
func newDocumentPruner(document *Document) *componentPruner {
	pruner := newComponentPruner(document.Components)
	pruner.bundle = document.bundle
	return pruner
}

// documentSchemaOccurrences inventories retained schemas through the same walk
// used for component closure, including inline schemas, shared edges and cycles.
func documentSchemaOccurrences(document *Document) map[*Schema]struct{} {
	p := newDocumentPruner(document)
	for _, named := range document.Components.Schemas {
		p.visitSchema(named.Schema)
	}
	for _, named := range document.Components.Parameters {
		p.visitParameter(named.Parameter)
	}
	for _, named := range document.Components.RequestBodies {
		p.visitRequestBody(named.RequestBody)
	}
	for _, named := range document.Components.Responses {
		p.visitResponse(named.Response)
	}
	for _, named := range document.Components.Headers {
		p.visitHeader(named.Header)
	}
	for _, operation := range document.Operations {
		p.visitOperation(operation)
	}
	return p.schemaOccurrences
}

func (p *componentPruner) visitOperation(operation Operation) {
	for _, schema := range operation.asyncSchemas {
		p.visitSchema(schema.value)
	}
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
	if _, visited := p.schemaOccurrences[schema]; visited {
		return
	}
	p.schemaOccurrences[schema] = struct{}{}
	if name, ok := p.constructors[schema]; ok {
		p.retainComponent("schemas", name)
	}
	p.visitExamples(schema.Examples)
	if hint := schema.Discriminator; hint != nil {
		if hint.DefaultMapping != "" {
			p.visitSchema(&Schema{Ref: hint.DefaultMapping})
		}
		if hint.Parent {
			for _, name := range p.weakChildren[hint] {
				p.visitSchema(&Schema{Ref: componentDeclaration(name)})
			}
			for _, entry := range hint.Explicit {
				for name, target := range p.schemas {
					if target == entry.Target {
						p.visitSchema(&Schema{Ref: componentDeclaration(name)})
						break
					}
				}
			}
		}
	}
	if name, ok := componentReferenceName(schema.Ref, "#/components/schemas/"); ok {
		p.retainComponent("schemas", name)
		return
	}
	for _, base := range schema.Bases {
		p.visitSchema(base)
	}
	for _, branch := range schema.OneOf {
		p.visitSchema(branch)
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
		p.retainComponent("parameters", name)
		return
	}
	p.visitSchema(parameter.Schema)
}

func (p *componentPruner) visitRequestBody(body RequestBody) {
	p.visitExamples(body.Examples)
	if name, ok := componentReferenceName(body.Ref, "#/components/requestBodies/"); ok {
		p.retainComponent("requestBodies", name)
		return
	}
	p.visitSchema(body.Schema)
}

func (p *componentPruner) visitHeader(header Header) {
	if name, ok := componentReferenceName(header.Ref, "#/components/headers/"); ok {
		p.retainComponent("headers", name)
		return
	}
	p.visitSchema(header.Schema)
}

func (p *componentPruner) visitResponse(response Response) {
	p.visitExamples(response.Examples)
	if name, ok := componentReferenceName(response.Ref, "#/components/responses/"); ok {
		p.retainComponent("responses", name)
		return
	}
	p.visitSchema(response.Schema)
	for _, named := range response.Headers {
		p.visitHeader(named.Header)
	}
}

// retainComponent records a reusable component's retained identity. On first
// retention it walks the component's normalized body and, when the pruner has a
// bundle, its raw source. Analysis adds normalized-only edges such as promoted
// declarations and discriminator children, while normalization can drop a raw
// reference, so retention follows the closure over both inventories and neither
// stands in for the other. It reports whether kind is a retained component kind.
func (p *componentPruner) retainComponent(kind, name string) bool {
	var retained map[string]struct{}
	var body func()
	switch kind {
	case "schemas":
		retained, body = p.closure.schemas, func() {
			p.visitSchema(p.schemas[name])
		}
	case "parameters":
		retained, body = p.closure.parameters, func() {
			p.visitParameter(p.parameters[name])
		}
	case "requestBodies":
		retained, body = p.closure.requestBodies, func() {
			p.visitRequestBody(p.requestBodies[name])
		}
	case "responses":
		retained, body = p.closure.responses, func() {
			p.visitResponse(p.responses[name])
		}
	case "headers":
		retained, body = p.closure.headers, func() {
			p.visitHeader(p.headers[name])
		}
	case "examples":
		retained = p.closure.examples
	case "securitySchemes":
		retained = p.closure.securitySchemes
	default:
		return false
	}
	retained[name] = struct{}{}
	key := kind + "/" + name
	if _, walked := p.bodies[key]; walked {
		return true
	}
	p.bodies[key] = struct{}{}
	if body != nil {
		body()
	}
	if p.bundle != nil {
		p.visitSourcePrefix(p.bundle, componentSource(kind, name), p.sources)
	}
	return true
}

func (p *componentPruner) retain(document *Document) {
	document.Components.Schemas = retainNamedSchemas(document.Components.Schemas, p.closure.schemas)
	document.Components.SecuritySchemes = slices.DeleteFunc(slices.Clone(document.Components.SecuritySchemes), func(scheme SecurityScheme) bool {
		_, retained := p.closure.securitySchemes[scheme.Name]
		return !retained
	})
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
	usedTags = tagParentClosure(usedTags, document.TagMetadata)
	document.Tags = retainStrings(document.Tags, usedTags)
	document.TagMetadata = retainTags(document.TagMetadata, usedTags)
	p.closure.tags = usedTags
	// A subset may still be unrenderable. Carry the naming error with its
	// closure so diagnostic filtering reports it after source admission causes.
	p.closure.namingError = allocateDesignNames(document)
}

func tagParentClosure(retained map[string]struct{}, metadata []Tag) map[string]struct{} {
	parents := make(map[string]string, len(metadata))
	for _, tag := range metadata {
		parents[tag.Name] = tag.Parent
	}

	queue := make([]string, 0, len(retained))
	for name := range retained {
		queue = append(queue, name)
	}
	for index := 0; index < len(queue); index++ {
		parent := parents[queue[index]]
		if parent == "" {
			continue
		}
		if _, seen := retained[parent]; seen {
			continue
		}
		retained[parent] = struct{}{}
		queue = append(queue, parent)
	}
	return retained
}

func retainTags(source []Tag, retained map[string]struct{}) []Tag {
	result := make([]Tag, 0, len(source))
	for _, tag := range source {
		if _, ok := retained[tag.Name]; ok {
			result = append(result, tag)
		}
	}
	return result
}

func componentReferenceName(ref, prefix string) (string, bool) {
	if ref == "" {
		return "", false
	}
	name, err := openapiref.Name(ref, prefix)
	return name, err == nil
}

func filterSelectionDiagnostics(diagnostics Diagnostics, closure componentClosure) Diagnostics {
	filtered := make(Diagnostics, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Path == "#/components" || diagnostic.Path == "#/components/securitySchemes" {
			continue
		}
		if componentDiagnosticRetained(diagnostic.Path, "#/components/examples/", closure.examples) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/securitySchemes/", closure.securitySchemes) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/schemas/", closure.schemas) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/parameters/", closure.parameters) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/requestBodies/", closure.requestBodies) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/responses/", closure.responses) &&
			componentDiagnosticRetained(diagnostic.Path, "#/components/headers/", closure.headers) &&
			componentDiagnosticRetained(diagnostic.Path, "#/tags/", closure.tags) {
			filtered = append(filtered, diagnostic)
		}
	}
	if closure.namingError != nil {
		if fatal, _ := filtered.Classify(true); len(fatal) == 0 {
			filtered = append(filtered, designNameDiagnostics(closure.namingError)...)
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
	pointer := jsontext.Pointer("/" + segment)
	if !pointer.IsValid() {
		// An invalid path cannot establish unselected ownership. Keep the
		// diagnostic instead of silently dropping an unresolved blocker.
		return true
	}
	_, ok := retained[pointer.LastToken()]
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
