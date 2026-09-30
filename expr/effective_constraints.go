package expr

import (
	"fmt"
	"reflect"
	"slices"
)

type (
	// EffectiveConstraints is an immutable snapshot of the constraints that
	// apply to one finalized attribute occurrence and its named-type ancestry.
	EffectiveConstraints struct {
		validation    EffectiveValidation
		defaultValue  *valueSourceSnapshot
		defaultSource EffectiveConstraintSource
		enumValues    []valueSourceSnapshot
		enumSource    EffectiveConstraintSource
		required      []EffectiveRequired
		lowerBound    EffectiveNumericBound
		upperBound    EffectiveNumericBound
	}

	// EffectiveValidation is the complete immutable validation snapshot for one
	// finalized occurrence.
	EffectiveValidation struct {
		rules       ValidationExpr
		clauses     []EffectiveValidationClause
		enumJSON    [][]any
		enumSources []*valueOccurrenceNode
	}

	// EffectiveValidationClause is one conjoined string predicate and its
	// declaration provenance.
	EffectiveValidationClause struct {
		// Kind identifies whether Value is a pattern or format predicate.
		Kind EffectiveValidationClauseKind
		// Value is the authored pattern or format name.
		Value string
		// Provenance identifies the declaration that authored the predicate.
		Provenance EffectiveConstraintSource
	}

	// EffectiveValidationClauseKind identifies a conjoined string predicate.
	EffectiveValidationClauseKind uint8

	// EffectiveConstraintSource identifies the occurrence and named declaration
	// that authored an effective enum, default, or required constraint.
	EffectiveConstraintSource struct {
		// Occurrence identifies the finalized attribute occurrence that authored
		// the constraint.
		Occurrence ValueIdentity
		// Declaration names the authored declaration for diagnostics and audits.
		Declaration string
	}

	// EffectiveRequired identifies one required field after resolving all
	// authored names against the finalized object occurrence.
	EffectiveRequired struct {
		// Name is the finalized design field name.
		Name string
		// WireName is the finalized serialized field name.
		WireName string
		// Member identifies the finalized object member.
		Member ValueIdentity
		// Provenance identifies the declaration that made the member required.
		Provenance EffectiveConstraintSource
	}

	// EffectiveNumericBound is one lowered numeric endpoint and its authored
	// declaration provenance. Present distinguishes an absent endpoint from zero.
	EffectiveNumericBound struct {
		// Value is the lowered endpoint value.
		Value float64
		// Exclusive reports whether values equal to the endpoint are excluded.
		Exclusive bool
		// Present reports whether an endpoint was authored.
		Present bool
		// Provenance identifies the declaration that supplied the tight endpoint.
		Provenance EffectiveConstraintSource
	}

	effectiveNumericBound struct {
		value     float64
		exclusive bool
		set       bool
		source    EffectiveConstraintSource
	}
)

const (
	// EffectivePatternClause identifies a regular-expression pattern predicate.
	EffectivePatternClause EffectiveValidationClauseKind = iota + 1
	// EffectiveFormatClause identifies a named string-format predicate.
	EffectiveFormatClause
)

// EffectiveConstraintsFor builds the immutable effective constraints for an
// attribute through the same finalized occurrence lifecycle used by values.
func EffectiveConstraintsFor(attribute *AttributeExpr) (EffectiveConstraints, error) {
	occurrence, err := NewValueContext().NewOccurrence(attribute)
	if err != nil {
		return EffectiveConstraints{}, err
	}
	return occurrence.EffectiveConstraints(), nil
}

// EffectiveConstraints returns the immutable constraints for this occurrence.
func (o ValueOccurrence) EffectiveConstraints() EffectiveConstraints {
	if o.node == nil || o.node.constraints == nil {
		return EffectiveConstraints{}
	}
	return *o.node.constraints
}

// Validation returns the complete effective validation snapshot.
func (c EffectiveConstraints) Validation() EffectiveValidation {
	return c.validation.clone()
}

