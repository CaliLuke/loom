package ir

import (
	"slices"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type (
	schemaSlotLocation int
	schemaSlot         struct {
		location   schemaSlotLocation
		index      int
		name       string
		member     string
		branch     string
		element    bool
		additional bool
		// base is the declaration a composition contributor references.
		base expr.UserType
		// contributor is a composition contributor projected into the wire
		// context of an HTTP body occurrence; it projects with its own plan.
		contributor *projectedContributor
	}
	schemaConstruction struct {
		kind  expr.Kind
		slots []schemaSlot
	}
	schemaProjectionKey struct {
		baseline   *Schema
		occurrence expr.ValuePlanNode
		projection string
	}
	// schemaPosition pairs a baseline schema with the value-plan node it is
	// projected at; the node carries the occurrence codec.
	schemaPosition struct {
		schema *Schema
		plan   expr.ValuePlanNode
	}
)

const (
	schemaProperty schemaSlotLocation = iota
	schemaItems
	schemaAdditional
	schemaOneOf
	schemaAllOf
	schemaAnyOf
)

// recordStructuralSchema captures only edges emitted by the owning analyzer.
// Projection consumes these records instead of guessing structural alternatives.
func (a *Analyzer) recordStructuralSchema(schema *Schema, attribute *expr.AttributeExpr) {
	construction := schemaConstruction{kind: attribute.Type.Kind()}
	switch actual := attribute.Type.(type) {
	case *expr.Object:
		for _, field := range *actual {
			wire := expr.JSONFieldName(expr.ElementName(field.Name), field.Attribute)
			if schema.Properties[wire] != nil {
				construction.slots = append(construction.slots, schemaSlot{location: schemaProperty, name: wire, member: field.Name})
			}
		}
		if attribute.AdditionalAttributes != nil && schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
			construction.slots = append(construction.slots, schemaSlot{location: schemaAdditional, additional: true})
		}
	case *expr.Array:
		if schema.Items != nil {
			construction.slots = append(construction.slots, schemaSlot{location: schemaItems, element: true})
		}
	case *expr.Map:
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
			construction.slots = append(construction.slots, schemaSlot{location: schemaAdditional, element: true})
		}
	case *expr.Union:
		for index, branch := range sortedUnionValues(actual) {
			construction.slots = append(construction.slots, schemaSlot{location: schemaOneOf, index: index, branch: expr.UnionVariantTag(branch)})
		}
	case expr.Primitive:
		for index := range schema.AnyOf {
			construction.slots = append(construction.slots, schemaSlot{location: schemaAnyOf, index: index})
		}
	}
	setAcquisitionEntry(a, a.constructions, schema, construction)
}

func (a *Analyzer) applyNullableSchema(schema *Schema) {
	construction, recorded := a.constructions[schema]
	applyNullableSchema(schema)
	if !asyncNullableWrapper(schema) {
		return
	}
	if recorded {
		setAcquisitionEntry(a, a.constructions, schema.AnyOf[0], construction)
	}
	setAcquisitionEntry(a, a.constructions, schema, schemaConstruction{slots: []schemaSlot{{location: schemaAnyOf}}})
}

