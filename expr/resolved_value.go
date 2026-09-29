package expr

import (
	"encoding/json/jsontext"
	"sync"
)

type (
	// ValueContext owns semantic values for one generation. Handles from different
	// contexts cannot be mixed. Builtin source data and all returned slices are copied.
	ValueContext struct {
		identity    *valueContextIdentity
		mu          sync.Mutex
		synthesisMu sync.Mutex
		examples    map[*ExampleExpr]*valueSourceData
		resolved    map[valueResolutionKey]ValueResult
		synthesized map[*valueOccurrenceNode]ValueResult
		projected   map[valueProjectionKey]*valueProjectionSlot
	}

	// ValueIdentity is a comparable opaque identity within a generation. Its zero
	// value identifies nothing; it is never a content hash or a generation seed.
	ValueIdentity struct {
		context *valueContextIdentity
		graph   *valueOccurrenceGraph
		source  *valueSourceData
		value   *resolvedValueNode
		index   uint64
	}

	// ValueRole selects the source completeness contract.
	ValueRole uint8
	// ValueOutcome distinguishes semantic success, partial examples and failures.
	ValueOutcome uint8
	// ValuePresence distinguishes absent, null, typed nil and concrete values.
	ValuePresence uint8
	// ValueKind describes a resolved node independently of its Go representation.
	ValueKind uint8

	// ValueInput supplies a source. Raw is copied for finite builtin values; custom
	// values remain borrowed until target materialization. Origin is diagnostic only.
	ValueInput struct {
		// Raw is the authored value, including nil when explicitly supplied.
		Raw any
		// ExplicitNull preserves DSL null provenance rather than implying absence.
		ExplicitNull bool
		// Origin describes the authored location for diagnostics, never cache identity.
		Origin string
	}

	// ValueSource identifies one supplied value without assigning target semantics.
	ValueSource struct{ data *valueSourceData }

	// ResolvedValue is an immutable semantic node. Accessors return copied
	// containers and preserve selected branches instead of guessing from wire data.
	ResolvedValue struct{ node *resolvedValueNode }

	// ResolvedField carries a declared member identity or, for an additional
	// object field, a zero Member with its authored Name.
	ResolvedField struct {
		// Member identifies the declared occurrence edge, or is zero for extras.
		Member ValueIdentity
		// Name is the authored member name, not a transport rename.
		Name string
		// Value retains absent and null states as distinct semantic nodes.
		Value ResolvedValue
	}

	// ResolvedEntry preserves typed map keys before target spelling and insertion.
	ResolvedEntry struct {
		// Key is a scalar semantic key.
		Key ResolvedValue
		// Value is the entry's semantic value.
		Value ResolvedValue
	}

	// ValueDiagnostic reports a source or projection failure without a null sentinel.
	ValueDiagnostic struct {
		// Code is a stable machine-readable failure category.
		Code string
		// Path is an ordered authored member/index path.
		Path []string
		// Message describes the violated contract.
		Message string
	}

	// ValueResult contains either a retained resolved graph or a failure outcome.
	// Incomplete examples retain their graph and exact missing member paths.
	ValueResult struct {
		outcome        ValueOutcome
		value          ResolvedValue
		diagnostics    []ValueDiagnostic
		missing        [][]ValueIdentity
		legacy         *valueSourceSnapshot
		role           ValueRole
		synthesized    bool
		source         *valueSourceData
		occurrence     ValueOccurrence
		customBoundary bool
	}

	valueSourceData struct {
		context      *valueContextIdentity
		snapshot     valueSourceSnapshot
		explicitNull bool
		origin       string
	}

	resolvedValueNode struct {
		context    *valueContextIdentity
		source     *valueSourceData
		occurrence ValueOccurrence
		presence   ValuePresence
		kind       ValueKind
		scalar     any
		elements   []ResolvedValue
		fields     []ResolvedField
		entries    []ResolvedEntry
		branch     ValueIdentity
		payload    ResolvedValue
		raw        any
		json       jsontext.Value
	}

	valueResolutionKey struct {
		occurrence *valueOccurrenceNode
		source     *valueSourceData
		role       ValueRole
	}
)

