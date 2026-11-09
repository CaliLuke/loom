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
	valueExampleCompatibility uint8

	valueMapKeyCollision struct {
		path        []string
		branches    []ValueIdentity
		branchNames []string
		name        string
	}

	valueUndeclaredMember struct {
		path []string
		name string
	}

	valueExampleEvidence struct {
		unknown    bool
		collisions []valueMapKeyCollision
		undeclared []valueUndeclaredMember
	}

	valueExampleShapeResult struct {
		compatibility valueExampleCompatibility
		evidence      valueExampleEvidence
	}

	// valueExampleShapeQuery checks structural facts of partial examples without
	// running codecs or applying value predicates. Evidence stays separate from
	// the historical shape compatibility used to consider union alternatives.
	valueExampleShapeQuery struct {
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
	valueExampleIncompatible valueExampleCompatibility = iota
	valueExampleCompatible
	valueExampleUnknown
)

func (c *ValueContext) exampleShapeResult(
	occurrence ValueOccurrence,
	source ValueSource,
) valueExampleShapeResult {
	if !c.ownsValue(occurrence, source) || errors.Is(source.data.snapshot.err, errValueSourceOwnership) {
		return exampleShapeUnknown(valueExampleUnknown)
	}
	query := valueExampleShapeQuery{
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

func (q *valueExampleShapeQuery) inspect(
	node *valueOccurrenceNode,
	raw any,
	path []string,
) valueExampleShapeResult {
	if node == nil || node.declaration == nil {
		return valueExampleShapeResult{compatibility: valueExampleIncompatible}
	}
	compatibility := q.compatibility(node, raw)
	owner := node
	for node.declaration.alias != nil {
		node = node.declaration.alias
	}
	if valueNullInput(raw) {
		return exampleShapeClear(compatibility)
	}
	if valueHasCustomCodec(reflect.TypeOf(raw)) {
		return exampleShapeUnknown(compatibility)
	}
	if selected, handled, failure := selectedUnionSource(q.context.identity, q.occurrence, node, raw); handled {
		if failure.outcome != 0 {
			return exampleShapeClear(valueExampleIncompatible)
		}
		result := q.inspect(selected.branch.node, selected.payload, path)
		return qualifyCollisionBranch(result, q.branchIdentity(selected.branch), selected.branch.name)
	}
	var result valueExampleShapeResult
	switch node.declaration.kind {
	case UnionKind:
		result = q.inspectUnion(node, raw, path)
	case AnyKind:
		result = q.inspectAny(node, raw, path)
	default:
		result = q.inspectDereferenced(node, owner, reflect.ValueOf(raw), path)
	}
	if node.declaration.kind != UnionKind {
		result.compatibility = compatibility
	}
	return result
}

func (q *valueExampleShapeQuery) inspectDereferenced(
	node, owner *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueExampleShapeResult {
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return exampleShapeClear(valueExampleIncompatible)
		}
		value = value.Elem()
	}
	if value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return exampleShapeClear(valueExampleCompatible)
		}
		if valueHasCustomCodec(value.Type()) {
			return exampleShapeUnknown(valueExampleUnknown)
		}
		leave, entered := q.enter(value)
		if !entered {
			return exampleShapeUnknown(valueExampleUnknown)
		}
		defer leave()
		return q.inspectDereferenced(node, owner, value.Elem(), path)
	}
	if !value.IsValid() {
		return exampleShapeClear(valueExampleIncompatible)
	}
	raw := value.Interface()
	switch node.declaration.kind {
	case ObjectKind:
		return q.inspectObject(node, owner, raw, path)
	case ArrayKind:
		return q.inspectArray(node, value, path)
	case MapKind:
		return q.inspectMap(node, value, path)
	default:
		return q.inspectScalar(node, raw)
	}
}

