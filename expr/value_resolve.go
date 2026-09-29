package expr

import (
	"errors"
	"reflect"

	"github.com/CaliLuke/loom/internal/examplevalue"
)

type valueResolver struct {
	context    *ValueContext
	occurrence ValueOccurrence
	source     *valueSourceData
	active     map[valueSnapshotVisit]bool
}

// Resolve interprets one immutable source against its effective semantic
// occurrence. It never receives a target, reselects an already selected branch,
// or substitutes generated data for an authored failure.
func (c *ValueContext) Resolve(occurrence ValueOccurrence, source ValueSource, role ValueRole) ValueResult {
	if c == nil || occurrence.node == nil || source.data == nil || occurrence.context != c.identity || source.data.context != c.identity {
		return valueFailure(ValueInvalid, "ownership", "source and occurrence require the same value context")
	}
	if role < ValueRoleExample || role > ValueRoleDefault {
		return valueFailure(ValueInvalid, "role", "unknown source role")
	}
	key := valueResolutionKey{occurrence.node, source.data, role}
	c.mu.Lock()
	defer c.mu.Unlock()
	if prior, found := c.resolved[key]; found {
		return prior
	}
	resolver := valueResolver{context: c, occurrence: occurrence, source: source.data, active: make(map[valueSnapshotVisit]bool)}
	var result ValueResult
	if errors.Is(source.data.snapshot.err, errValueSourceOwnership) {
		result = valueFailure(ValueInvalid, "ownership", source.data.snapshot.err.Error())
	} else {
		result = resolver.resolve(occurrence.node, source.data.snapshot.raw, role != ValueRoleExample, occurrence.node, nil)
	}
	result.legacy = &source.data.snapshot
	result.role = role
	result.source, result.occurrence = source.data, occurrence
	result.customBoundary = result.outcome == ValueUnsupported && valueCustomSource(source.data.snapshot.raw)
	c.resolved[key] = result
	return result
}

func (r *valueResolver) resolve(node *valueOccurrenceNode, raw any, complete bool, owner *valueOccurrenceNode, skipEnum *valueOccurrenceNode) ValueResult {
	result := r.body(node, raw, complete, owner, skipEnum)
	if result.outcome != ValueResolved && result.outcome != ValueIncomplete {
		return result
	}
	if node == skipEnum || len(node.enumValues) == 0 {
		return result
	}
	for _, allowed := range node.enumValues {
		if allowed.err != nil {
			return valueFailure(ValueInvalid, "enum", "invalid authored enum member")
		}
		member := r.resolve(node, allowed.raw, true, owner, node)
		if member.outcome == ValueResolved && resolvedEnumEqual(member.value, result.value) {
			return result
		}
	}
	return valueFailure(ValueInvalid, "enum", "value does not match any enum member")
}

func (r *valueResolver) body(node *valueOccurrenceNode, raw any, complete bool, owner *valueOccurrenceNode, skipEnum *valueOccurrenceNode) ValueResult {
	if valueNullInput(raw) && node.attribute.Nullable {
		return r.success(node, ValueNull, 0, raw)
	}
	if valueCustomSource(raw) {
		return valueFailure(ValueUnsupported, "custom", "custom values require their target codec")
	}
	declaration := node.declaration
	if declaration.alias != nil {
		result := r.resolve(declaration.alias, raw, complete, owner, skipEnum)
		if result.outcome == ValueResolved || result.outcome == ValueIncomplete {
			if !valueLocalRules(node.attribute, result.value) {
				return valueFailure(ValueInvalid, "validation", "value violates occurrence constraints")
			}
		}
		return result
	}
	if declaration.kind != AnyKind && declaration.kind != UnionKind {
		leave, entered := r.enter(raw)
		if !entered {
			return valueFailure(ValueInvalid, "cyclic", "cyclic value")
		}
		defer leave()
	}
	var result ValueResult
	switch declaration.kind {
	case AnyKind:
		result = r.raw(node, raw)
		if result.outcome == ValueResolved {
			wrapper := r.success(node, ValuePresent, ValueKindAny, raw)
			wrapper.value.node.payload = result.value
			result = wrapper
		}
	case ObjectKind:
		result = r.object(node, owner, raw, complete)
	case ArrayKind:
		result = r.array(node, raw, complete)
	case MapKind:
		result = r.mapping(node, raw, complete)
	case UnionKind:
		result = r.union(node, raw, complete)
	default:
		result = r.scalar(node, raw)
	}
	if (result.outcome == ValueResolved || result.outcome == ValueIncomplete) && !valueLocalRules(node.attribute, result.value) {
		return valueFailure(ValueInvalid, "validation", "value violates occurrence constraints")
	}
	return result
}

func valueCustomSource(raw any) bool {
	if valueNullInput(raw) {
		return false
	}
	switch raw.(type) {
	case valueSelectedInput, examplevalue.Union:
		return false
	}
	typ := reflect.TypeOf(raw)
	return valueHasCustomCodec(typ) || typ.Kind() == reflect.Struct || typ.Kind() == reflect.Pointer
}

func (r *valueResolver) enter(raw any) (func(), bool) {
	value := reflect.ValueOf(raw)
	if !value.IsValid() || (value.Kind() != reflect.Map && value.Kind() != reflect.Slice) || value.IsNil() {
		return func() {
		}, true
	}
	visit := valueSnapshotVisit{value.Type(), value.Pointer(), value.Len()}
	if r.active[visit] {
		return nil, false
	}
	r.active[visit] = true
	return func() {
		delete(r.active, visit)
	}, true
}

