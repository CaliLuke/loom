package openapiimport

import (
	"github.com/CaliLuke/loom/internal/openapiref"
)

type (
	// exampleSource binds an admitted example to its authored owner: the
	// normalized schema it was authored on, or the media occurrence schema that
	// uses it (nil when the media has no schema). Constructors that attach a
	// reference, bases, properties or remove occurrence null complete that same
	// owner, while normalization may copy the example into another schema. path
	// is the authored or referencing location, used only for diagnostics.
	exampleSource struct {
		owner      *Schema
		path       string
		referenced bool
	}

	// exampleGraph decides example membership over the complete normalized
	// document. Reading admits decoding, literal and metadata facts only;
	// membership depends on references, bases and promoted constructors that
	// exist only once the whole graph is built.
	exampleGraph struct {
		schemas     map[string]*Schema
		visited     map[*Schema]bool
		diagnostics Diagnostics
	}

	// evaluatedMembers names the members of an object value that a matching
	// schema evaluated: its declared properties, members admitted by explicit
	// additionalProperties, and the members evaluated by its reference target,
	// bases and single matching oneOf branch.
	evaluatedMembers map[string]struct{}
)

// retainMemberExamples keeps each example only when its value is a member of
// both its authored owner and the final schema that renders it: the schema that
// carries it, or the schema of its media occurrence. It runs once after
// constructor promotion and before selection pruning, so every domain is
// resolved against the complete final component graph and rejections are
// reported at the example's own source path.
func (a *analyzer) retainMemberExamples(document *Document) {
	graph := &exampleGraph{
		schemas: make(map[string]*Schema, len(document.Components.Schemas)),
		visited: make(map[*Schema]bool),
	}
	for _, named := range document.Components.Schemas {
		graph.schemas[named.Name] = named.Schema
	}
	components := &document.Components
	for _, named := range components.Schemas {
		graph.schema(named.Schema)
	}
	for index := range components.Parameters {
		graph.schema(components.Parameters[index].Parameter.Schema)
	}
	for index := range components.RequestBodies {
		graph.requestBody(&components.RequestBodies[index].RequestBody)
	}
	for index := range components.Responses {
		graph.response(&components.Responses[index].Response)
	}
	for index := range components.Headers {
		graph.schema(components.Headers[index].Header.Schema)
	}
	for index := range document.Operations {
		operation := &document.Operations[index]
		for parameter := range operation.Parameters {
			graph.schema(operation.Parameters[parameter].Schema)
		}
		if operation.RequestBody != nil {
			graph.requestBody(operation.RequestBody)
		}
		for response := range operation.Responses {
			graph.response(&operation.Responses[response].Response)
		}
	}
	a.diagnostics = append(a.diagnostics, graph.diagnostics...)
}

func (g *exampleGraph) requestBody(body *RequestBody) {
	body.Examples = g.retain(body.Examples, body.Schema)
	g.schema(body.Schema)
}

func (g *exampleGraph) response(response *Response) {
	response.Examples = g.retain(response.Examples, response.Schema)
	g.schema(response.Schema)
	for index := range response.Headers {
		g.schema(response.Headers[index].Header.Schema)
	}
}

// schema visits each final schema owner once by identity.
func (g *exampleGraph) schema(schema *Schema) {
	if schema == nil || g.visited[schema] {
		return
	}
	g.visited[schema] = true
	schema.Examples = g.retain(schema.Examples, schema)
	for index := range schema.Properties {
		g.schema(schema.Properties[index].Schema)
	}
	g.schema(schema.Items)
	if schema.AdditionalProperties != nil {
		g.schema(schema.AdditionalProperties.Schema)
	}
	for _, branch := range schema.OneOf {
		g.schema(branch)
	}
	for _, base := range schema.Bases {
		g.schema(base)
	}
}