// Lowered returns a detached ValidationExpr that preserves every predicate.
// Authored Values, Pattern, and Format are cleared. Selected enum membership is
// stored in EnumClauses, and each string predicate kind is stored in
// derived-to-base order in PatternClauses or FormatClauses. This distinguishes
// a materialized inherited enum from a newly authored enum. Interleaving
// between string predicate kinds and declaration provenance remain available
// only through Clauses. Re-querying a lowered physical attribute derives new
// physical provenance. Built-in enum values are detached while opaque custom
// values remain borrowed.
func (v EffectiveValidation) Lowered() *ValidationExpr {
	result := cloneEffectiveValidation(&v.rules)
	enums := copyEnumClauseValues(result.Enums())
	result.Values = nil
	result.EnumClauses = enums
	result.Pattern = ""
	result.Format = ""
	result.PatternClauses = nil
	result.FormatClauses = nil
	for _, clause := range v.clauses {
		switch clause.Kind {
		case EffectivePatternClause:
			result.PatternClauses = append(result.PatternClauses, clause.Value)
		case EffectiveFormatClause:
			result.FormatClauses = append(result.FormatClauses, ValidationFormat(clause.Value))
		}
	}
	return result
}

// Clauses returns detached pattern and format predicates in derived-to-base
// order. Exact kind-and-value duplicates retain the first, most-derived
// declaration provenance.
func (v EffectiveValidation) Clauses() []EffectiveValidationClause {
	return slices.Clone(v.clauses)
}

func (v EffectiveValidation) clone() EffectiveValidation {
	return EffectiveValidation{
		rules:       *cloneEffectiveValidation(&v.rules),
		clauses:     slices.Clone(v.clauses),
		enumJSON:    copyEnumClauseValues(v.enumJSON),
		enumSources: slices.Clone(v.enumSources),
	}
}

// Default returns the effective default and whether one was authored. Built-in
// values are detached; opaque custom values remain borrowed.
func (c EffectiveConstraints) Default() (any, bool) {
	if c.defaultValue == nil {
		return nil, false
	}
	return copyValueRaw(c.defaultValue.raw), true
}

// DefaultSource returns the declaration provenance of the effective default.
func (c EffectiveConstraints) DefaultSource() EffectiveConstraintSource {
	return c.defaultSource
}

// EnumSource returns the declaration provenance of the effective enum.
func (c EffectiveConstraints) EnumSource() EffectiveConstraintSource {
	return c.enumSource
}

// EnumCandidates returns the detached finite candidates from the first
// effective enum predicate that satisfy the complete contract, preserving
// their authored order. Present distinguishes no enum predicate from a present
// empty domain. Validation retains every unfiltered enum clause. Built-in
// values are detached while opaque custom values remain borrowed.
func (c EffectiveConstraints) EnumCandidates() (values []any, present bool) {
	if c.enumValues == nil {
		return nil, false
	}
	return rawSnapshotValues(c.enumValues), true
}

// Required returns detached required-field identities in deterministic
// current-to-base declaration order.
func (c EffectiveConstraints) Required() []EffectiveRequired {
	return slices.Clone(c.required)
}

// LowerBound returns the effective lowered numeric minimum.
func (c EffectiveConstraints) LowerBound() EffectiveNumericBound {
	return c.lowerBound
}

// UpperBound returns the effective lowered numeric maximum.
func (c EffectiveConstraints) UpperBound() EffectiveNumericBound {
	return c.upperBound
}