func (q *valueExampleShapeQuery) inspectAny(
	node *valueOccurrenceNode,
	raw any,
	path []string,
) valueExampleShapeResult {
	value := reflect.ValueOf(raw)
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return exampleShapeClear(valueExampleCompatible)
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return exampleShapeClear(valueExampleCompatible)
	}
	if valueHasCustomCodec(value.Type()) {
		return exampleShapeUnknown(valueExampleCompatible)
	}
	switch value.Kind() {
	case reflect.Map:
		return q.inspectRawMap(node, value, path)
	case reflect.Array, reflect.Slice:
		return q.inspectRawArray(node, value, path)
	case reflect.Pointer, reflect.Struct:
		return exampleShapeUnknown(valueExampleCompatible)
	case reflect.Bool, reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return exampleShapeClear(valueExampleCompatible)
	default:
		return exampleShapeUnknown(valueExampleCompatible)
	}
}

func (q *valueExampleShapeQuery) inspectArray(
	node *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueExampleShapeResult {
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		return exampleShapeClear(valueExampleIncompatible)
	}
	leave, entered := q.enter(value)
	if !entered {
		return exampleShapeUnknown(valueExampleUnknown)
	}
	defer leave()
	children := make([]valueExampleShapeResult, 0, value.Len())
	for index := range value.Len() {
		children = append(children, q.inspect(node.declaration.element, value.Index(index).Interface(),
			appendCollisionPath(path, strconv.Itoa(index))))
	}
	return combineExampleShapeChildren(children)
}

func (q *valueExampleShapeQuery) inspectRawArray(
	node *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueExampleShapeResult {
	leave, entered := q.enter(value)
	if !entered {
		return exampleShapeUnknown(valueExampleUnknown)
	}
	defer leave()
	children := make([]valueExampleShapeResult, 0, value.Len())
	for index := range value.Len() {
		children = append(children, q.inspectAny(node, value.Index(index).Interface(),
			appendCollisionPath(path, strconv.Itoa(index))))
	}
	result := combineExampleShapeChildren(children)
	result.compatibility = valueExampleCompatible
	return result
}

func (q *valueExampleShapeQuery) inspectObject(
	node, owner *valueOccurrenceNode,
	raw any,
	path []string,
) valueExampleShapeResult {
	value := reflect.ValueOf(raw)
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return exampleShapeClear(valueExampleIncompatible)
		}
		value = value.Elem()
	}
	if value.IsValid() && value.Kind() == reflect.Map {
		leave, entered := q.enter(value)
		if !entered {
			return exampleShapeUnknown(q.compatibility(node, raw))
		}
		defer leave()
	}
	entries, ok := objectSourceEntries(node, raw)
	if !ok || entries == nil {
		return exampleShapeClear(valueExampleIncompatible)
	}
	known := make(map[string]bool)
	var children []valueExampleShapeResult
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
		if node.additional != nil {
			children = append(children, q.inspect(node.additional, entries[name], appendCollisionPath(path, name)))
		} else {
			children = append(children, q.inspectAny(node, entries[name], appendCollisionPath(path, name)))
		}
	}
	result := combineExampleShapeChildren(children)
	if !owner.attribute.effectiveObjectOpenness().AdmitsUndeclared() {
		for _, name := range extras {
			result.evidence.undeclared = append(result.evidence.undeclared, valueUndeclaredMember{path: append([]string(nil), path...), name: name})
		}
	}
	result.compatibility = q.compatibility(node, raw)
	return result
}