// retain returns the examples that belong to their authored owner and to the
// rendering schema, without mutating a slice that copied schemas may share.
func (g *exampleGraph) retain(examples []Example, rendering *Schema) []Example {
	var retained []Example
	for index, example := range examples {
		owned, _ := g.member(example.source.owner, example.Value, make(map[string]struct{}))
		rendered, _ := g.member(rendering, example.Value, make(map[string]struct{}))
		if owned && rendered {
			if retained != nil {
				retained = append(retained, example)
			}
			continue
		}
		if retained == nil {
			retained = append(make([]Example, 0, len(examples)), examples[:index]...)
		}
		message := "example value is not compatible with its schema"
		switch {
		case example.source.referenced:
			message = "referenced example value is not compatible with its schema"
		case example.Value == nil:
			message = "null example requires a nullable schema"
		}
		g.diagnostics.add("examples", example.source.path, message)
	}
	if retained == nil {
		return examples
	}
	if len(retained) == 0 {
		return nil
	}
	return retained
}

// member reports whether value belongs to the normalized value domain of
// schema: its local assertions, its reference target, every retained base and
// exactly one oneOf branch. For an object value it also returns the members
// those contributors evaluated, so a closed union admits only members that its
// matching branch evaluated. active holds the component references entered at
// this value position; a child value starts a new position and evaluation.
func (g *exampleGraph) member(schema *Schema, value any, active map[string]struct{}) (bool, evaluatedMembers) {
	if schema == nil {
		return value != nil, nil
	}
	// Classification owns the unconstrained domain, which includes null. A
	// missing type alone is not that domain. It evaluates no member.
	if schema.Unconstrained {
		return true, nil
	}
	if value == nil {
		return g.nullMember(schema, active), nil
	}
	evaluated, ok := g.localMember(schema, value)
	if !ok {
		return false, nil
	}
	if schema.Ref != "" {
		matched, members := g.reference(schema.Ref, value, active)
		if !matched {
			return false, nil
		}
		evaluated = evaluated.with(members)
	}
	for _, base := range schema.Bases {
		matched, members := g.member(base, value, active)
		if !matched {
			return false, nil
		}
		evaluated = evaluated.with(members)
	}
	if len(schema.OneOf) > 0 {
		matched, members := g.exactlyOne(schema.OneOf, value, active)
		if !matched {
			return false, nil
		}
		evaluated = evaluated.with(members)
	}
	// UnionClosed retains unevaluatedProperties: false at this occurrence.
	if schema.UnionClosed && !evaluated.covers(value) {
		return false, nil
	}
	return true, evaluated
}

// nullMember applies null ownership. Nullable is the explicit null
// alternative, including an outer anyOf of a reference and null, so it is not
// conjoined with a nonnullable target. A normalized enum never lists null, and
// a local type admits null only through Nullable. Otherwise every contributor
// must admit null, a union through exactly one branch.
func (g *exampleGraph) nullMember(schema *Schema, active map[string]struct{}) bool {
	if schema.Nullable {
		return true
	}
	if len(schema.Enum) > 0 || !composedSchema(schema) {
		return false
	}
	if schema.Ref != "" {
		if matched, _ := g.reference(schema.Ref, nil, active); !matched {
			return false
		}
	}
	for _, base := range schema.Bases {
		if matched, _ := g.member(base, nil, active); !matched {
			return false
		}
	}
	if len(schema.OneOf) == 0 {
		return true
	}
	matched, _ := g.exactlyOne(schema.OneOf, nil, active)
	return matched
}

// localMember checks the assertions authored at this schema for a non-null
// value and returns the object members it evaluated. Absent keywords assert
// nothing; a referenced target owns its own.
func (g *exampleGraph) localMember(schema *Schema, value any) (evaluatedMembers, bool) {
	if !normalizedExampleValidationMatches(schema, value) {
		return nil, false
	}
	switch schema.Type {
	case "":
		// A bare typeless schema is not the unconstrained domain; only a
		// reference, base or union contributes the type.
		return nil, composedSchema(schema)
	case "string":
		_, ok := value.(string)
		return nil, ok
	case "boolean":
		_, ok := value.(bool)
		return nil, ok
	case "integer":
		return nil, isExampleInteger(value)
	case "number":
		return nil, isExampleInteger(value) || isExampleFloat(value)
	case "array":
		return nil, g.arrayMember(schema, value)
	case "object":
		return g.objectMember(schema, value)
	default:
		return nil, false
	}
}

