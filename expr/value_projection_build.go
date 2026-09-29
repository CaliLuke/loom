package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"

	"github.com/CaliLuke/loom/internal/jsonkey"
)

func observeJSON(plan *valuePlanNode, role ValueRole, value ResolvedValue) (ResolvedValue, *valueProjectionFailure) {
	if value.Presence() == ValueAbsent {
		return ResolvedValue{}, nil
	}
	if value.Presence() == ValueNull && plan.nullable {
		return value, nil
	}
	if plan.alias != nil {
		return observeJSON(plan.alias, role, value)
	}
	copy := *value.node
	observed := ResolvedValue{node: &copy}
	if value.Presence() == ValueNil && (plan.kind == ArrayKind || plan.kind == MapKind) {
		if role == ValueRoleExample {
			return ResolvedValue{}, projectionBuildFailure(ProjectionIncomplete, "nil collection example has no concrete target value")
		}
		copy.presence = ValuePresent
		return observed, nil
	}
	switch plan.kind {
	case AnyKind:
		if value.Kind() != ValueKindAny {
			return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "Any target requires an Any source")
		}
		wire, err := json.Marshal(value.node.raw, json.Deterministic(true))
		if err != nil {
			return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "Any materialization: %v", err)
		}
		copy.json = wire
		return observed, nil
	case ArrayKind:
		return observeJSONArray(plan, role, value)
	case MapKind:
		return observeJSONMap(plan, role, value)
	case ObjectKind:
		return observeJSONObject(plan, role, value)
	case UnionKind:
		return observeJSONUnion(plan, role, value)
	default:
		if value.Kind() != ValueKindScalar || value.Presence() != ValuePresent {
			return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "scalar target requires a concrete scalar")
		}
	}
	return observed, nil
}

func observeJSONObject(plan *valuePlanNode, role ValueRole, value ResolvedValue) (ResolvedValue, *valueProjectionFailure) {
	if value.Kind() != ValueKindObject {
		return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "object target requires an object")
	}
	copy := *value.node
	copy.fields = nil
	for _, member := range plan.members {
		if !member.visible {
			continue
		}
		child, failure := observeJSON(member.node, role, projectionMember(value, member.source))
		if failure != nil {
			return ResolvedValue{}, failure
		}
		if member.presence == ValueFieldOmitEmpty && projectionEmpty(member.node, child) {
			child = ResolvedValue{}
		}
		if member.presence == ValueFieldImplicitDefault && child.Presence() == ValueAbsent {
			child = ResolvedValue{node: &resolvedValueNode{presence: ValuePresent, kind: ValueKindScalar, scalar: member.implicitDefault}}
		}
		copy.fields = append(copy.fields, ResolvedField{Member: ValueIdentity{context: value.node.context, graph: value.node.occurrence.graph, index: member.source}, Name: member.wire, Value: child})
	}
	if plan.preserveAdditional {
		for _, field := range value.node.fields {
			if field.Member != (ValueIdentity{}) {
				continue
			}
			wire, err := json.Marshal(field.Value.node.raw, json.Deterministic(true))
			if err != nil {
				return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "additional value materialization: %v", err)
			}
			extra := *field.Value.node
			extra.json = wire
			copy.fields = append(copy.fields, ResolvedField{Name: field.Name, Value: ResolvedValue{node: &extra}})
		}
	}
	return ResolvedValue{node: &copy}, nil
}

func projectionEmpty(plan *valuePlanNode, value ResolvedValue) bool {
	if value.Presence() != ValuePresent {
		return false
	}
	if plan.alias != nil {
		return projectionEmpty(plan.alias, value)
	}
	switch plan.kind {
	case StringKind:
		return value.node.scalar == ""
	case BytesKind:
		return reflect.ValueOf(value.node.scalar).Len() == 0
	case ArrayKind:
		return len(value.node.elements) == 0
	case MapKind:
		return len(value.node.entries) == 0
	case AnyKind:
		return string(value.node.json) == `""` || string(value.node.json) == "[]" || string(value.node.json) == "{}"
	case ObjectKind:
		for _, field := range value.node.fields {
			if field.Member == (ValueIdentity{}) {
				return false
			}
			if field.Value.Presence() != ValueAbsent && !projectionDefaultOmitted(plan, field) {
				return false
			}
		}
		return true
	case UnionKind:
		if plan.untagged {
			for _, branch := range plan.branches {
				if branch.source == value.node.branch.index {
					return projectionEmpty(branch.node, value.node.payload)
				}
			}
		}
	}
	return false
}

