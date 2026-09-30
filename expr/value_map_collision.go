package expr

import (
	"cmp"
	"errors"
	"reflect"
	"slices"
	"strconv"

	"github.com/CaliLuke/loom/internal/jsonkey"
)

type (
	valueCollisionCompatibility uint8

	valueMapKeyCollision struct {
		path        []string
		branches    []ValueIdentity
		branchNames []string
		name        string
	}

	valueCollisionEvidence struct {
		unknown    bool
		collisions []valueMapKeyCollision
	}

	valueCollisionResult struct {
		compatibility valueCollisionCompatibility
		evidence      valueCollisionEvidence
	}

	valueMapKeyCollisionQuery struct {
		context             *ValueContext
		occurrence          ValueOccurrence
		active              map[valueSnapshotVisit]bool
		compatibilityActive map[valueCompatibilityVisit]bool
		resolver            valueResolver
	}

	valueCompatibilityVisit struct {
		node  *valueOccurrenceNode
		value valueSnapshotVisit
	}
)

const (
	valueCollisionIncompatible valueCollisionCompatibility = iota
	valueCollisionCompatible
	valueCollisionUnknown
)

func (c *ValueContext) mapKeyCollisionResult(
	occurrence ValueOccurrence,
	source ValueSource,
) valueCollisionResult {
	if !c.ownsValue(occurrence, source) || errors.Is(source.data.snapshot.err, errValueSourceOwnership) {
		return collisionUnknown(valueCollisionUnknown)
	}
	query := valueMapKeyCollisionQuery{
		context:             c,
		occurrence:          occurrence,
		active:              make(map[valueSnapshotVisit]bool),
		compatibilityActive: make(map[valueCompatibilityVisit]bool),
	}
	query.resolver = valueResolver{
		context:    c,
		occurrence: occurrence,
		source:     source.data,
		active:     make(map[valueSnapshotVisit]bool),
	}
	return query.inspect(occurrence.node, source.data.snapshot.raw, nil)
}

func (q *valueMapKeyCollisionQuery) inspect(
	node *valueOccurrenceNode,
	raw any,
	path []string,
) valueCollisionResult {
	if node == nil || node.declaration == nil {
		return valueCollisionResult{compatibility: valueCollisionIncompatible}
	}
	compatibility := q.compatibility(node, raw)
	for node.declaration.alias != nil {
		node = node.declaration.alias
	}
	if valueNullInput(raw) {
		return collisionClear(compatibility)
	}
	if valueHasCustomCodec(reflect.TypeOf(raw)) {
		return collisionUnknown(compatibility)
	}
	if selected, handled, failure := selectedUnionSource(q.context.identity, q.occurrence, node, raw); handled {
		if failure.outcome != 0 {
			return collisionClear(valueCollisionIncompatible)
		}
		result := q.inspect(selected.branch.node, selected.payload, path)
		return qualifyCollisionBranch(result, q.branchIdentity(selected.branch), selected.branch.name)
	}
	var result valueCollisionResult
	switch node.declaration.kind {
	case UnionKind:
		result = q.inspectUnion(node, raw, path)
	case AnyKind:
		result = q.inspectAny(node, raw, path)
	default:
		result = q.inspectDereferenced(node, reflect.ValueOf(raw), path)
	}
	if node.declaration.kind != UnionKind {
		result.compatibility = compatibility
	}
	return result
}

func (q *valueMapKeyCollisionQuery) inspectDereferenced(
	node *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueCollisionResult {
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return collisionClear(valueCollisionIncompatible)
		}
		value = value.Elem()
	}
	if value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return collisionClear(valueCollisionCompatible)
		}
		if valueHasCustomCodec(value.Type()) {
			return collisionUnknown(valueCollisionUnknown)
		}
		leave, entered := q.enter(value)
		if !entered {
			return collisionUnknown(valueCollisionUnknown)
		}
		defer leave()
		return q.inspectDereferenced(node, value.Elem(), path)
	}
	if !value.IsValid() {
		return collisionClear(valueCollisionIncompatible)
	}
	raw := value.Interface()
	switch node.declaration.kind {
	case ObjectKind:
		return q.inspectObject(node, raw, path)
	case ArrayKind:
		return q.inspectArray(node, value, path)
	case MapKind:
		return q.inspectMap(node, value, path)
	default:
		return q.inspectScalar(node, raw)
	}
}