func (g *exampleGraph) arrayMember(schema *Schema, value any) bool {
	values, ok := value.([]any)
	if !ok {
		return false
	}
	count := int64(len(values))
	if schema.MinItems != nil && count < *schema.MinItems || schema.MaxItems != nil && count > *schema.MaxItems {
		return false
	}
	if schema.Items == nil {
		return true
	}
	for _, item := range values {
		if matched, _ := g.member(schema.Items, item, make(map[string]struct{})); !matched {
			return false
		}
	}
	return true
}

// objectMember evaluates declared properties and members admitted by explicit
// additionalProperties. Without that keyword an undeclared member is admitted
// but not evaluated.
func (g *exampleGraph) objectMember(schema *Schema, value any) (evaluatedMembers, bool) {
	values, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	for _, name := range schema.Required {
		if _, ok := values[name]; !ok {
			return nil, false
		}
	}
	properties := make(map[string]*Schema, len(schema.Properties))
	for _, property := range schema.Properties {
		properties[property.Name] = property.Schema
	}
	var evaluated evaluatedMembers
	for name, item := range values {
		member, declared := properties[name]
		if !declared {
			additional := schema.AdditionalProperties
			if additional == nil {
				continue
			}
			if additional.Allowed != nil && !*additional.Allowed {
				return nil, false
			}
			if additional.Schema == nil {
				evaluated = evaluated.with(evaluatedMembers{name: {}})
				continue
			}
			member = additional.Schema
		}
		if matched, _ := g.member(member, item, make(map[string]struct{})); !matched {
			return nil, false
		}
		evaluated = evaluated.with(evaluatedMembers{name: {}})
	}
	return evaluated, true
}

// reference resolves a component in the final graph. An unresolved target or a
// reference already entered at this value position is never membership.
func (g *exampleGraph) reference(ref string, value any, active map[string]struct{}) (bool, evaluatedMembers) {
	name, err := openapiref.Name(ref, "#/components/schemas/")
	if err != nil {
		return false, nil
	}
	target := g.schemas[name]
	if _, cycle := active[name]; cycle || target == nil {
		return false, nil
	}
	active[name] = struct{}{}
	defer delete(active, name)
	return g.member(target, value, active)
}

// exactlyOne applies oneOf to every value, including null, and returns the
// members evaluated by the single matching branch; unmatched branches
// evaluate nothing.
func (g *exampleGraph) exactlyOne(branches []*Schema, value any, active map[string]struct{}) (bool, evaluatedMembers) {
	matches := 0
	var evaluated evaluatedMembers
	for _, branch := range branches {
		if matched, members := g.member(branch, value, active); matched {
			matches++
			evaluated = members
		}
	}
	if matches != 1 {
		return false, nil
	}
	return true, evaluated
}

// composedSchema reports whether a reference, base or union contributes to the
// schema's domain beyond its local assertions.
func composedSchema(schema *Schema) bool {
	return schema.Ref != "" || len(schema.Bases) > 0 || len(schema.OneOf) > 0
}

// with adds the members of other to e, allocating e when it is empty.
func (e evaluatedMembers) with(other evaluatedMembers) evaluatedMembers {
	if len(other) == 0 {
		return e
	}
	if e == nil {
		e = make(evaluatedMembers, len(other))
	}
	for name := range other {
		e[name] = struct{}{}
	}
	return e
}

// covers reports whether every member of an object value was evaluated. Other
// values have no members to evaluate.
func (e evaluatedMembers) covers(value any) bool {
	values, ok := value.(map[string]any)
	if !ok {
		return true
	}
	for name := range values {
		if _, evaluated := e[name]; !evaluated {
			return false
		}
	}
	return true
}
