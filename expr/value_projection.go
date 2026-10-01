package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

type (
	// ProjectionOutcome describes whether one target can represent the retained
	// semantic value. Rejection never authorizes branch reselection or synthesis.
	ProjectionOutcome uint8
	// ProjectionResult owns canonical JSON bytes or a target-specific diagnostic.
	ProjectionResult struct {
		outcome     ProjectionOutcome
		wire        jsontext.Value
		diagnostics []ValueDiagnostic
		err         error
	}
	valueProjectionFailure struct {
		outcome ProjectionOutcome
		message string
	}
	valueProjectionKey struct {
		source      *valueSourceData
		occurrence  *valueOccurrenceNode
		role        ValueRole
		plan        *valuePlanNode
		association *valuePlanAssociation
	}
	valueProjectionSlot struct {
		done   chan struct{}
		result ProjectionResult
	}
)

const (
	// ProjectionEmitted contains one checked target representation.
	ProjectionEmitted ProjectionOutcome = iota + 1
	// ProjectionIncomplete lacks a retained required wire member.
	ProjectionIncomplete
	// ProjectionUnsupported requires a codec outside builtin JSON interpretation.
	ProjectionUnsupported
	// ProjectionInvalidPlan has incompatible ownership or target structure.
	ProjectionInvalidPlan
	// ProjectionUnrepresentable fails schema, decoding or semantic preservation.
	ProjectionUnrepresentable
	// ProjectionSuppressed does not attempt materialization or synthesis.
	ProjectionSuppressed
)

// ProjectJSON observes target visibility and presence before constructing one
// canonical wire. Schema checks and runtime decoding consume that same wire;
// neither may change the semantic union branch. Documentation plans make no
// runtime-decoder claim.
func (c *ValueContext) ProjectJSON(result ValueResult, plan ValuePlan) ProjectionResult {
	if c == nil || plan.context != c.identity || plan.root == nil {
		return projectionFailure(ProjectionInvalidPlan, "plan does not belong to the value context")
	}
	if plan.root.schemaOnly {
		return projectionFailure(ProjectionInvalidPlan, "structural schema plans do not authorize value projection")
	}
	if result.source == nil {
		return c.projectJSON(result, plan)
	}
	if result.source.context != c.identity || (result.occurrence.graph != plan.source.graph || result.occurrence.node != plan.source.node) {
		return projectionFailure(ProjectionInvalidPlan, "value and plan have different semantic owners")
	}
	key := valueProjectionKey{source: result.source, occurrence: result.occurrence.node, role: result.role,
		plan: plan.root, association: plan.association}
	c.mu.Lock()
	if prior := c.projected[key]; prior != nil {
		c.mu.Unlock()
		<-prior.done
		return prior.result
	}
	slot := &valueProjectionSlot{done: make(chan struct{})}
	c.projected[key] = slot
	c.mu.Unlock()
	// Always release concurrent readers, including when a user codec panics.
	// The panic remains the caller's panic; subsequent readers see the failure.
	slot.result = projectionFailure(ProjectionUnsupported, "custom materialization did not complete")
	defer close(slot.done)
	slot.result = c.projectJSON(result, plan)
	return slot.result
}

func (c *ValueContext) projectJSON(result ValueResult, plan ValuePlan) ProjectionResult {
	if result.outcome == ValueSuppressed {
		return ProjectionResult{outcome: ProjectionSuppressed}
	}
	if result.outcome == ValueUnsupported {
		if result.customBoundary {
			return projectCustomJSON(result, plan)
		}
		return projectionFailure(ProjectionUnsupported, "source requires an explicit custom materialization boundary")
	}
	value, available := result.Value()
	if !available {
		return projectionFailure(ProjectionUnrepresentable, "source has no unambiguous semantic value")
	}
	if value.node.context != c.identity || value.node.occurrence.graph != plan.source.graph {
		return projectionFailure(ProjectionInvalidPlan, "value and plan have different semantic owners")
	}
	if plan.root.codec != ValueCodecJSON {
		return projectionFailure(ProjectionUnsupported, "target does not use the builtin JSON codec")
	}
	for _, selected := range plan.selection {
		value = projectionMember(value, selected)
	}
	observed, failure := observeJSON(plan.root, result.role, value)
	if failure != nil {
		return projectionFailure(failure.outcome, failure.message)
	}
	wire, failure := constructJSON(plan.root, observed)
	if failure != nil {
		return projectionFailure(failure.outcome, failure.message)
	}
	if len(wire) == 0 {
		return projectionFailure(ProjectionIncomplete, "selected value is absent")
	}
	if !schemaJSON(plan.root, wire) {
		return projectionFailure(ProjectionUnrepresentable, "canonical JSON does not satisfy the target schema")
	}
	if !plan.root.documentary {
		decoded, accepted := decodeJSON(plan.root, wire)
		if !accepted || !projectionEqual(observed, decoded) {
			return projectionFailure(ProjectionUnrepresentable, "target decoding does not preserve the observed value")
		}
	}
	return ProjectionResult{outcome: ProjectionEmitted, wire: wire}
}

