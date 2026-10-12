package expr

import "reflect"

// enumClauseDeclaration finds the captured owner of one retained predicate.
// A later refinement changes membership, not the meaning of its constants.
func enumClauseDeclaration(occurrence ValueOccurrence, clause []any) ValueOccurrence {
	constraints := occurrence.node.constraints
	if constraints == nil {
		return ValueOccurrence{}
	}
	for index, sourceClause := range constraints.validation.rules.Enums() {
		if reflect.DeepEqual(sourceClause, clause) {
			return ValueOccurrence{context: occurrence.context, graph: occurrence.graph, node: constraints.validation.enumSources[index]}
		}
	}
	return ValueOccurrence{}
}

// enumDeclarationPlan retains declaration member identities and representation.
// It uses the same builder as ordinary plans; no value is resolved against the
// refined occurrence merely to construct an inherited predicate constant.
func (c *ValueContext) enumDeclarationPlan(occurrence ValueOccurrence) (*valuePlanNode, error) {
	builder := &valuePlanBuilder{
		request: ValuePlanRequest{Codec: ValueCodecJSON, Use: ValuePlanDocumentation},
		context: c, source: occurrence, nodes: make(map[valuePlanKey]*valuePlanNode),
		assocs: make(map[*valueOccurrenceNode][]*valuePlanAssociation),
	}
	return builder.node(occurrence.node, occurrence.node, ValueCodecJSON)
}

// enumConstantPlan binds the declaring semantic identities to the actual target
// representation using the ordinary builder and policies. Target predicates
// still constrain actual values, but cannot discard inherited constants.
func (b *valuePlanBuilder) enumConstantPlan(declaration ValueOccurrence, target *valueOccurrenceNode, codec ValueCodec) (*valuePlanNode, error) {
	builder := &valuePlanBuilder{
		enumConstant: true, request: b.request, context: b.context, source: declaration,
		nodes: make(map[valuePlanKey]*valuePlanNode), assocs: make(map[*valueOccurrenceNode][]*valuePlanAssociation),
		used: make([]bool, len(b.request.Fields)), containers: make([]bool, len(b.request.Containers)), codecs: make([]bool, len(b.request.Codecs)),
	}
	return builder.node(declaration.node, target, codec)
}

func (node *valuePlanNode) enumProjection(index int) *valuePlanNode {
	if index < len(node.enumPlans) {
		return node.enumPlans[index]
	}
	return node
}

// enumRootPlan detaches only the root and its named ancestry. Clause
// constants are projected without root membership predicates; declaration checks
// suppress only the enum conjunction. Nested constraints and branch selection
// remain owned by the unchanged captured child nodes.
func enumRootPlan(node *valuePlanNode, shape bool) *valuePlanNode {
	copy := *node
	copy.hasEnum, copy.enumClauses = false, nil
	if shape {
		copy.validation = nil
	}
	if copy.alias != nil {
		copy.alias = enumRootPlan(copy.alias, shape)
	}
	return &copy
}