func (q *valueMapKeyCollisionQuery) inspectAny(
	node *valueOccurrenceNode,
	raw any,
	path []string,
) valueCollisionResult {
	value := reflect.ValueOf(raw)
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return collisionClear(valueCollisionCompatible)
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return collisionClear(valueCollisionCompatible)
	}
	if valueHasCustomCodec(value.Type()) {
		return collisionUnknown(valueCollisionCompatible)
	}
	switch value.Kind() {
	case reflect.Map:
		return q.inspectRawMap(node, value, path)
	case reflect.Array, reflect.Slice:
		return q.inspectRawArray(node, value, path)
	case reflect.Pointer, reflect.Struct:
		return collisionUnknown(valueCollisionCompatible)
	case reflect.Bool, reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return collisionClear(valueCollisionCompatible)
	default:
		return collisionUnknown(valueCollisionCompatible)
	}
}

func (q *valueMapKeyCollisionQuery) inspectArray(
	node *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueCollisionResult {
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		return collisionClear(valueCollisionIncompatible)
	}
	leave, entered := q.enter(value)
	if !entered {
		return collisionUnknown(valueCollisionUnknown)
	}
	defer leave()
	children := make([]valueCollisionResult, 0, value.Len())
	for index := range value.Len() {
		children = append(children, q.inspect(node.declaration.element, value.Index(index).Interface(),
			appendCollisionPath(path, strconv.Itoa(index))))
	}
	return combineCollisionChildren(children)
}

func (q *valueMapKeyCollisionQuery) inspectRawArray(
	node *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueCollisionResult {
	leave, entered := q.enter(value)
	if !entered {
		return collisionUnknown(valueCollisionUnknown)
	}
	defer leave()
	children := make([]valueCollisionResult, 0, value.Len())
	for index := range value.Len() {
		children = append(children, q.inspectAny(node, value.Index(index).Interface(),
			appendCollisionPath(path, strconv.Itoa(index))))
	}
	result := combineCollisionChildren(children)
	result.compatibility = valueCollisionCompatible
	return result
}

func (q *valueMapKeyCollisionQuery) inspectObject(
	node *valueOccurrenceNode,
	raw any,
	path []string,
) valueCollisionResult {
	value := reflect.ValueOf(raw)
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return collisionClear(valueCollisionIncompatible)
		}
		value = value.Elem()
	}
	if value.IsValid() && value.Kind() == reflect.Map {
		leave, entered := q.enter(value)
		if !entered {
			return collisionUnknown(q.compatibility(node, raw))
		}
		defer leave()
	}
	entries, ok := objectSourceEntries(node, raw)
	if !ok || entries == nil {
		return collisionClear(valueCollisionIncompatible)
	}
	known := make(map[string]bool)
	var children []valueCollisionResult
	for _, member := range node.declaration.members {
		known[member.name], known[member.wire] = true, true
		for _, name := range []string{member.name, member.wire} {
			if value, found := entries[name]; found {
				children = append(children, q.inspect(member.node, value, appendCollisionPath(path, name)))
			}
			if member.name == member.wire {
				break
			}
		}
	}
	var extras []string
	for name := range entries {
		if !known[name] {
			extras = append(extras, name)
		}
	}
	slices.Sort(extras)
	for _, name := range extras {
		children = append(children, q.inspectAny(node, entries[name], appendCollisionPath(path, name)))
	}
	result := combineCollisionChildren(children)
	result.compatibility = q.compatibility(node, raw)
	return result
}