func (q *valueExampleShapeQuery) inspectMap(
	node *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueExampleShapeResult {
	if value.Kind() != reflect.Map {
		return exampleShapeClear(valueExampleIncompatible)
	}
	return q.inspectMapEntries(node, node.declaration.key, node.declaration.element, value, path)
}

func (q *valueExampleShapeQuery) inspectRawMap(
	node *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueExampleShapeResult {
	return q.inspectMapEntries(node, nil, nil, value, path)
}

func (q *valueExampleShapeQuery) inspectMapEntries(
	owner, keyNode, elementNode *valueOccurrenceNode,
	value reflect.Value,
	path []string,
) valueExampleShapeResult {
	leave, entered := q.enter(value)
	if !entered {
		return exampleShapeUnknown(valueExampleUnknown)
	}
	defer leave()

	entries := valueSortedMapEntries(value)
	seen := make(map[string]bool)
	witnessed := make(map[string]bool)
	result := exampleShapeClear(valueExampleCompatible)
	for _, entry := range entries {
		key := q.mapKeyName(keyNode, entry.key.Interface())
		result.compatibility = combineExampleCompatibility(result.compatibility, key.compatibility)
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
		var child valueExampleShapeResult
		if elementNode == nil {
			child = q.inspectAny(owner, entry.value.Interface(), appendCollisionPath(path, segment))
		} else {
			child = q.inspect(elementNode, entry.value.Interface(), appendCollisionPath(path, segment))
		}
		result = combineExampleShapeChildren([]valueExampleShapeResult{result, child})
	}
	return result
}

type valueMapKeyName struct {
	compatibility valueExampleCompatibility
	known         bool
	name          string
}

func (q *valueExampleShapeQuery) mapKeyName(node *valueOccurrenceNode, raw any) valueMapKeyName {
	value := reflect.ValueOf(raw)
	if value.IsValid() && valueHasCustomCodec(value.Type()) {
		return valueMapKeyName{compatibility: valueExampleUnknown}
	}
	if node == nil {
		name, ok := jsonkey.Name(value)
		return valueMapKeyName{compatibility: valueExampleCompatible, known: ok, name: name}
	}
	for node.declaration.alias != nil {
		node = node.declaration.alias
	}
	if node.declaration.kind == AnyKind {
		name, ok := jsonkey.Name(value)
		return valueMapKeyName{compatibility: valueExampleCompatible, known: ok, name: name}
	}
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return valueMapKeyName{compatibility: valueExampleIncompatible}
		}
		if valueHasCustomCodec(value.Type()) {
			return valueMapKeyName{compatibility: valueExampleUnknown}
		}
		if value.Kind() == reflect.Pointer {
			leave, entered := q.enter(value)
			if !entered {
				return valueMapKeyName{compatibility: valueExampleUnknown}
			}
			defer leave()
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return valueMapKeyName{compatibility: valueExampleIncompatible}
	}
	resolved := q.resolver.scalar(node, value.Interface())
	if resolved.outcome != ValueResolved || resolved.value.node == nil {
		return valueMapKeyName{compatibility: valueExampleIncompatible}
	}
	name, ok := jsonkey.Name(reflect.ValueOf(resolved.value.node.scalar))
	return valueMapKeyName{compatibility: valueExampleCompatible, known: ok, name: name}
}

func (q *valueExampleShapeQuery) inspectUnion(
	node *valueOccurrenceNode,
	raw any,
	path []string,
) valueExampleShapeResult {
	var compatible []valueExampleShapeResult
	unknownEligibility := false
	for _, branch := range node.declaration.branches {
		result := q.inspect(branch.node, raw, path)
		switch result.compatibility {
		case valueExampleCompatible:
			compatible = append(compatible, qualifyCollisionBranch(result,
				q.branchIdentity(branch), branch.name))
		case valueExampleUnknown:
			unknownEligibility = true
		}
	}
	if len(compatible) == 0 {
		if unknownEligibility {
			return exampleShapeUnknown(valueExampleUnknown)
		}
		return exampleShapeClear(valueExampleIncompatible)
	}
	if unknownEligibility {
		return exampleShapeUnknown(valueExampleCompatible)
	}
	for _, result := range compatible {
		if len(result.evidence.collisions) == 0 && len(result.evidence.undeclared) == 0 {
			return exampleShapeUnknown(valueExampleCompatible)
		}
	}
	return combineExampleShapeChildren(compatible)
}

func (q *valueExampleShapeQuery) inspectScalar(
	node *valueOccurrenceNode,
	raw any,
) valueExampleShapeResult {
	return exampleShapeClear(q.compatibility(node, raw))
}

func (q *valueExampleShapeQuery) compatibility(
	node *valueOccurrenceNode,
	raw any,
) valueExampleCompatibility {
	if node == nil || node.declaration == nil {
		return valueExampleIncompatible
	}
	if raw == nil {
		if AllowsNull(node.attribute) {
			return valueExampleCompatible
		}
		return valueExampleIncompatible
	}
	for node.declaration.alias != nil {
		node = node.declaration.alias
	}
	if selected, handled, failure := selectedUnionSource(q.context.identity, q.occurrence, node, raw); handled {
		if failure.outcome != 0 {
			return valueExampleIncompatible
		}
		return q.compatibility(selected.branch.node, selected.payload)
	}
	value := reflect.ValueOf(raw)
	switch node.declaration.kind {
	case AnyKind:
		return valueExampleCompatible
	case ObjectKind:
		if value.Kind() == reflect.Map || value.Kind() == reflect.Struct {
			return valueExampleCompatible
		}
		return valueExampleIncompatible
	case ArrayKind:
		return q.arrayCompatibility(node, value)
	case MapKind:
		return q.mapCompatibility(node, value)
	case UnionKind:
		return q.unionCompatibility(node, raw)
	default:
		if node.declaration.typ.IsCompatible(raw) {
			return valueExampleCompatible
		}
		return valueExampleIncompatible
	}
}

func (q *valueExampleShapeQuery) arrayCompatibility(
	node *valueOccurrenceNode,
	value reflect.Value,
) valueExampleCompatibility {
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		return valueExampleIncompatible
	}
	leave, entered := q.enterCompatibility(node, value)
	if !entered {
		return valueExampleUnknown
	}
	defer leave()
	result := valueExampleCompatible
	for index := range value.Len() {
		result = combineExampleCompatibility(result,
			q.compatibility(node.declaration.element, value.Index(index).Interface()))
	}
	return result
}

