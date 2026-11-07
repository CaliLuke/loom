package expr

// EffectiveConstraintFacts contains immutable predicate and default-declaration
// facts for a finalized occurrence, including a physical representation. It does
// not establish that a declared default is admitted by that representation.
// Semantic admission and materialization use EffectiveConstraintsFor or a
// ValueContext occurrence instead.
type EffectiveConstraintFacts struct {
	validation         EffectiveValidation
	defaultDeclaration *valueSourceSnapshot
}

// EffectiveConstraintFactsFor derives effective predicates and default-source
// precedence from attribute's current occurrence graph. It checks predicate
// domains and enum declarations, but does not resolve raw defaults against the
// current physical shape or adjudicate conflicting inherited default values.
// Authored designs must pass semantic admission before physical generation.
func EffectiveConstraintFactsFor(attribute *AttributeExpr) (EffectiveConstraintFacts, error) {
	context := NewValueContext()
	builder, err := context.captureOccurrence(attribute, false)
	if err != nil {
		return EffectiveConstraintFacts{}, err
	}
	if err := builder.effectiveValidation(context); err != nil {
		return EffectiveConstraintFacts{}, err
	}
	facts := EffectiveConstraintFacts{validation: builder.graph.root.constraints.validation}
	if layer := effectiveDefaultLayer(builder.graph.root); layer != nil {
		facts.defaultDeclaration = layer.defaultValue
	}
	return facts, nil
}

// Validation returns detached effective predicates, preserving physical required
// members and the occurrence's named and local constraint layers.
func (f EffectiveConstraintFacts) Validation() EffectiveValidation {
	return f.validation.clone()
}

// HasDefault reports whether the effective source precedence selects a default
// declaration. False, zero and nonnil empty values remain present declarations.
// Presence does not certify default admission or resolve inherited conflicts.
func (f EffectiveConstraintFacts) HasDefault() bool {
	return f.defaultDeclaration != nil
}

// DefaultDeclaration returns a detached selected raw declaration for consumers
// that compare declaration identity. Opaque host values remain borrowed without
// invoking codecs, as in semantic source capture. The result is not an admitted
// or materialized value; use a semantic ValueContext to produce such a value.
func (f EffectiveConstraintFacts) DefaultDeclaration() (any, bool) {
	if f.defaultDeclaration == nil {
		return nil, false
	}
	return copyValueRaw(f.defaultDeclaration.raw), true
}

func (b *valueOccurrenceBuilder) effectiveValidation(context *ValueContext) error {
	for _, node := range b.graph.nodes {
		node.constraints = buildEffectiveConstraints(context, b.graph, node)
	}
	for _, node := range b.graph.nodes {
		if err := validateEffectiveDomain(context, b.graph, node); err != nil {
			return err
		}
	}
	for _, node := range b.graph.nodes {
		if err := projectEffectiveEnumJSON(context, b.graph, node); err != nil {
			return err
		}
	}
	for _, node := range b.graph.nodes {
		if err := validateEffectiveEnums(context, b.graph, node); err != nil {
			return err
		}
	}
	return nil
}