func (q *valueMapKeyCollisionQuery) inspectMap(
	node *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueCollisionResult {
	if value.Kind() != reflect.Map {
		return collisionClear(valueCollisionIncompatible)
	}
	return q.inspectMapEntries(node, node.declaration.key, node.declaration.element, value, path)
}

func (q *valueMapKeyCollisionQuery) inspectRawMap(
	node *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueCollisionResult {
	return q.inspectMapEntries(node, nil, nil, value, path)
}

func (q *valueMapKeyCollisionQuery) inspectMapEntries(
	owner, keyNode, elementNode *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueCollisionResult {
	leave, entered := q.enter(value)
	if !entered {
		return collisionUnknown(valueCollisionUnknown)
	}
	defer leave()

	entries := valueSortedMapEntries(value)
	seen := make(map[string]bool)
	witnessed := make(map[string]bool)
	result := collisionClear(valueCollisionCompatible)
	for _, entry := range entries {
		key := q.mapKeyName(keyNode, entry.key.Interface())
		result.compatibility = combineCollisionCompatibility(result.compatibility, key.compatibility)
		switch {
		case !key.known:
			result.evidence.unknown = true
		case seen[key.name] && !witnessed[key.name]:
			result.evidence.collisions = append(result.evidence.collisions,
				valueMapKeyCollision{path: append([]string(nil), path...), name: key.name})
			witnessed[key.name] = true
		default:
			seen[key.name] = true
		}
		segment := key.name
		if !key.known {
			segment = valueKeyOrder(entry.key)
		}
		var child valueCollisionResult
		if elementNode == nil {
			child = q.inspectAny(owner, entry.value.Interface(), appendCollisionPath(path, segment))
		} else {
			child = q.inspect(elementNode, entry.value.Interface(), appendCollisionPath(path, segment))
		}
		result = combineCollisionChildren([]valueCollisionResult{result, child})
	}
	return result
}

type valueMapKeyName struct {
	compatibility valueCollisionCompatibility
	known         bool
	name          string
}

func (q *valueMapKeyCollisionQuery) mapKeyName(node *valueOccurrenceNode, raw any) valueMapKeyName {
	value := reflect.ValueOf(raw)
	if value.IsValid() && valueHasCustomCodec(value.Type()) {
		return valueMapKeyName{compatibility: valueCollisionUnknown}
	}
	if node == nil {
		name, ok := jsonkey.Name(value)
		return valueMapKeyName{compatibility: valueCollisionCompatible, known: ok, name: name}
	}
	for node.declaration.alias != nil {
		node = node.declaration.alias
	}
	if node.declaration.kind == AnyKind {
		name, ok := jsonkey.Name(value)
		return valueMapKeyName{compatibility: valueCollisionCompatible, known: ok, name: name}
	}
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return valueMapKeyName{compatibility: valueCollisionIncompatible}
		}
		if valueHasCustomCodec(value.Type()) {
			return valueMapKeyName{compatibility: valueCollisionUnknown}
		}
		if value.Kind() == reflect.Pointer {
			leave, entered := q.enter(value)
			if !entered {
				return valueMapKeyName{compatibility: valueCollisionUnknown}
			}
			defer leave()
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return valueMapKeyName{compatibility: valueCollisionIncompatible}
	}
	resolved := q.resolver.scalar(node, value.Interface())
	if resolved.outcome != ValueResolved || resolved.value.node == nil {
		return valueMapKeyName{compatibility: valueCollisionIncompatible}
	}
	name, ok := jsonkey.Name(reflect.ValueOf(resolved.value.node.scalar))
	return valueMapKeyName{compatibility: valueCollisionCompatible, known: ok, name: name}
}

func (q *valueMapKeyCollisionQuery) inspectUnion(
	node *valueOccurrenceNode,
	raw any,
	path []string,
) valueCollisionResult {
	var compatible []valueCollisionResult
	unknownEligibility := false
	for _, branch := range node.declaration.branches {
		result := q.inspect(branch.node, raw, path)
		switch result.compatibility {
		case valueCollisionCompatible:
			compatible = append(compatible, qualifyCollisionBranch(result,
				q.branchIdentity(branch), branch.name))
		case valueCollisionUnknown:
			unknownEligibility = true
		}
	}
	if len(compatible) == 0 {
		if unknownEligibility {
			return collisionUnknown(valueCollisionUnknown)
		}
		return collisionClear(valueCollisionIncompatible)
	}
	if unknownEligibility {
		return collisionUnknown(valueCollisionCompatible)
	}
	for _, result := range compatible {
		if len(result.evidence.collisions) == 0 {
			return collisionUnknown(valueCollisionCompatible)
		}
	}
	return combineCollisionChildren(compatible)
}

func (q *valueMapKeyCollisionQuery) inspectScalar(
	node *valueOccurrenceNode,
	raw any,
) valueCollisionResult {
	return collisionClear(q.compatibility(node, raw))
}

func (q *valueMapKeyCollisionQuery) compatibility(
	node *valueOccurrenceNode,
	raw any,
) valueCollisionCompatibility {
	if node == nil || node.declaration == nil {
		return valueCollisionIncompatible
	}
	if raw == nil {
		if AllowsNull(node.attribute) {
			return valueCollisionCompatible
		}
		return valueCollisionIncompatible
	}
	for node.declaration.alias != nil {
		node = node.declaration.alias
	}
	if selected, handled, failure := selectedUnionSource(q.context.identity, q.occurrence, node, raw); handled {
		if failure.outcome != 0 {
			return valueCollisionIncompatible
		}
		return q.compatibility(selected.branch.node, selected.payload)
	}
	value := reflect.ValueOf(raw)
	switch node.declaration.kind {
	case AnyKind:
		return valueCollisionCompatible
	case ObjectKind:
		if value.Kind() == reflect.Map || value.Kind() == reflect.Struct {
			return valueCollisionCompatible
		}
		return valueCollisionIncompatible
	case ArrayKind:
		return q.arrayCompatibility(node, value)
	case MapKind:
		return q.mapCompatibility(node, value)
	case UnionKind:
		return q.unionCompatibility(node, raw)
	default:
		if node.declaration.typ.IsCompatible(raw) {
			return valueCollisionCompatible
		}
		return valueCollisionIncompatible
	}
}

func (q *valueMapKeyCollisionQuery) arrayCompatibility(
	node *valueOccurrenceNode,
	value reflect.Value,
) valueCollisionCompatibility {
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		return valueCollisionIncompatible
	}
	leave, entered := q.enterCompatibility(node, value)
	if !entered {
		return valueCollisionUnknown
	}
	defer leave()
	result := valueCollisionCompatible
	for index := range value.Len() {
		result = combineCollisionCompatibility(result,
			q.compatibility(node.declaration.element, value.Index(index).Interface()))
	}
	return result
}

func (q *valueMapKeyCollisionQuery) mapCompatibility(
	node *valueOccurrenceNode,
	value reflect.Value,
) valueCollisionCompatibility {
	if value.Kind() != reflect.Map {
		return valueCollisionIncompatible
	}
	leave, entered := q.enterCompatibility(node, value)
	if !entered {
		return valueCollisionUnknown
	}
	defer leave()
	result := valueCollisionCompatible
	for _, entry := range valueSortedMapEntries(value) {
		result = combineCollisionCompatibility(result,
			q.compatibility(node.declaration.key, entry.key.Interface()))
		result = combineCollisionCompatibility(result,
			q.compatibility(node.declaration.element, entry.value.Interface()))
	}
	return result
}

func (q *valueMapKeyCollisionQuery) unionCompatibility(
	node *valueOccurrenceNode,
	raw any,
) valueCollisionCompatibility {
	result := valueCollisionIncompatible
	for _, branch := range node.declaration.branches {
		branchResult := q.compatibility(branch.node, raw)
		if branchResult == valueCollisionCompatible {
			return valueCollisionCompatible
		}
		if branchResult == valueCollisionUnknown {
			result = valueCollisionUnknown
		}
	}
	return result
}

func (q *valueMapKeyCollisionQuery) enterCompatibility(
	node *valueOccurrenceNode,
	value reflect.Value,
) (func(), bool) {
	visit, tracked := valueTrackedVisit(value)
	if !tracked || value.IsNil() {
		return func() {}, true
	}
	key := valueCompatibilityVisit{node: node, value: visit}
	if q.compatibilityActive[key] {
		return nil, false
	}
	q.compatibilityActive[key] = true
	return func() {
		delete(q.compatibilityActive, key)
	}, true
}

func (q *valueMapKeyCollisionQuery) branchIdentity(branch valueOccurrenceBranch) ValueIdentity {
	return ValueIdentity{context: q.context.identity, graph: q.occurrence.graph, index: branch.id}
}

func (q *valueMapKeyCollisionQuery) enter(value reflect.Value) (func(), bool) {
	visit, tracked := valueTrackedVisit(value)
	if !tracked || value.IsNil() {
		return func() {}, true
	}
	if q.active[visit] {
		return nil, false
	}
	q.active[visit] = true
	return func() {
		delete(q.active, visit)
	}, true
}

func collisionClear(compatibility valueCollisionCompatibility) valueCollisionResult {
	return valueCollisionResult{compatibility: compatibility}
}

func collisionUnknown(compatibility valueCollisionCompatibility) valueCollisionResult {
	return valueCollisionResult{compatibility: compatibility, evidence: valueCollisionEvidence{unknown: true}}
}

func combineCollisionChildren(children []valueCollisionResult) valueCollisionResult {
	result := collisionClear(valueCollisionCompatible)
	for _, child := range children {
		result.compatibility = combineCollisionCompatibility(result.compatibility, child.compatibility)
		result.evidence.unknown = result.evidence.unknown || child.evidence.unknown
		result.evidence.collisions = append(result.evidence.collisions, child.evidence.collisions...)
	}
	return result
}

func combineCollisionCompatibility(
	left, right valueCollisionCompatibility,
) valueCollisionCompatibility {
	if left == valueCollisionIncompatible || right == valueCollisionIncompatible {
		return valueCollisionIncompatible
	}
	if left == valueCollisionUnknown || right == valueCollisionUnknown {
		return valueCollisionUnknown
	}
	return valueCollisionCompatible
}

func qualifyCollisionBranch(
	result valueCollisionResult,
	identity ValueIdentity,
	name string,
) valueCollisionResult {
	for index := range result.evidence.collisions {
		result.evidence.collisions[index].branches = append(
			[]ValueIdentity{identity}, result.evidence.collisions[index].branches...)
		result.evidence.collisions[index].branchNames = append(
			[]string{name}, result.evidence.collisions[index].branchNames...)
	}
	return result
}

func appendCollisionPath(path []string, segment string) []string {
	return append(append([]string(nil), path...), segment)
}

func compareValueMapKeyCollisions(left, right valueMapKeyCollision) int {
	if order := slices.Compare(left.path, right.path); order != 0 {
		return order
	}
	if order := slices.CompareFunc(left.branches, right.branches, func(left, right ValueIdentity) int {
		return cmp.Compare(left.index, right.index)
	}); order != 0 {
		return order
	}
	return cmp.Compare(left.name, right.name)
}