// constructionContext identifies the JSON representations that the composition
// contributors recorded below baseline refer to. The construction owns the
// contributors a projected composition materializes; member projections
// receive value-plan attribute copies, which do not carry them. The walk pairs
// every recorded slot with the plan node planProjectionChildren selects,
// follows local component references as the projection planner does, and
// counts a contributor only where its composition is projected with the JSON
// codec. A projection decision therefore sees every inline composition its
// materialization reaches, at any depth and through any intermediate.
func (a *Analyzer) constructionContext(baseline *Schema, plan expr.ValuePlanNode) string {
	root := schemaPosition{baseline, plan}
	if context, found := a.constructionContexts[root]; found {
		return context
	}
	var closures []string
	visited := make(map[schemaPosition]bool)
	var visit func(schemaPosition)
	visit = func(position schemaPosition) {
		if position.schema == nil || !position.plan.Valid() || visited[position] {
			return
		}
		visited[position] = true
		if name, local := schemaComponentName(position.schema.Ref); local {
			visit(schemaPosition{a.schemas[name], position.plan})
		}
		construction := a.constructions[position.schema]
		structural := construction.structuralPlan(position.plan)
		for _, slot := range construction.slots {
			if slot.contributor != nil {
				visit(schemaPosition{slot.get(position.schema), slot.contributor.plan})
				continue
			}
			if slot.base == nil {
				visit(schemaPosition{slot.get(position.schema), slot.plan(structural)})
				continue
			}
			if position.plan.Codec() != expr.ValueCodecJSON {
				continue
			}
			if closure := a.declarationClosure(slot.base); closure != "" {
				closures = append(closures, closure)
			}
		}
	}
	visit(root)
	slices.Sort(closures)
	context := strings.Join(slices.Compact(closures), "|")
	setAcquisitionEntry(a, a.constructionContexts, root, context)
	return context
}

// structuralPlan returns the plan node a construction's slots pair with: a
// typed construction descends the plan's named-declaration edges.
func (c schemaConstruction) structuralPlan(plan expr.ValuePlanNode) expr.ValuePlanNode {
	structural := plan
	if c.kind != 0 {
		for structural.Underlying().Valid() {
			structural = structural.Underlying()
		}
	}
	return structural
}

// positional reports whether the slot's child stays at its parent's value
// position, as wrappers and composition members do.
func (s schemaSlot) positional() bool {
	return s.member == "" && s.branch == "" && !s.element && !s.additional
}

func (s schemaSlot) plan(parent expr.ValuePlanNode) expr.ValuePlanNode {
	if s.additional {
		return parent.Additional()
	}
	if s.element {
		return parent.Element()
	}
	if s.member != "" {
		for _, member := range parent.Members() {
			if member.Name == s.member {
				return member.Node
			}
		}
		return expr.ValuePlanNode{}
	}
	if s.branch != "" {
		for _, branch := range parent.Branches() {
			if branch.Tag == s.branch {
				return branch.Node
			}
		}
		return expr.ValuePlanNode{}
	}
	return parent
}

func (s schemaSlot) get(schema *Schema) *Schema {
	switch s.location {
	case schemaProperty:
		return schema.Properties[s.name]
	case schemaItems:
		return schema.Items
	case schemaAdditional:
		return schema.AdditionalProperties.Schema
	case schemaOneOf:
		return schema.OneOf[s.index]
	case schemaAllOf:
		return schema.AllOf[s.index]
	case schemaAnyOf:
		return schema.AnyOf[s.index]
	default:
		panic("unknown baseline schema slot")
	}
}

func (s schemaSlot) set(schema *Schema, child *Schema) {
	switch s.location {
	case schemaProperty:
		properties := make(map[string]*Schema, len(schema.Properties))
		for name, value := range schema.Properties {
			properties[name] = value
		}
		properties[s.name] = child
		schema.Properties = properties
	case schemaItems:
		schema.Items = child
	case schemaAdditional:
		additional := *schema.AdditionalProperties
		additional.Schema = child
		schema.AdditionalProperties = &additional
	case schemaOneOf:
		schema.OneOf = append([]*Schema(nil), schema.OneOf...)
		schema.OneOf[s.index] = child
	case schemaAllOf:
		schema.AllOf = append([]*Schema(nil), schema.AllOf...)
		schema.AllOf[s.index] = child
	case schemaAnyOf:
		schema.AnyOf = append([]*Schema(nil), schema.AnyOf...)
		schema.AnyOf[s.index] = child
	}
}

// copySchemaConstruction carries the existing owning edge correspondence through
// reference rewriting. It never infers a structural child from schema contents.
func (a *Analyzer) copySchemaConstruction(source, target *Schema) {
	if construction, found := a.constructions[source]; found {
		setAcquisitionEntry(a, a.constructions, target, construction)
	}
}