func projectCustomJSON(result ValueResult, plan ValuePlan) ProjectionResult {
	if plan.root.codec != ValueCodecJSON || len(plan.selection) != 0 {
		return projectionFailure(ProjectionUnsupported, "custom source requires a whole-value JSON observation")
	}
	effective := plan.root
	for effective.alias != nil {
		effective = effective.alias
	}
	if !plan.root.documentary && effective.kind != AnyKind {
		return projectionFailure(ProjectionUnsupported, "custom runtime value lacks an external observation contract")
	}
	wire, err := json.Marshal(result.source.snapshot.raw, json.Deterministic(true))
	if err != nil {
		failure := projectionFailure(ProjectionUnrepresentable, "custom materialization failed: "+err.Error())
		if !result.synthesized {
			failure.err = fmt.Errorf("materialize authored custom value: %w", err)
		}
		return failure
	}
	wire, err = canonicalJSONSnapshot(wire)
	if err != nil {
		failure := projectionFailure(ProjectionUnrepresentable, "custom materialization produced invalid JSON")
		if !result.synthesized {
			failure.err = fmt.Errorf("snapshot authored custom value: %w", err)
		}
		return failure
	}
	if !schemaJSON(plan.root, wire) {
		return projectionFailure(ProjectionUnrepresentable, "custom snapshot does not satisfy target schema")
	}
	if !plan.root.documentary {
		decoded, valid := decodeJSON(plan.root, wire)
		observed := ResolvedValue{node: &resolvedValueNode{presence: ValuePresent, kind: ValueKindAny, json: wire}}
		if !valid || !projectionEqual(observed, decoded) {
			return projectionFailure(ProjectionUnrepresentable, "runtime Any decoding changed the external JSON observation")
		}
	}
	return ProjectionResult{outcome: ProjectionEmitted, wire: wire}
}

func canonicalJSONSnapshot(wire jsontext.Value) (jsontext.Value, error) {
	if !wire.IsValid() {
		return nil, fmt.Errorf("invalid JSON snapshot")
	}
	switch wire.Kind() {
	case '{':
		var fields map[string]jsontext.Value
		if err := json.Unmarshal(wire, &fields); err != nil {
			return nil, err
		}
		for name, value := range fields {
			copy, err := canonicalJSONSnapshot(value)
			if err != nil {
				return nil, err
			}
			fields[name] = copy
		}
		return json.Marshal(fields, json.Deterministic(true))
	case '[':
		var items []jsontext.Value
		if err := json.Unmarshal(wire, &items); err != nil {
			return nil, err
		}
		for index, value := range items {
			copy, err := canonicalJSONSnapshot(value)
			if err != nil {
				return nil, err
			}
			items[index] = copy
		}
		return json.Marshal(items, json.Deterministic(true))
	default:
		return append(jsontext.Value(nil), wire...), nil
	}
}

// Outcome returns the target projection result.
func (r ProjectionResult) Outcome() ProjectionOutcome {
	return r.outcome
}

// JSON returns copied canonical bytes only when the target emitted a value.
func (r ProjectionResult) JSON() ([]byte, bool) {
	return append([]byte(nil), r.wire...), r.outcome == ProjectionEmitted
}

// Diagnostics returns owned diagnostic records and paths.
func (r ProjectionResult) Diagnostics() []ValueDiagnostic {
	return (ValueResult{diagnostics: r.diagnostics}).Diagnostics()
}

// Err reports an authored custom materialization error when one occurred.
func (r ProjectionResult) Err() error {
	return r.err
}

func projectionFailure(outcome ProjectionOutcome, message string) ProjectionResult {
	return ProjectionResult{outcome: outcome, diagnostics: []ValueDiagnostic{{Code: "projection", Message: message}}}
}

func projectionBuildFailure(outcome ProjectionOutcome, format string, args ...any) *valueProjectionFailure {
	return &valueProjectionFailure{outcome: outcome, message: fmt.Sprintf(format, args...)}
}

func projectionMember(value ResolvedValue, identity uint64) ResolvedValue {
	if value.node != nil {
		for _, field := range value.node.fields {
			if field.Member.index == identity {
				return field.Value
			}
		}
	}
	return ResolvedValue{}
}
