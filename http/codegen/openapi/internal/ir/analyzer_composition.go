package ir

import "github.com/CaliLuke/loom/expr"

// analyzeCompositionReferences retains the separately authored base contract
// when an intersection asks for reference identity. The physical object still
// uses the composed fields and constraints; neither layer replaces the other.
func (a *Analyzer) analyzeCompositionReferences(schema *Schema, attribute *expr.AttributeExpr, context string) *Schema {
	if !metaBool(attribute.Meta, "openapi:allOf:reference") {
		return schema
	}
	bases := attribute.CompositionBases()
	if len(bases) == 0 {
		return schema
	}
	result := &Schema{}
	slots := make([]schemaSlot, 0, len(bases)+1)
	for _, base := range bases {
		contributor := a.analyzeSchemaPositions(&expr.AttributeExpr{Type: base}, context,
			expr.ValuePlanNode{}, expr.ValuePlanNode{})
		slots = append(slots, schemaSlot{location: schemaAllOf, index: len(result.AllOf)})
		result.AllOf = append(result.AllOf, contributor)
	}
	slots = append(slots, schemaSlot{location: schemaAllOf, index: len(result.AllOf)})
	result.AllOf = append(result.AllOf, schema)
	a.constructions[result] = schemaConstruction{slots: slots}
	return result
}