func (b *valueOccurrenceBuilder) effectiveConstraints(context *ValueContext) error {
	for _, node := range b.graph.nodes {
		node.constraints = buildEffectiveConstraints(context, b.graph, node)
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
	for _, node := range b.graph.nodes {
		if err := selectEffectiveDefault(context, b.graph, node); err != nil {
			return err
		}
	}
	for _, node := range b.graph.nodes {
		selectEffectiveEnumCandidates(context, b.graph, node)
	}
	return nil
}

func selectEffectiveEnumCandidates(context *ValueContext, graph *valueOccurrenceGraph, node *valueOccurrenceNode) {
	if node.constraints == nil || node.constraints.enumValues == nil {
		return
	}
	selected := make([]valueSourceSnapshot, 0, len(node.constraints.enumValues))
	for _, candidate := range node.constraints.enumValues {
		result := resolveConstraintValue(context, graph, node, candidate, nil, false)
		if constraintValueAdmitted(result) {
			selected = append(selected, candidate)
		}
	}
	node.constraints.enumValues = selected
}

func buildEffectiveConstraints(context *ValueContext, graph *valueOccurrenceGraph, node *valueOccurrenceNode) *EffectiveConstraints {
	layers := effectiveConstraintLayers(node)
	constraints := &EffectiveConstraints{}
	mergeEffectiveValidation(context, graph, constraints, layers)
	constraints.required = effectiveRequiredFields(context, graph, node, layers)
	constraints.validation.rules.Required = make([]string, len(constraints.required))
	for index, required := range constraints.required {
		constraints.validation.rules.Required[index] = required.Name
	}

	var firstEnumClause []valueSourceSnapshot
	var firstEnumClauseSource *valueOccurrenceNode
	var enumClauseSources []*valueOccurrenceNode
	var enumValueSource *valueOccurrenceNode
	for _, layer := range layers {
		for _, clause := range layer.enumClauses {
			if firstEnumClauseSource == nil {
				firstEnumClause = clause
				firstEnumClauseSource = layer
			}
			constraints.validation.rules.EnumClauses = append(
				constraints.validation.rules.EnumClauses,
				rawSnapshotValues(clause),
			)
			enumClauseSources = append(enumClauseSources, layer)
		}
		if layer.enumValues == nil {
			continue
		}
		constraints.enumValues = cloneValueSnapshots(layer.enumValues)
		constraints.validation.rules.Values = rawSnapshotValues(constraints.enumValues)
		constraints.enumSource = constraintSource(context, graph, layer)
		enumValueSource = layer
		break
	}
	if constraints.validation.rules.Values == nil && firstEnumClauseSource != nil {
		constraints.enumValues = cloneValueSnapshots(firstEnumClause)
		constraints.enumSource = constraintSource(context, graph, firstEnumClauseSource)
	}
	var seenEnumClauses [][]any
	if values := constraints.validation.rules.Values; values != nil {
		seenEnumClauses = append(seenEnumClauses, values)
		constraints.validation.enumSources = append(constraints.validation.enumSources, enumValueSource)
	}
	for index, clause := range constraints.validation.rules.EnumClauses {
		duplicate := slices.ContainsFunc(seenEnumClauses, func(existing []any) bool {
			return reflect.DeepEqual(existing, clause)
		})
		if duplicate {
			continue
		}
		seenEnumClauses = append(seenEnumClauses, clause)
		constraints.validation.enumSources = append(constraints.validation.enumSources, enumClauseSources[index])
	}
	return constraints
}

func projectEffectiveEnumJSON(context *ValueContext, graph *valueOccurrenceGraph, node *valueOccurrenceNode) error {
	if node.constraints == nil {
		return nil
	}
	validation := &node.constraints.validation
	enums := validation.rules.Enums()
	if len(enums) != len(validation.enumSources) {
		return fmt.Errorf("%seffective enum projection has %d clauses and %d declaration sources",
			effectiveConstraintPrefix(node), len(enums), len(validation.enumSources))
	}
	validation.enumJSON = make([][]any, len(enums))
	for clauseIndex, values := range enums {
		source := validation.enumSources[clauseIndex]
		validation.enumJSON[clauseIndex] = make([]any, len(values))
		for valueIndex, raw := range values {
			snapshot := snapshotValueSource(raw)
			if snapshot.err != nil {
				return fmt.Errorf("%sinvalid authored enum member: %w", effectiveConstraintPrefix(node), snapshot.err)
			}
			result := resolveConstraintValue(context, graph, source, snapshot, nil, true)
			admitted := constraintValueAdmitted(result)
			projected, ok := result.DeclaredJSONValue()
			if !ok {
				result = resolveConstraintShape(context, graph, source, snapshot)
				admitted = admitted || constraintValueAdmitted(result)
				projected, ok = result.DeclaredJSONValue()
			}
			authored := clauseIndex == 0 && validation.rules.Values != nil
			if !ok && (admitted || authored) {
				projected, ok = copyValueRaw(raw), true
			}
			if !ok {
				return fmt.Errorf("%senum member %#v declared by %q violates its declared type",
					effectiveConstraintPrefix(node), raw, effectiveLayerName(source))
			}
			validation.enumJSON[clauseIndex][valueIndex] = projected
		}
	}
	return nil
}

func validateEffectiveEnums(context *ValueContext, graph *valueOccurrenceGraph, node *valueOccurrenceNode) error {
	layers := effectiveConstraintLayers(node)
	for index := len(layers) - 1; index >= 0; index-- {
		layer := layers[index]
		if len(layer.enumValues) == 0 {
			continue
		}
		for _, member := range layer.enumValues {
			if member.err != nil {
				return fmt.Errorf("%sinvalid authored enum member: %w", effectiveConstraintPrefix(node), member.err)
			}
			contractResult := resolveConstraintValue(context, graph, layer, member, nil, true)
			if !constraintValueAdmitted(contractResult) {
				return fmt.Errorf("%senum member %#v declared by %q violates the effective contract for %q",
					effectiveConstraintPrefix(node), member.raw, effectiveLayerName(layer), effectiveLayerName(layer))
			}
			ancestor := effectiveEnumAncestor(layer)
			if ancestor == nil {
				continue
			}
			result := resolveConstraintValue(context, graph, layer, member, layer, false)
			if constraintValueAdmitted(result) {
				continue
			}
			return fmt.Errorf("%senum member %#v is not admitted by ancestor %q",
				effectiveConstraintPrefix(node), member.raw, effectiveLayerName(ancestor))
		}
	}
	return nil
}

func selectEffectiveDefault(context *ValueContext, graph *valueOccurrenceGraph, node *valueOccurrenceNode) error {
	constraints := node.constraints
	for _, layer := range effectiveConstraintLayers(node) {
		if layer.defaultValue == nil {
			continue
		}
		if layer.defaultValue.err != nil {
			return fmt.Errorf("%sinvalid authored default: %w", effectiveConstraintPrefix(node), layer.defaultValue.err)
		}
		result := resolveConstraintValue(context, graph, node, *layer.defaultValue, nil, false)
		if !constraintValueAdmitted(result) {
			return fmt.Errorf("%sdefault value %#v declared by %q violates the effective contract for %q",
				effectiveConstraintPrefix(node), layer.defaultValue.raw, effectiveLayerName(layer), effectiveLayerName(node))
		}
		copy := *layer.defaultValue
		copy.raw = copyValueRaw(copy.raw)
		constraints.defaultValue = &copy
		constraints.defaultSource = constraintSource(context, graph, layer)
		return nil
	}
	return nil
}

func effectiveConstraintLayers(node *valueOccurrenceNode) []*valueOccurrenceNode {
	var layers []*valueOccurrenceNode
	seen := make(map[*valueOccurrenceNode]bool)
	for node != nil && !seen[node] {
		seen[node] = true
		layers = append(layers, node)
		node = node.declaration.alias
	}
	return layers
}

func mergeEffectiveValidation(context *ValueContext, graph *valueOccurrenceGraph, constraints *EffectiveConstraints, layers []*valueOccurrenceNode) {
	target := &constraints.validation.rules
	var lower, upper effectiveNumericBound
	for _, layer := range layers {
		validation := layer.attribute.Validation
		if validation == nil {
			continue
		}
		source := constraintSource(context, graph, layer)
		mergeEffectiveClauses(&constraints.validation, validation, source)
		lower = tighterEffectiveBound(lower, validation.Minimum, false, true, source)
		lower = tighterEffectiveBound(lower, validation.ExclusiveMinimum, true, true, source)
		upper = tighterEffectiveBound(upper, validation.Maximum, false, false, source)
		upper = tighterEffectiveBound(upper, validation.ExclusiveMaximum, true, false, source)
		target.MinLength = tighterBound(target.MinLength, validation.MinLength, true)
		target.MaxLength = tighterBound(target.MaxLength, validation.MaxLength, false)
	}
	applyEffectiveBound(target, lower, true)
	applyEffectiveBound(target, upper, false)
	constraints.lowerBound = publicEffectiveBound(lower)
	constraints.upperBound = publicEffectiveBound(upper)
}

func mergeEffectiveClauses(target *EffectiveValidation, validation *ValidationExpr, source EffectiveConstraintSource) {
	for _, pattern := range validation.Patterns() {
		target.addClause(EffectiveValidationClause{Kind: EffectivePatternClause, Value: pattern, Provenance: source})
	}
	for _, format := range validation.Formats() {
		target.addClause(EffectiveValidationClause{Kind: EffectiveFormatClause, Value: string(format), Provenance: source})
	}
}

func (v *EffectiveValidation) addClause(clause EffectiveValidationClause) {
	for _, existing := range v.clauses {
		if existing.Kind == clause.Kind && existing.Value == clause.Value {
			return
		}
	}
	v.clauses = append(v.clauses, clause)
	switch clause.Kind {
	case EffectivePatternClause:
		if v.rules.Pattern == "" {
			v.rules.Pattern = clause.Value
		} else {
			v.rules.PatternClauses = append(v.rules.PatternClauses, clause.Value)
		}
	case EffectiveFormatClause:
		format := ValidationFormat(clause.Value)
		if v.rules.Format == "" {
			v.rules.Format = format
		} else {
			v.rules.FormatClauses = append(v.rules.FormatClauses, format)
		}
	}
}

func tighterEffectiveBound(current effectiveNumericBound, candidate *float64, exclusive, minimum bool, source EffectiveConstraintSource) effectiveNumericBound {
	if candidate == nil {
		return current
	}
	if !current.set || (minimum && *candidate > current.value) || (!minimum && *candidate < current.value) ||
		(*candidate == current.value && exclusive && !current.exclusive) {
		return effectiveNumericBound{value: *candidate, exclusive: exclusive, set: true, source: source}
	}
	return current
}

func publicEffectiveBound(bound effectiveNumericBound) EffectiveNumericBound {
	return EffectiveNumericBound{
		Value: bound.value, Exclusive: bound.exclusive, Present: bound.set, Provenance: bound.source,
	}
}

func applyEffectiveBound(validation *ValidationExpr, bound effectiveNumericBound, minimum bool) {
	if !bound.set {
		return
	}
	value := bound.value
	if minimum {
		if bound.exclusive {
			validation.ExclusiveMinimum = &value
		} else {
			validation.Minimum = &value
		}
		return
	}
	if bound.exclusive {
		validation.ExclusiveMaximum = &value
	} else {
		validation.Maximum = &value
	}
}

func effectiveRequiredFields(context *ValueContext, graph *valueOccurrenceGraph, node *valueOccurrenceNode, layers []*valueOccurrenceNode) []EffectiveRequired {
	objectNode := node
	for objectNode != nil && objectNode.declaration.alias != nil {
		objectNode = objectNode.declaration.alias
	}
	if objectNode == nil || objectNode.declaration.kind != ObjectKind {
		return nil
	}
	seen := make(map[uint64]bool)
	var required []EffectiveRequired
	for _, layer := range layers {
		if layer.attribute.Validation == nil {
			continue
		}
		for _, authored := range layer.attribute.Validation.Required {
			for _, member := range objectNode.declaration.members {
				if !requiredNameMatches(member.name, authored) {
					continue
				}
				if !seen[member.id] {
					seen[member.id] = true
					required = append(required, EffectiveRequired{
						Name: member.name, WireName: member.wire,
						Member:     ValueIdentity{context: context.identity, graph: graph, index: member.id},
						Provenance: constraintSource(context, graph, layer),
					})
				}
				break
			}
		}
	}
	return required
}

func resolveConstraintValue(
	context *ValueContext,
	graph *valueOccurrenceGraph,
	node *valueOccurrenceNode,
	snapshot valueSourceSnapshot,
	skipEnum *valueOccurrenceNode,
	ignoreEnums bool,
) ValueResult {
	source := &valueSourceData{context: context.identity, snapshot: snapshot}
	var ignoredEnums map[*valueOccurrenceNode]bool
	if ignoreEnums || skipEnum != nil {
		ignoredEnums = make(map[*valueOccurrenceNode]bool)
		if skipEnum != nil {
			ignoredEnums[skipEnum] = true
		}
	}
	if ignoreEnums {
		for _, layer := range effectiveConstraintLayers(node) {
			ignoredEnums[layer] = true
		}
	}
	resolver := valueResolver{
		context:      context,
		occurrence:   ValueOccurrence{context: context.identity, graph: graph, node: node},
		source:       source,
		active:       make(map[valueSnapshotVisit]bool),
		admission:    true,
		ignoredEnums: ignoredEnums,
	}
	return resolver.resolve(node, snapshot.raw, true, node, nil)
}

func resolveConstraintShape(
	context *ValueContext,
	graph *valueOccurrenceGraph,
	node *valueOccurrenceNode,
	snapshot valueSourceSnapshot,
) ValueResult {
	source := ValueSource{data: &valueSourceData{context: context.identity, snapshot: snapshot}}
	occurrence := ValueOccurrence{context: context.identity, graph: graph, node: node}
	return context.ResolveDeclaredShape(occurrence, source)
}

func constraintValueAdmitted(result ValueResult) bool {
	return result.Outcome() == ValueResolved ||
		(result.Outcome() == ValueUnsupported && !result.checkableFailure && result.value.node != nil)
}

func constraintSource(context *ValueContext, graph *valueOccurrenceGraph, node *valueOccurrenceNode) EffectiveConstraintSource {
	return EffectiveConstraintSource{
		Occurrence:  ValueIdentity{context: context.identity, graph: graph, index: node.id},
		Declaration: effectiveLayerName(node),
	}
}

func effectiveLayerName(node *valueOccurrenceNode) string {
	if node == nil {
		return "attribute"
	}
	if node.ownerName != "" {
		return node.ownerName
	}
	if named, ok := node.declaration.typ.(UserType); ok {
		return named.Name()
	}
	if node.declarationID != "" {
		return node.declarationID
	}
	return node.attribute.Type.Name()
}

func effectiveEnumAncestor(node *valueOccurrenceNode) *valueOccurrenceNode {
	seen := make(map[*valueOccurrenceNode]bool)
	for node = node.declaration.alias; node != nil && !seen[node]; node = node.declaration.alias {
		seen[node] = true
		if node.enumValues != nil || len(node.enumClauses) > 0 {
			return node
		}
	}
	return nil
}

func effectiveConstraintPrefix(node *valueOccurrenceNode) string {
	if node == nil || node.declarationID == "" {
		return ""
	}
	return fmt.Sprintf("attribute %q: ", node.declarationID)
}

func cloneValueSnapshots(values []valueSourceSnapshot) []valueSourceSnapshot {
	result := make([]valueSourceSnapshot, len(values))
	for index, value := range values {
		result[index] = value
		result[index].raw = copyValueRaw(value.raw)
	}
	return result
}

func rawSnapshotValues(values []valueSourceSnapshot) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = copyValueRaw(value.raw)
	}
	return result
}