const (
	// ValueRoleExample allows partial authored documentation values.
	ValueRoleExample ValueRole = iota + 1
	// ValueRoleEnum requires a complete, valid enum member.
	ValueRoleEnum
	// ValueRoleDefault requires a complete, valid default.
	ValueRoleDefault
)

const (
	// ValueResolved is a successful complete semantic value.
	ValueResolved ValueOutcome = iota + 1
	// ValueIncomplete retains an example with missing required members.
	ValueIncomplete
	// ValueAmbiguous has multiple viable semantic alternatives.
	ValueAmbiguous
	// ValueInvalid violates a source contract.
	ValueInvalid
	// ValueUnsupported lies outside builtin semantic interpretation.
	ValueUnsupported
	// ValueSuppressed was excluded before resolution or synthesis.
	ValueSuppressed
)

const (
	// ValueAbsent marks an unsupplied member.
	ValueAbsent ValuePresence = iota
	// ValueNull marks an explicit null.
	ValueNull
	// ValueNil marks a typed nil collection, distinguished by Kind.
	ValueNil
	// ValuePresent marks a concrete value, including zero and empty containers.
	ValuePresent
)

const (
	// ValueKindScalar is a boolean, number, string or byte scalar.
	ValueKindScalar ValueKind = iota + 1
	// ValueKindArray is an ordered collection.
	ValueKindArray
	// ValueKindObject contains declared fields and named extras.
	ValueKindObject
	// ValueKindMap contains typed key-value entries.
	ValueKindMap
	// ValueKindUnion carries the already selected semantic branch.
	ValueKindUnion
	// ValueKindAny retains raw builtin meaning until codec materialization.
	ValueKindAny
)

// NewValueContext creates a generation-scoped semantic owner with no global cache.
func NewValueContext() *ValueContext {
	return &ValueContext{
		identity:    &valueContextIdentity{marker: 1},
		examples:    make(map[*ExampleExpr]*valueSourceData),
		resolved:    make(map[valueResolutionKey]ValueResult),
		synthesized: make(map[*valueOccurrenceNode]ValueResult),
		projected:   make(map[valueProjectionKey]*valueProjectionSlot),
	}
}

// SupplyValue captures source data without validating a semantic contract or
// invoking a custom codec. Capture errors remain deferred until Resolve.
func (c *ValueContext) SupplyValue(input ValueInput) ValueSource {
	return ValueSource{data: &valueSourceData{
		context:      c.identity,
		snapshot:     snapshotValueSource(input.Raw),
		explicitNull: input.ExplicitNull,
		origin:       input.Origin,
	}}
}

// ID returns this source's opaque identity, or zero for an absent handle.
func (s ValueSource) ID() ValueIdentity {
	if s.data == nil {
		return ValueIdentity{}
	}
	return ValueIdentity{context: s.data.context, source: s.data}
}

// ExplicitNull reports authored DSL null provenance without conflating absence.
func (s ValueSource) ExplicitNull() bool {
	return s.data != nil && s.data.explicitNull
}

// Origin returns the copied diagnostic source description, never a cache key.
func (s ValueSource) Origin() string {
	if s.data == nil {
		return ""
	}
	return s.data.origin
}

// ID returns this node's opaque identity, or zero for an absent handle.
func (v ResolvedValue) ID() ValueIdentity {
	if v.node == nil {
		return ValueIdentity{}
	}
	return ValueIdentity{context: v.node.context, value: v.node}
}

// SourceID identifies the source from which this node was resolved.
func (v ResolvedValue) SourceID() ValueIdentity {
	if v.node == nil {
		return ValueIdentity{}
	}
	return (ValueSource{data: v.node.source}).ID()
}

// OccurrenceID identifies the semantic occurrence, independently of its target.
func (v ResolvedValue) OccurrenceID() ValueIdentity {
	if v.node == nil {
		return ValueIdentity{}
	}
	return v.node.occurrence.ID()
}

// Presence returns the node's presence; the zero handle is absent.
func (v ResolvedValue) Presence() ValuePresence {
	if v.node == nil {
		return ValueAbsent
	}
	return v.node.presence
}

