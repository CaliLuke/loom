package ir

import "github.com/CaliLuke/loom/expr"

// analyzeCompositionReferences retains the separately authored base contract
// when an intersection asks for reference identity. The physical object still
// uses the composed fields and constraints; neither layer replaces the other.
//
// An HTTP body object carries the composition of the authored source its
// projection records, whatever members the body selects. Each contributor
// is its own declaration projected
// into the body's selection and wire context. Any other attribute owns its
// composition; a text, form or multipart occurrence is analyzed from its
// value-plan copy, which keeps the occurrence metadata but, by the capture
// contract, not the composition bases, so the original graph attribute paired
// with the same plan position owns them.
func (a *Analyzer) analyzeCompositionReferences(schema *Schema, attribute *expr.AttributeExpr, context string) *Schema {
	projection, source := a.bodyProjectionAt(attribute)
	var bases []expr.DataType
	switch {
	case projection != nil:
		bases = projection.Source.CompositionReferenceBases()
	case source != attribute && metaBool(attribute.Meta, "openapi:allOf:reference"):
		bases = source.CompositionReferenceBases()
	default:
		bases = attribute.CompositionReferenceBases()
	}
	if len(bases) == 0 {
		return schema
	}
	result := &Schema{}
	slots := make([]schemaSlot, 0, len(bases)+1)
	for _, base := range bases {
		declaration, _ := base.(expr.UserType)
		slot := schemaSlot{location: schemaAllOf, index: len(result.AllOf)}
		if projection != nil && declaration != nil {
			wire := a.bodyWireContext(source)
			slot.contributor = a.contributor(wire, projection, declaration, context)
			slots = append(slots, slot)
			result.AllOf = append(result.AllOf, slot.contributor.schema)
			continue
		}
		slot.base = declaration
		slots = append(slots, slot)
		result.AllOf = append(result.AllOf, a.analyzeSchemaPositions(&expr.AttributeExpr{Type: base}, context,
			expr.ValuePlanNode{}, expr.ValuePlanNode{}))
	}
	slots = append(slots, schemaSlot{location: schemaAllOf, index: len(result.AllOf)})
	result.AllOf = append(result.AllOf, schema)
	a.constructions[result] = schemaConstruction{slots: slots}
	return result
}