func (r *valueResolver) success(node *valueOccurrenceNode, presence ValuePresence, kind ValueKind, raw any) ValueResult {
	occurrence := r.occurrence
	occurrence.node = node
	return ValueResult{outcome: ValueResolved, value: ResolvedValue{node: &resolvedValueNode{
		context: r.context.identity, source: r.source, occurrence: occurrence,
		presence: presence, kind: kind, raw: raw,
	}}}
}

func (r *valueResolver) union(node *valueOccurrenceNode, raw any, complete bool) ValueResult {
	if handled, result := r.selectedUnion(node, raw, complete); handled {
		return result
	}
	branches := node.declaration.branches
	type candidate struct {
		branch    valueOccurrenceBranch
		result    ValueResult
		preferred bool
	}
	var candidates []candidate
	for _, branch := range branches {
		result := r.resolve(branch.node, raw, true, branch.node, nil)
		if result.outcome == ValueResolved && len(result.missing) == 0 {
			candidates = append(candidates, candidate{branch, result, valuePrefersObject(branch.node, raw)})
		}
	}
	if len(candidates) == 0 && !complete {
		for _, branch := range branches {
			result := r.resolve(branch.node, raw, false, branch.node, nil)
			if result.outcome == ValueResolved || result.outcome == ValueIncomplete || result.outcome == ValueAmbiguous {
				candidates = append(candidates, candidate{branch, result, valuePrefersObject(branch.node, raw)})
			}
		}
	}
	var preferred []candidate
	for _, candidate := range candidates {
		if candidate.preferred {
			preferred = append(preferred, candidate)
		}
	}
	if len(preferred) > 0 {
		candidates = preferred
	}
	if len(candidates) == 0 {
		return valueFailure(ValueInvalid, "union", "no viable union branch")
	}
	if len(candidates) != 1 {
		return valueFailure(ValueAmbiguous, "union", "multiple viable union branches")
	}
	chosen := candidates[0]
	return r.wrapBranch(node, chosen.branch, chosen.result)
}

func (r *valueResolver) wrapBranch(node *valueOccurrenceNode, branch valueOccurrenceBranch, result ValueResult) ValueResult {
	if result.outcome != ValueResolved && result.outcome != ValueIncomplete {
		return result
	}
	wrapped := r.success(node, ValuePresent, ValueKindUnion, result.value.node.raw)
	wrapped.value.node.branch = ValueIdentity{context: r.context.identity, graph: r.occurrence.graph, index: branch.id}
	wrapped.value.node.payload = result.value
	wrapped.missing, wrapped.outcome = result.missing, result.outcome
	return wrapped
}

func valuePrefersObject(node *valueOccurrenceNode, raw any) bool {
	entries, ok := stringMapExample(raw)
	if !ok {
		return false
	}
	for node.declaration.alias != nil {
		node = node.declaration.alias
	}
	if node.declaration.kind != ObjectKind {
		return true
	}
	for _, member := range node.declaration.members {
		if _, ok := entries[member.name]; ok {
			return true
		}
		if _, ok := entries[member.wire]; ok {
			return true
		}
	}
	return false
}

func valueNullInput(raw any) bool {
	if raw == nil {
		return true
	}
	value := reflect.ValueOf(raw)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

func valueFailure(outcome ValueOutcome, code, message string) ValueResult {
	return ValueResult{outcome: outcome, diagnostics: []ValueDiagnostic{{Code: code, Message: message}}}
}

func valueErrorPriority(result ValueResult) int {
	if result.outcome == ValueResolved || result.outcome == ValueIncomplete {
		return 99
	}
	if len(result.diagnostics) > 0 && result.diagnostics[0].Code == "cyclic" {
		return 1
	}
	switch result.outcome {
	case ValueUnsupported:
		return 2
	case ValueAmbiguous:
		return 4
	default:
		return 3
	}
}

func valueCombine(parent ValueResult, children []ValueResult) ValueResult {
	failure := parent
	for _, child := range children {
		if valueErrorPriority(child) < valueErrorPriority(failure) {
			failure = child
		}
	}
	if valueErrorPriority(failure) != 99 {
		return failure
	}
	for _, child := range children {
		parent.missing = append(parent.missing, child.missing...)
	}
	if len(parent.missing) > 0 {
		parent.outcome = ValueIncomplete
	}
	return parent
}

func valueAtPath(result ValueResult, segment string) ValueResult {
	result.diagnostics = result.Diagnostics()
	for index := range result.diagnostics {
		result.diagnostics[index].Path = append([]string{segment}, result.diagnostics[index].Path...)
	}
	return result
}

func (r *valueResolver) selectedUnion(node *valueOccurrenceNode, raw any, complete bool) (bool, ValueResult) {
	branches := node.declaration.branches
	if selected, ok := raw.(valueSelectedInput); ok {
		occurrence := r.occurrence
		occurrence.node = node
		if selected.occurrence != occurrence.ID() {
			return true, valueFailure(ValueInvalid, "branch", "selected occurrence does not match semantic owner")
		}
		for _, branch := range branches {
			if selected.branch == (ValueIdentity{context: r.context.identity, graph: r.occurrence.graph, index: branch.id}) {
				return true, r.wrapBranch(node, branch, r.resolve(branch.node, selected.payload, complete, branch.node, nil))
			}
		}
		return true, valueFailure(ValueInvalid, "branch", "selected branch does not belong to occurrence")
	}
	if selected, ok := raw.(examplevalue.Union); ok {
		if selected.Branch < 0 || selected.Branch >= len(branches) {
			return true, valueFailure(ValueInvalid, "branch", "selected branch does not belong to occurrence")
		}
		branch := branches[selected.Branch]
		return true, r.wrapBranch(node, branch, r.resolve(branch.node, selected.Value, complete, branch.node, nil))
	}
	return false, ValueResult{}
}