func projectionDefaultOmitted(plan *valuePlanNode, field ResolvedField) bool {
	for _, member := range plan.members {
		if member.source == field.Member.index && member.presence == ValueFieldImplicitDefault && field.Value.Kind() == ValueKindScalar {
			return projectionScalarCategory(member.implicitDefault) == projectionScalarCategory(field.Value.node.scalar) && exampleValuesEqual(member.implicitDefault, field.Value.node.scalar)
		}
	}
	return false
}

func constructJSON(plan *valuePlanNode, value ResolvedValue) (jsontext.Value, *valueProjectionFailure) {
	if value.Presence() == ValueAbsent {
		return nil, nil
	}
	if value.Presence() == ValueNull && plan.nullable {
		return jsontext.Value("null"), nil
	}
	if plan.alias != nil {
		return constructJSON(plan.alias, value)
	}
	switch plan.kind {
	case AnyKind:
		return append(jsontext.Value(nil), value.node.json...), nil
	case ArrayKind:
		return constructJSONArray(plan, value)
	case MapKind:
		return constructJSONMap(plan, value)
	case ObjectKind:
		return constructJSONObject(plan, value)
	case UnionKind:
		return constructJSONUnion(plan, value)
	default:
		return marshalProjection(value.node.scalar)
	}
}

func constructJSONObject(plan *valuePlanNode, value ResolvedValue) (jsontext.Value, *valueProjectionFailure) {
	fields := make(map[string]jsontext.Value)
	for _, member := range plan.members {
		if !member.visible {
			continue
		}
		child := projectionMember(value, member.source)
		var wire jsontext.Value
		if child.Presence() != ValueAbsent && !projectionDefaultOmitted(plan, ResolvedField{Member: ValueIdentity{index: member.source}, Value: child}) {
			var failure *valueProjectionFailure
			wire, failure = constructJSON(member.node, child)
			if failure != nil {
				return nil, failure
			}
		}
		if len(wire) == 0 {
			if member.required {
				return nil, projectionBuildFailure(ProjectionIncomplete, "required wire member %q is absent", member.wire)
			}
			continue
		}
		fields[member.wire] = wire
	}
	if plan.preserveAdditional {
		for _, field := range value.node.fields {
			if field.Member != (ValueIdentity{}) {
				continue
			}
			if _, collision := fields[field.Name]; collision {
				return nil, projectionBuildFailure(ProjectionUnrepresentable, "additional member collides with wire field %q", field.Name)
			}
			fields[field.Name] = field.Value.node.json
		}
	}
	return marshalProjection(fields)
}

func marshalProjection(value any) (jsontext.Value, *valueProjectionFailure) {
	wire, err := json.Marshal(value, json.Deterministic(true))
	if err != nil {
		return nil, projectionBuildFailure(ProjectionUnrepresentable, "JSON construction: %v", err)
	}
	return wire, nil
}

func projectionNull(value ResolvedValue) bool {
	return value.Presence() == ValueNull || (value.Kind() == ValueKindAny && value.node.payload.Presence() == ValueNull)
}

func projectionScalarCategory(raw any) reflect.Kind {
	if raw == nil {
		return reflect.Invalid
	}
	kind := reflect.TypeOf(raw).Kind()
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflect.Int
	case reflect.Float32, reflect.Float64:
		return reflect.Float64
	default:
		return kind
	}
}

func observeJSONArray(plan *valuePlanNode, role ValueRole, value ResolvedValue) (ResolvedValue, *valueProjectionFailure) {
	copy := *value.node
	observed := ResolvedValue{node: &copy}
	if value.Kind() != ValueKindArray {
		return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "array target requires an array")
	}
	copy.elements = nil
	for _, item := range value.node.elements {
		if plan.nonNullableElements && projectionNull(item) {
			return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "non-null array element is null")
		}
		child, failure := observeJSON(plan.element, role, item)
		if failure != nil {
			return ResolvedValue{}, failure
		}
		copy.elements = append(copy.elements, child)
	}
	return observed, nil
}