func (q *valueExampleShapeQuery) mapCompatibility(
	node *valueOccurrenceNode,
	value reflect.Value,
) valueExampleCompatibility {
	if value.Kind() != reflect.Map {
		return valueExampleIncompatible
	}
	leave, entered := q.enterCompatibility(node, value)
	if !entered {
		return valueExampleUnknown
	}
	defer leave()
	result := valueExampleCompatible
	for _, entry := range valueSortedMapEntries(value) {
		result = combineExampleCompatibility(result,
			q.compatibility(node.declaration.key, entry.key.Interface()))
		result = combineExampleCompatibility(result,
			q.compatibility(node.declaration.element, entry.value.Interface()))
	}
	return result
}

func (q *valueExampleShapeQuery) unionCompatibility(
	node *valueOccurrenceNode,
	raw any,
) valueExampleCompatibility {
	result := valueExampleIncompatible
	for _, branch := range node.declaration.branches {
		branchResult := q.compatibility(branch.node, raw)
		if branchResult == valueExampleCompatible {
			return valueExampleCompatible
		}
		if branchResult == valueExampleUnknown {
			result = valueExampleUnknown
		}
	}
	return result
}

func (q *valueExampleShapeQuery) enterCompatibility(
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

func (q *valueExampleShapeQuery) branchIdentity(branch valueOccurrenceBranch) ValueIdentity {
	return ValueIdentity{context: q.context.identity, graph: q.occurrence.graph, index: branch.id}
}

func (q *valueExampleShapeQuery) enter(value reflect.Value) (func(), bool) {
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

func exampleShapeClear(compatibility valueExampleCompatibility) valueExampleShapeResult {
	return valueExampleShapeResult{compatibility: compatibility}
}

func exampleShapeUnknown(compatibility valueExampleCompatibility) valueExampleShapeResult {
	return valueExampleShapeResult{compatibility: compatibility, evidence: valueExampleEvidence{unknown: true}}
}

func combineExampleShapeChildren(children []valueExampleShapeResult) valueExampleShapeResult {
	result := exampleShapeClear(valueExampleCompatible)
	for _, child := range children {
		result.compatibility = combineExampleCompatibility(result.compatibility, child.compatibility)
		result.evidence.unknown = result.evidence.unknown || child.evidence.unknown
		result.evidence.collisions = append(result.evidence.collisions, child.evidence.collisions...)
		result.evidence.undeclared = append(result.evidence.undeclared, child.evidence.undeclared...)
	}
	return result
}

func combineExampleCompatibility(
	left, right valueExampleCompatibility,
) valueExampleCompatibility {
	if left == valueExampleIncompatible || right == valueExampleIncompatible {
		return valueExampleIncompatible
	}
	if left == valueExampleUnknown || right == valueExampleUnknown {
		return valueExampleUnknown
	}
	return valueExampleCompatible
}

func qualifyCollisionBranch(
	result valueExampleShapeResult,
	identity ValueIdentity,
	name string,
) valueExampleShapeResult {
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
