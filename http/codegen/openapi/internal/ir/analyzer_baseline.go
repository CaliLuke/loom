package ir

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
)

type (
	schemaSlotLocation int
	schemaSlot         struct {
		location schemaSlotLocation
		index    int
		name     string
		member   string
		branch   string
		element  bool
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
	a.constructions[schema] = construction
}

func (a *Analyzer) applyNullableSchema(schema *Schema) {
	construction, recorded := a.constructions[schema]
	applyNullableSchema(schema)
	if !asyncNullableWrapper(schema) {
		return
	}
	if recorded {
		a.constructions[schema.AnyOf[0]] = construction
	}
	a.constructions[schema] = schemaConstruction{slots: []schemaSlot{{location: schemaAnyOf}}}
}

// projectBaseline changes only byte-owned constraints. The baseline graph owns
// every emitted edge and annotation, including absence and memo-elided aliases.
// No source selection, sampling or schema reconstruction occurs in this walk.
func (a *Analyzer) projectBaseline(baseline *Schema, attribute *expr.AttributeExpr, plan expr.ValuePlanNode) *Schema {
	if baseline == nil {
		return nil
	}
	projection := byteProjectionIdentity(attribute, plan)
	if projection == "" {
		return baseline
	}
	key := schemaProjectionKey{baseline, plan, projection}
	if prior := a.projectedSchemas[key]; prior != nil {
		return prior
	}
	result := *baseline
	a.projectedSchemas[key] = &result
	a.copySchemaConstruction(baseline, &result)
	if name, local := schemaComponentName(baseline.Ref); local {
		result.Ref = toRef(a.projectBaselineComponent(name, attribute, plan, projection))
	}
	layers, bytes := byteOccurrenceLayers(attribute, plan)
	if bytes {
		result.MinLength, result.MaxLength = nil, nil
		if result.Format == "binary" {
			result.Format = ""
		}
	}
	construction, recorded := a.constructions[baseline]
	if !recorded {
		if baseline.Ref == "" {
			panic("OpenAPI projected baseline has no construction record")
		}
		return &result
	}
	if bytes && construction.kind == expr.BytesKind && len(construction.slots) == 0 {
		result.ContentEncoding = "base64"
		applyByteGate(&result, byteConstraintSchema(effectiveByteConstraint(attribute, layers, "$")))
	}
	structural := plan
	if construction.kind != 0 {
		for structural.Underlying().Valid() {
			structural = structural.Underlying()
		}
	}
	for _, slot := range construction.slots {
		childPlan := slot.plan(structural)
		childAttribute := attribute
		if slot.member != "" || slot.branch != "" || slot.element {
			if !childPlan.Valid() {
				panic("OpenAPI baseline construction has no paired target child")
			}
			childAttribute = childPlan.Attribute()
		}
		child := a.projectBaseline(slot.get(baseline), childAttribute, childPlan)
		slot.set(&result, child)
		if slot.branch != "" && result.Discriminator != nil {
			discriminator := *result.Discriminator
			discriminator.Mapping = copySchemaMapping(result.Discriminator.Mapping)
			discriminator.Mapping[slot.branch] = child.Ref
			result.Discriminator = &discriminator
		}
	}
	return &result
}

func (a *Analyzer) projectBaselineComponent(original string, attribute *expr.AttributeExpr, plan expr.ValuePlanNode, projection string) string {
	baseline := a.schemas[original]
	if baseline == nil {
		panic(fmt.Sprintf("OpenAPI baseline component %q is missing", original))
	}
	key := schemaProjectionKey{baseline, plan, projection}
	if name := a.projectedComponents[key]; name != "" {
		return name
	}
	fingerprint := fingerprintString(original + ":" + projection)
	name := a.Uniquify(original, fingerprint)
	a.projectedComponents[key] = name
	placeholder := &Schema{}
	a.schemas[name] = placeholder
	a.schemaFingerprints[name] = fingerprint
	a.schemaTypeIDs[name] = a.schemaTypeIDs[original]
	naming := a.schemaNames[original]
	if naming.desired == "" {
		naming = representationComponentName{desired: original, logical: original}
	}
	naming.json = true
	a.schemaNames[name] = naming
	*placeholder = *a.projectBaseline(baseline, attribute, plan)
	return name
}

func (s schemaSlot) plan(parent expr.ValuePlanNode) expr.ValuePlanNode {
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

func copySchemaMapping(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for name, value := range source {
		result[name] = value
	}
	return result
}

// copySchemaConstruction carries the existing owning edge correspondence through
// reference rewriting. It never infers a structural child from schema contents.
func (a *Analyzer) copySchemaConstruction(source, target *Schema) {
	if construction, found := a.constructions[source]; found {
		a.constructions[target] = construction
	}
}