func cloneEffectiveValidation(validation *ValidationExpr) *ValidationExpr {
	if validation == nil {
		return &ValidationExpr{}
	}
	copy := validation.Dup()
	if validation.Values != nil {
		copy.Values = make([]any, len(validation.Values))
		for index, value := range validation.Values {
			copy.Values[index] = copyValueRaw(value)
		}
	}
	if validation.EnumClauses != nil {
		copy.EnumClauses = make([][]any, len(validation.EnumClauses))
		for clauseIndex, clause := range validation.EnumClauses {
			copy.EnumClauses[clauseIndex] = make([]any, len(clause))
			for valueIndex, value := range clause {
				copy.EnumClauses[clauseIndex][valueIndex] = copyValueRaw(value)
			}
		}
	}
	copy.ExclusiveMinimum = copyValuePointer(validation.ExclusiveMinimum)
	copy.Minimum = copyValuePointer(validation.Minimum)
	copy.Maximum = copyValuePointer(validation.Maximum)
	copy.ExclusiveMaximum = copyValuePointer(validation.ExclusiveMaximum)
	copy.MinLength = copyValuePointer(validation.MinLength)
	copy.MaxLength = copyValuePointer(validation.MaxLength)
	return copy
}

func copyEnumClauseValues(clauses [][]any) [][]any {
	if clauses == nil {
		return nil
	}
	result := make([][]any, len(clauses))
	for clauseIndex, clause := range clauses {
		result[clauseIndex] = make([]any, len(clause))
		for valueIndex, value := range clause {
			result[clauseIndex][valueIndex] = copyValueRaw(value)
		}
	}
	return result
}
