package ir

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// representationContext pairs a named declaration with the JSON schema plan
// of its own occurrence, independent of any referencing occurrence.
type representationContext struct {
	attribute *expr.AttributeExpr
	plan      expr.ValuePlanNode
}

// declarationRepresentation returns the JSON representation context of typ.
// Composition contributors and discriminator references denote the referenced
// declaration itself, so they are projected in this context rather than in the
// plan of the occurrence that refers to them.
func (a *Analyzer) declarationRepresentation(typ expr.UserType) (*expr.AttributeExpr, expr.ValuePlanNode) {
	if representation, found := a.declarationRepresentations[typ.ID()]; found {
		return representation.attribute, representation.plan
	}
	representation := representationContext{attribute: &expr.AttributeExpr{Type: typ}}
	plan, err := declarationSchemaPlan(representation.attribute)
	if err != nil && a.err == nil {
		a.err = fmt.Errorf("OpenAPI representation of %q: %w", typ.Name(), err)
	}
	representation.plan = plan
	a.declarationRepresentations[typ.ID()] = representation
	return representation.attribute, representation.plan
}

// declarationSchemaPlan builds the same JSON schema plan as a standalone
// documentation occurrence of attribute.
func declarationSchemaPlan(attribute *expr.AttributeExpr) (expr.ValuePlanNode, error) {
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		return expr.ValuePlanNode{}, err
	}
	plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{
		TargetOccurrence: occurrence, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema,
	})
	if err != nil {
		return expr.ValuePlanNode{}, err
	}
	return plan.Root(), nil
}

// representationIdentity identifies the representation an occurrence needs:
// its byte projection plus, for each attribute it reaches in a JSON codec
// context, the JSON publication context of that attribute. The occurrence
// codec context is decided here, before the allocation quotient merges equal
// bodies.
func (a *Analyzer) representationIdentity(attribute *expr.AttributeExpr, plan expr.ValuePlanNode) string {
	return projectionIdentity(attribute, plan, a.occurrenceContext)
}

// occurrenceContext identifies the JSON representations an occurrence
// attribute refers to outside its own physical graph: the closures of the
// publication neighbors of the whole declaration graph the node denotes, and of
// the attribute's own publication neighbors. The declaration graph is read from
// the declaration itself because the attribute below a member can be a
// value-plan copy that no longer carries the composition contributors of
// nested inline attributes.
func (a *Analyzer) occurrenceContext(attribute *expr.AttributeExpr, node expr.ValuePlanNode) string {
	neighbors := a.attributePublicationNeighbors(attribute, nil)
	if declaration := a.occurrenceDeclaration(attribute, node); declaration != nil {
		neighbors = append(neighbors, a.declarationNeighbors(declaration)...)
	}
	var closures []string
	for _, neighbor := range neighbors {
		if closure := a.declarationClosure(neighbor); closure != "" {
			closures = append(closures, closure)
		}
	}
	slices.Sort(closures)
	return strings.Join(slices.Compact(closures), "|")
}

// occurrenceDeclaration returns the declaration an occurrence node denotes.
// The value plan owns that identity: TargetDeclarationID survives HTTP body
// copies, renamed nested copies and flattened unions, whose physical types are
// not declarations. A node without a plan target falls back to the
// attribute's own named type when that type is a declaration.
func (a *Analyzer) occurrenceDeclaration(attribute *expr.AttributeExpr, node expr.ValuePlanNode) expr.UserType {
	if declaration := a.declarationTypes[node.TargetDeclarationID()]; declaration != nil {
		return declaration
	}
	return a.namedDeclaration(attribute)
}

// namedDeclaration returns the declaration that attribute's own named type
// is, if any.
func (a *Analyzer) namedDeclaration(attribute *expr.AttributeExpr) expr.UserType {
	if named, ok := attribute.Type.(expr.UserType); ok {
		return a.declarationTypes[named.ID()]
	}
	return nil
}

// attributePublicationNeighbors is the single publication-neighbor contract
// shared by occurrence and declaration walks. An attribute refers outside its
// physical graph to the publication dependencies and composition contributors
// of the declaration it denotes, and to its own composition contributors,
// including those of inline attributes that no declaration owns.
func (a *Analyzer) attributePublicationNeighbors(attribute *expr.AttributeExpr, declaration expr.UserType) []expr.UserType {
	var neighbors []expr.UserType
	if declaration != nil {
		neighbors = append(neighbors, a.publicationDependencies(declaration)...)
		neighbors = append(neighbors, compositionContributors(declaration.Attribute())...)
	}
	return append(neighbors, compositionContributors(attribute)...)
}