// Kind returns the semantic shape, or zero for a zero handle.
func (v ResolvedValue) Kind() ValueKind {
	if v.node == nil {
		return 0
	}
	return v.node.kind
}

// Scalar returns a scalar copy, preserving its concrete numeric or byte type.
func (v ResolvedValue) Scalar() (any, bool) {
	if v.node == nil || v.node.kind != ValueKindScalar || v.node.presence != ValuePresent {
		return nil, false
	}
	return snapshotValueSource(v.node.scalar).raw, true
}

// Elements returns copied handles in array order.
func (v ResolvedValue) Elements() []ResolvedValue {
	if v.node == nil {
		return nil
	}
	return append([]ResolvedValue(nil), v.node.elements...)
}

// Fields returns copied declared-field and additional-field records.
func (v ResolvedValue) Fields() []ResolvedField {
	if v.node == nil {
		return nil
	}
	return append([]ResolvedField(nil), v.node.fields...)
}

// Entries returns copied typed map entries.
func (v ResolvedValue) Entries() []ResolvedEntry {
	if v.node == nil {
		return nil
	}
	return append([]ResolvedEntry(nil), v.node.entries...)
}

// Union returns the selected occurrence, branch and payload without reselection.
func (v ResolvedValue) Union() (ValueIdentity, ValueIdentity, ResolvedValue, bool) {
	if v.node == nil || v.node.kind != ValueKindUnion {
		return ValueIdentity{}, ValueIdentity{}, ResolvedValue{}, false
	}
	return v.OccurrenceID(), v.node.branch, v.node.payload, true
}

// Any returns the retained builtin payload of an Any node.
func (v ResolvedValue) Any() (ResolvedValue, bool) {
	if v.node == nil || v.node.kind != ValueKindAny {
		return ResolvedValue{}, false
	}
	return v.node.payload, true
}

// Outcome returns the semantic outcome; zero denotes an uninitialized result.
func (r ValueResult) Outcome() ValueOutcome {
	return r.outcome
}

// Role returns the source completeness policy used for this resolution.
func (r ValueResult) Role() ValueRole {
	return r.role
}

// SourceID preserves source identity even when semantic resolution failed.
func (r ValueResult) SourceID() ValueIdentity {
	return (ValueSource{data: r.source}).ID()
}

// OccurrenceID identifies the effective occurrence used for resolution.
func (r ValueResult) OccurrenceID() ValueIdentity {
	return r.occurrence.ID()
}

// Synthesized distinguishes generated examples from supplied authored values.
func (r ValueResult) Synthesized() bool {
	return r.synthesized
}

// Value returns retained complete or incomplete semantic data.
func (r ValueResult) Value() (ResolvedValue, bool) {
	return r.value, r.outcome == ValueResolved || r.outcome == ValueIncomplete
}

// Diagnostics returns an owned copy of all diagnostic records and paths.
func (r ValueResult) Diagnostics() []ValueDiagnostic {
	result := append([]ValueDiagnostic(nil), r.diagnostics...)
	for index := range result {
		result[index].Path = append([]string(nil), result[index].Path...)
	}
	return result
}

// Missing returns copied ordered paths of required semantic member identities.
func (r ValueResult) Missing() [][]ValueIdentity {
	result := make([][]ValueIdentity, len(r.missing))
	for index, path := range r.missing {
		result[index] = append([]ValueIdentity(nil), path...)
	}
	return result
}

// LegacyValue returns copied raw data from the same source or synthesis attempt.
// Its presence does not assert semantic success; compatibility callers may still
// observe authored values rejected by the builtin pipeline.
func (r ValueResult) LegacyValue() (any, bool) {
	if r.legacy == nil {
		return nil, false
	}
	return snapshotValueSource(r.legacy.raw).raw, true
}

// RawAny returns an owned copy of the finite builtin host snapshot retained by
// an Any value. Concrete container and nested scalar types are preserved. This
// is source host meaning, not projected JSON or a fallback for other kinds; no
// custom materializer is invoked. Non-Any values return false.
func (v ResolvedValue) RawAny() (any, bool) {
	if v.node == nil || v.node.kind != ValueKindAny {
		return nil, false
	}
	return snapshotValueSource(v.node.raw).raw, true
}
