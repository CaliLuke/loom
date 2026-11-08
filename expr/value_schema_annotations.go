package expr

import (
	"fmt"
)

type (
	// DefaultValueOwner is a comparable framework-local identity of an authored
	// default. Controlled copies retain it; a new Default assignment does not.
	// It is not a public name, content hash, or serializable value.
	DefaultValueOwner struct {
		attribute  *AttributeExpr
		assignment *valueDefaultAssignment
	}

	// JSONSchemaAnnotations retains checked defaults and enum clauses for one
	// captured documentation occurrence. Values preserve declared scalar precision
	// and selected union branches. Enum clauses remain conjunctive predicates;
	// their members need not be admitted by a subsequent refinement.
	JSONSchemaAnnotations struct {
		// Default is the checked default, when HasDefault is true.
		Default JSONSchemaValue
		// HasDefault distinguishes an absent annotation from an explicit null.
		HasDefault bool
		// Enums retains all clauses in target declaration order.
		Enums [][]JSONSchemaValue
	}

	// JSONSchemaValue retains both the exact checked projection and its declared
	// value. ProjectionUnsupported denotes an explicit opaque codec boundary only;
	// consumers must not treat that boundary as a builtin membership guarantee.
	JSONSchemaValue struct {
		owner       DefaultValueOwner
		declaration EffectiveConstraintSource
		declared    any
		projection  ProjectionResult
	}

	// Nonzero size gives each assignment a distinct comparable allocation.
	valueDefaultAssignment byte
)

// PrepareJSONSchemaAnnotations projects required schema values through a
// captured documentation plan. Defaults must satisfy the final target. Authored
// enum members must be representable at their declaring occurrence; every
// conjoined clause is then projected through the target's declared shape.
// Errors retain the authored role and declaration, and never authorize dropping
// a default or enum member. Neither occurrences nor plans are modified.
func (c *ValueContext) PrepareJSONSchemaAnnotations(occurrence ValueOccurrence, plan ValuePlan) (JSONSchemaAnnotations, error) {
	var annotations JSONSchemaAnnotations
	if c == nil || occurrence.context != c.identity || plan.context != c.identity ||
		plan.root == nil || !plan.root.documentary || plan.source.node != occurrence.node || len(plan.selection) != 0 {
		return annotations, fmt.Errorf("schema annotations require their captured documentation occurrence and plan")
	}
	if published := plan.root.publishedDefault; published != nil {
		result := resolveConstraintValue(c, occurrence.graph, occurrence.node, published.value, nil, false)
		result.role = ValueRoleDefault
		value, err := c.schemaAnnotationValue(result, plan)
		if err != nil {
			return annotations, fmt.Errorf("default declared by %q: %w", published.declaration.Declaration, err)
		}
		value.declaration = published.declaration
		value.owner = published.owner
		annotations.Default, annotations.HasDefault = value, true
	}
	if plan.root.validation == nil {
		return annotations, nil
	}
	shape := plan
	shape.root = enumRootPlan(plan.root, true)
	for clauseIndex, clause := range plan.root.validation.Enums() {
		values := make([]JSONSchemaValue, len(clause))
		declaration := enumClauseDeclaration(occurrence, clause)
		constantOccurrence, constantPlan := occurrence, shape
		if projection := plan.root.enumProjection(clauseIndex); projection != plan.root {
			constantOccurrence = declaration
			constantPlan.source = declaration
			constantPlan.root = enumRootPlan(projection, true)
		}
		for index, raw := range clause {
			if declaration.node != nil && declaration.node.enumValues != nil {
				if err := c.checkDeclaredEnumAnnotation(declaration, raw); err != nil {
					return annotations, fmt.Errorf("enum clause %d member %d declared by %q: %w", clauseIndex, index, effectiveLayerName(declaration.node), err)
				}
			}
			result := c.ResolveDeclaredShape(constantOccurrence, c.SupplyValue(ValueInput{Raw: raw, ExplicitNull: raw == nil, Origin: "schema enum"}))
			value, err := c.schemaAnnotationValue(result, constantPlan)
			if err != nil {
				return annotations, fmt.Errorf("enum clause %d member %d at %q: %w", clauseIndex, index, effectiveLayerName(occurrence.node), err)
			}
			if declaration.node != nil {
				value.declaration = constraintSource(c, declaration.graph, declaration.node)
			}
			values[index] = value
		}
		annotations.Enums = append(annotations.Enums, values)
	}
	return annotations, nil
}

// Declaration returns the captured source of the authored annotation. A zero
// source means a target-only derived clause has no authored source occurrence.
func (v JSONSchemaValue) Declaration() EffectiveConstraintSource {
	return v.declaration
}

// DefaultOwner returns the authored owner retained by checked default preparation.
// The zero identity denotes a value without an authored default owner.
func (v JSONSchemaValue) DefaultOwner() DefaultValueOwner {
	return v.owner
}

// Projection returns the immutable wire result or explicit codec boundary.
func (v JSONSchemaValue) Projection() ProjectionResult {
	return v.projection
}

// DeclaredValue returns an owned builtin value; opaque codec leaves remain borrowed.
func (v JSONSchemaValue) DeclaredValue() any {
	return copyValueRaw(v.declared)
}

func (c *ValueContext) schemaAnnotationValue(result ValueResult, plan ValuePlan) (JSONSchemaValue, error) {
	projection := c.ProjectJSON(result, plan)
	if projection.Err() != nil {
		return JSONSchemaValue{}, projection.Err()
	}
	value, retained := c.DeclaredJSONValue(result, plan)
	if projection.Outcome() == ProjectionUnsupported && retained {
		if _, err := checkKnownSchemaAnnotation(plan.root, result.value, result.role); err != nil {
			return JSONSchemaValue{}, err
		}
		return JSONSchemaValue{declared: value, projection: projection}, nil
	}
	if projection.Outcome() != ProjectionEmitted {
		return JSONSchemaValue{}, fmt.Errorf("required JSON schema value is unrepresentable: %v", projection.Diagnostics())
	}
	if !retained {
		return JSONSchemaValue{}, fmt.Errorf("required JSON schema value has no retained declared value")
	}
	return JSONSchemaValue{declared: value, projection: projection}, nil
}

func (c *ValueContext) checkDeclaredEnumAnnotation(occurrence ValueOccurrence, raw any) error {
	root, err := c.enumDeclarationPlan(occurrence)
	if err != nil {
		return err
	}
	plan := ValuePlan{context: c.identity, source: occurrence, root: enumRootPlan(root, false), projections: &valueProjectionMemo{}}
	result := c.ResolveDeclaredShape(occurrence, c.SupplyValue(ValueInput{Raw: raw, ExplicitNull: raw == nil, Origin: "declared schema enum"}))
	_, err = c.schemaAnnotationValue(result, plan)
	return err
}

// defaultValueOwner retains authored assignment identity through controlled
// copies. Direct finalized attributes without a DSL assignment use their
// existing copy ancestry; no raw value is compared or serialized.
func defaultValueOwner(attribute *AttributeExpr) DefaultValueOwner {
	if attribute.defaultOwner != (DefaultValueOwner{}) {
		return attribute.defaultOwner
	}
	return DefaultValueOwner{attribute: valueAttributeOrigin(attribute)}
}