// compositionContributors returns the named declarations that the schema of
// attribute references through allOf.
func compositionContributors(attribute *expr.AttributeExpr) []expr.UserType {
	var contributors []expr.UserType
	for _, base := range attribute.CompositionReferenceBases() {
		if declaration, ok := base.(expr.UserType); ok {
			contributors = append(contributors, declaration)
		}
	}
	return contributors
}

// declarationClosure identifies the JSON representations that a JSON
// representation of typ depends on: typ itself and, transitively, the
// publication neighbors of every attribute in the declarations reached. A
// declaration whose own body has no byte fields still needs a JSON
// representation when this closure is not empty, because the members it
// publishes and the parent it extends differ from its raw registration.
func (a *Analyzer) declarationClosure(typ expr.UserType) string {
	if closure, found := a.publicationContexts[typ.ID()]; found {
		return closure
	}
	visited := make(map[string]bool)
	var parts []string
	queue := []expr.UserType{typ}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if visited[next.ID()] {
			continue
		}
		visited[next.ID()] = true
		attribute, plan := a.declarationRepresentation(next)
		if identity := byteProjectionIdentity(attribute, plan); identity != "" {
			parts = append(parts, next.ID()+"="+identity)
		}
		queue = append(queue, a.declarationNeighbors(next)...)
	}
	slices.Sort(parts)
	closure := strings.Join(parts, ";")
	setAcquisitionEntry(a, a.publicationContexts, typ.ID(), closure)
	return closure
}

// declarationNeighbors applies attributePublicationNeighbors to every
// attribute of typ's declaration graph: fields, array elements, map keys and
// values, union branches, additional attributes and nested declarations.
// Recursive declarations are visited once. The result is memoized per
// declaration and returned clipped, so callers that append never share it.
func (a *Analyzer) declarationNeighbors(typ expr.UserType) []expr.UserType {
	if neighbors, found := a.declarationNeighborCache[typ.ID()]; found {
		return slices.Clip(neighbors)
	}
	var neighbors []expr.UserType
	seen := make(map[string]bool)
	var walk func(*expr.AttributeExpr)
	walk = func(attribute *expr.AttributeExpr) {
		if attribute == nil {
			return
		}
		neighbors = append(neighbors, a.attributePublicationNeighbors(attribute, a.namedDeclaration(attribute))...)
		switch actual := attribute.Type.(type) {
		case expr.UserType:
			if seen[actual.ID()] {
				return
			}
			seen[actual.ID()] = true
			walk(actual.Attribute())
		case *expr.Array:
			walk(actual.ElemType)
		case *expr.Map:
			walk(actual.KeyType)
			walk(actual.ElemType)
		case *expr.Object:
			for _, field := range *actual {
				walk(field.Attribute)
			}
		case *expr.Union:
			for _, branch := range actual.Values {
				walk(branch.Attribute)
			}
		}
		walk(attribute.AdditionalAttributes)
	}
	walk(&expr.AttributeExpr{Type: typ})
	setAcquisitionEntry(a, a.declarationNeighborCache, typ.ID(), neighbors)
	return slices.Clip(neighbors)
}

// publicationDependencies returns the declarations published with the root
// component of typ: its implicit discriminator children and the discriminator
// mapping and default targets that its body does not otherwise reference. The
// result is memoized per declaration and returned clipped.
func (a *Analyzer) publicationDependencies(typ expr.UserType) []expr.UserType {
	if dependencies, found := a.publicationDependencyCache[typ.ID()]; found {
		return slices.Clip(dependencies)
	}
	name, ok := a.declarationComponentName(typ)
	if !ok {
		return nil
	}
	dependencies := slices.Clone(a.discriminatorFamilies[name])
	physical := make(map[string]bool)
	var targets []string
	mapSchemaReferences(a.publicationBaseline(a.schemas[name]), func(ref, path string) string {
		if target, local := schemaComponentName(ref); local {
			if strings.Contains(path, "/Discriminator/") {
				targets = append(targets, target)
			} else {
				physical[target] = true
			}
		}
		return ref
	})
	for _, target := range targets {
		if declaration := a.rootDeclaration(target); declaration != nil && !physical[target] {
			dependencies = append(dependencies, declaration)
		}
	}
	setAcquisitionEntry(a, a.publicationDependencyCache, typ.ID(), dependencies)
	return slices.Clip(dependencies)
}

// rootDeclaration returns the declaration whose root component is name.
func (a *Analyzer) rootDeclaration(name string) expr.UserType {
	typ := a.declarationTypes[a.schemaTypeIDs[name]]
	if typ == nil {
		return nil
	}
	if root, ok := a.declarationComponentName(typ); !ok || root != name {
		return nil
	}
	return typ
}