func observeJSONMap(plan *valuePlanNode, role ValueRole, value ResolvedValue) (ResolvedValue, *valueProjectionFailure) {
	copy := *value.node
	observed := ResolvedValue{node: &copy}
	if value.Kind() != ValueKindMap {
		return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "map target requires a map")
	}
	copy.entries = nil
	for _, entry := range value.node.entries {
		key := entry.Key
		keyPlan := plan.key
		for keyPlan.alias != nil {
			keyPlan = keyPlan.alias
		}
		if keyPlan.kind == AnyKind {
			name, valid := jsonkey.Name(reflect.ValueOf(key.node.scalar))
			if !valid {
				return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "map key has no JSON spelling")
			}
			copyKey := *key.node
			copyKey.scalar = name
			key = ResolvedValue{node: &copyKey}
		}
		child, failure := observeJSON(plan.element, role, entry.Value)
		if failure != nil {
			return ResolvedValue{}, failure
		}
		copy.entries = append(copy.entries, ResolvedEntry{Key: key, Value: child})
	}
	return observed, nil
}

func observeJSONUnion(plan *valuePlanNode, role ValueRole, value ResolvedValue) (ResolvedValue, *valueProjectionFailure) {
	copy := *value.node
	observed := ResolvedValue{node: &copy}
	if value.Kind() != ValueKindUnion {
		return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "union target requires a selected source branch")
	}
	if value.OccurrenceID().index != plan.source.id {
		return ResolvedValue{}, projectionBuildFailure(ProjectionInvalidPlan, "union occurrence identity changed")
	}
	for _, branch := range plan.branches {
		if branch.source == value.node.branch.index {
			child, failure := observeJSON(branch.node, role, value.node.payload)
			if failure != nil {
				return ResolvedValue{}, failure
			}
			copy.payload = child
			return observed, nil
		}
	}
	return ResolvedValue{}, projectionBuildFailure(ProjectionUnrepresentable, "selected source branch is not visible in target")
}

func constructJSONArray(plan *valuePlanNode, value ResolvedValue) (jsontext.Value, *valueProjectionFailure) {
	items := make([]jsontext.Value, 0, len(value.node.elements))
	for _, item := range value.node.elements {
		wire, failure := constructJSON(plan.element, item)
		if failure != nil {
			return nil, failure
		}
		if len(wire) == 0 {
			return nil, projectionBuildFailure(ProjectionIncomplete, "array element is absent")
		}
		items = append(items, wire)
	}
	return marshalProjection(items)
}

func constructJSONMap(plan *valuePlanNode, value ResolvedValue) (jsontext.Value, *valueProjectionFailure) {
	fields := make(map[string]jsontext.Value)
	for _, entry := range value.node.entries {
		name, valid := jsonkey.Name(reflect.ValueOf(entry.Key.node.scalar))
		if _, duplicate := fields[name]; !valid || duplicate {
			return nil, projectionBuildFailure(ProjectionUnrepresentable, "map key spelling collides")
		}
		wire, failure := constructJSON(plan.element, entry.Value)
		if failure != nil {
			return nil, failure
		}
		if len(wire) == 0 {
			return nil, projectionBuildFailure(ProjectionIncomplete, "map value is absent")
		}
		fields[name] = wire
	}
	return marshalProjection(fields)
}

func constructJSONUnion(plan *valuePlanNode, value ResolvedValue) (jsontext.Value, *valueProjectionFailure) {
	for _, branch := range plan.branches {
		if branch.source != value.node.branch.index {
			continue
		}
		wire, failure := constructJSON(branch.node, value.node.payload)
		if failure != nil {
			return nil, failure
		}
		if len(wire) == 0 {
			return nil, projectionBuildFailure(ProjectionIncomplete, "union payload is absent")
		}
		if plan.untagged {
			return wire, nil
		}
		return marshalProjection(map[string]any{plan.typeKey: branch.tag, plan.valueKey: wire})
	}
	return nil, projectionBuildFailure(ProjectionUnrepresentable, "selected branch is absent from plan")
}
