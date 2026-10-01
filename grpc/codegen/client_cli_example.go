package codegen

import (
	"errors"
	"fmt"
	"math"
	"strings"

	codegenpkg "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

type (
	protoJSONExampleProjector struct {
		context *codegenpkg.Context
		service string
		method  string
		payload *expr.AttributeExpr
		message *expr.AttributeExpr
	}

	protoJSONUnavailableError struct {
		message string
	}
)

func (e *protoJSONUnavailableError) Error() string {
	return e.message
}

// protoJSONExample projects the retained service payload example into the
// protocol buffer JSON mapping of message. It never selects a source,
// synthesizes a value or chooses a union branch.
func protoJSONExample(
	ctx *codegenpkg.Context,
	value *service.ValueData,
	payload, message *expr.AttributeExpr,
	serviceName, methodName string,
) any {
	p := protoJSONExampleProjector{
		context: ctx,
		service: serviceName,
		method:  methodName,
		payload: payload,
		message: message,
	}
	return p.project(value)
}

func (p protoJSONExampleProjector) project(data *service.ValueData) any {
	if data == nil || data.Context == nil {
		panic(codegenpkg.NewError(p.context, p.payload, errors.New("build gRPC CLI message example: missing retained source owner")))
	}
	result := data.Example
	switch result.Outcome() {
	case expr.ValueInvalid:
		if !result.Synthesized() {
			panic(codegenpkg.NewError(p.context, p.payload,
				protoJSONExampleError("invalid authored gRPC CLI message example", result.Diagnostics())))
		}
		p.warnOmitted("invalid synthesized source", result.Diagnostics(), nil)
		return nil
	case expr.ValueIncomplete, expr.ValueAmbiguous, expr.ValueUnsupported:
		p.warnOmitted("source is not usable", result.Diagnostics(), nil)
		return nil
	case expr.ValueSuppressed:
		return nil
	case expr.ValueResolved:
	default:
		panic(codegenpkg.NewError(p.context, p.payload,
			fmt.Errorf("build gRPC CLI message example: invalid retained source outcome %d", result.Outcome())))
	}
	resolved, ok := result.Value()
	if !ok {
		panic(codegenpkg.NewError(p.context, p.payload, errors.New("build gRPC CLI message example: resolved source has no value")))
	}
	projected, err := projectProtoJSONValue(data.Occurrence, p.payload, p.message, resolved)
	if err == nil {
		return projected
	}
	var unavailable *protoJSONUnavailableError
	if errors.As(err, &unavailable) {
		p.warnOmitted("protobuf projection is not usable", nil, unavailable)
		return nil
	}
	panic(codegenpkg.NewError(p.context, p.payload, fmt.Errorf("build gRPC CLI message example: %w", err)))
}

func (p protoJSONExampleProjector) warnOmitted(reason string, diagnostics []expr.ValueDiagnostic, err error) {
	args := []any{
		"service", p.service,
		"method", p.method,
		"reason", reason,
	}
	if len(diagnostics) > 0 {
		args = append(args, "diagnostics", diagnostics)
	}
	if err != nil {
		args = append(args, "error", err)
	}
	p.context.Warn("omitting unusable gRPC CLI message example", args...)
}

func protoJSONExampleError(prefix string, diagnostics []expr.ValueDiagnostic) error {
	if len(diagnostics) == 0 {
		return errors.New(prefix)
	}
	parts := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		path := strings.Join(diagnostic.Path, ".")
		if path == "" {
			parts = append(parts, diagnostic.Message)
			continue
		}
		parts = append(parts, path+": "+diagnostic.Message)
	}
	return fmt.Errorf("%s: %s", prefix, strings.Join(parts, "; "))
}

func projectProtoJSONValue(
	occurrence expr.ValueOccurrence,
	source, target *expr.AttributeExpr,
	value expr.ResolvedValue,
) (any, error) {
	if len(target.Meta["struct:field:proto"]) > 0 {
		return nil, &protoJSONUnavailableError{message: "opaque protobuf field mapping cannot project a retained service value"}
	}
	switch value.Presence() {
	case expr.ValueAbsent:
		return nil, errors.New("cannot project an absent retained value")
	case expr.ValueNull:
		return nil, &protoJSONUnavailableError{message: "explicit null has no protobuf message representation"}
	case expr.ValueNil:
		return nil, &protoJSONUnavailableError{message: "typed nil has no protobuf message representation"}
	case expr.ValuePresent:
	default:
		return nil, fmt.Errorf("invalid retained value presence %d", value.Presence())
	}

	if wrapper := protoJSONWrapperField(source, target, value.Kind()); wrapper != nil {
		names := newProtoMessageNames(target)
		if value.Kind() == expr.ValueKindUnion {
			name, projected, err := projectProtoJSONUnion(occurrence, source, wrapper.Attribute, value,
				names.oneofFields(wrapper.Name))
			if err != nil {
				return nil, err
			}
			return map[string]any{name: projected}, nil
		}
		projected, err := projectProtoJSONValue(occurrence, source, wrapper.Attribute, value)
		if err != nil {
			return nil, err
		}
		return map[string]any{names.field(wrapper.Name): projected}, nil
	}

	switch value.Kind() {
	case expr.ValueKindScalar:
		return projectProtoJSONScalar(source, value)
	case expr.ValueKindArray:
		return projectProtoJSONArray(occurrence, source, target, value)
	case expr.ValueKindObject:
		return projectProtoJSONObject(occurrence, source, target, value)
	case expr.ValueKindMap:
		return projectProtoJSONMap(occurrence, source, target, value)
	case expr.ValueKindUnion:
		return nil, errors.New("protobuf union value is missing its containing message allocation")
	case expr.ValueKindAny:
		raw, ok := value.RawAny()
		if !ok {
			return nil, errors.New("retained Any value has no raw value")
		}
		return raw, nil
	default:
		return nil, fmt.Errorf("unsupported retained value kind %d", value.Kind())
	}
}

func protoJSONWrapperField(source, target *expr.AttributeExpr, kind expr.ValueKind) *expr.NamedAttributeExpr {
	if kind == expr.ValueKindObject {
		return nil
	}
	obj := expr.AsObject(target.Type)
	if obj == nil || len(*obj) != 1 {
		return nil
	}
	wrapper := (*obj)[0]
	if codegenpkg.IsCompatible(source.Type, wrapper.Attribute.Type, "retained source", "protobuf wrapper") != nil {
		return nil
	}
	return wrapper
}

func projectProtoJSONObject(
	occurrence expr.ValueOccurrence,
	source, target *expr.AttributeExpr,
	value expr.ResolvedValue,
) (any, error) {
	if expr.AsObject(source.Type) == nil || expr.AsObject(target.Type) == nil {
		return nil, errors.New("retained object does not match its protobuf message")
	}
	members := occurrence.Members()
	membersByID := make(map[expr.ValueIdentity]expr.ValueMember, len(members))
	for _, member := range members {
		membersByID[member.ID] = member
	}
	names := newProtoMessageNames(target)
	projected := make(map[string]any)
	for _, field := range value.Fields() {
		if field.Value.Presence() == expr.ValueAbsent {
			continue
		}
		member, ok := membersByID[field.Member]
		if !ok {
			if field.Member == (expr.ValueIdentity{}) {
				return nil, &protoJSONUnavailableError{message: fmt.Sprintf("additional field %q has no protobuf field", field.Name)}
			}
			return nil, fmt.Errorf("retained object field %q has no occurrence member", field.Name)
		}
		_, sourceField := source.FindAttribute(member.Name)
		if sourceField == nil {
			return nil, fmt.Errorf("retained object member %q is absent from its service payload", member.Name)
		}
		targetName, targetField := target.FindAttribute(member.Name)
		if targetField == nil {
			continue
		}
		if field.Value.Kind() == expr.ValueKindUnion {
			branchName, branchValue, err := projectProtoJSONUnion(
				member.Occurrence, sourceField, targetField, field.Value, names.oneofFields(targetName),
			)
			if err != nil {
				return nil, err
			}
			projected[branchName] = branchValue
			continue
		}
		fieldValue, err := projectProtoJSONValue(member.Occurrence, sourceField, targetField, field.Value)
		if err != nil {
			return nil, err
		}
		projected[names.field(targetName)] = fieldValue
	}
	return projected, nil
}

func projectProtoJSONUnion(
	occurrence expr.ValueOccurrence,
	source, target *expr.AttributeExpr,
	value expr.ResolvedValue,
	fieldNames []string,
) (string, any, error) {
	_, selectedID, payload, selected := value.Union()
	if !selected {
		return "", nil, errors.New("retained union has no selected branch")
	}
	branches := occurrence.Branches()
	var sourceBranch expr.ValueBranch
	found := false
	for _, branch := range branches {
		if branch.ID == selectedID {
			sourceBranch = branch
			found = true
			break
		}
	}
	if !found {
		return "", nil, errors.New("retained union branch is not owned by its occurrence")
	}
	sourceUnion := expr.AsUnion(source.Type)
	targetUnion := expr.AsUnion(target.Type)
	if sourceUnion == nil || targetUnion == nil {
		return "", nil, errors.New("retained union does not match its protobuf oneof")
	}
	var sourceAttribute *expr.AttributeExpr
	for _, branch := range sourceUnion.Values {
		if expr.AttributeName(branch.Name) == expr.AttributeName(sourceBranch.Name) {
			sourceAttribute = branch.Attribute
			break
		}
	}
	if sourceAttribute == nil {
		return "", nil, fmt.Errorf("retained union branch %q is absent from its service type", sourceBranch.Name)
	}
	for index, branch := range targetUnion.Values {
		if expr.AttributeName(branch.Name) != expr.AttributeName(sourceBranch.Name) {
			continue
		}
		if index >= len(fieldNames) {
			return "", nil, fmt.Errorf("protobuf oneof branch %q has no allocated field name", branch.Name)
		}
		projected, err := projectProtoJSONValue(sourceBranch.Occurrence, sourceAttribute, branch.Attribute, payload)
		return fieldNames[index], projected, err
	}
	return "", nil, fmt.Errorf("retained union branch %q is absent from its protobuf oneof", sourceBranch.Name)
}

func projectProtoJSONArray(
	occurrence expr.ValueOccurrence,
	source, target *expr.AttributeExpr,
	value expr.ResolvedValue,
) (any, error) {
	sourceArray := expr.AsArray(source.Type)
	targetArray := expr.AsArray(target.Type)
	if sourceArray == nil || targetArray == nil {
		return nil, errors.New("retained array does not match its protobuf repeated field")
	}
	elementOccurrence := occurrence.Element()
	elements := value.Elements()
	projected := make([]any, len(elements))
	for index, element := range elements {
		item, err := projectProtoJSONValue(elementOccurrence, sourceArray.ElemType, targetArray.ElemType, element)
		if err != nil {
			return nil, err
		}
		projected[index] = item
	}
	return projected, nil
}

func projectProtoJSONMap(
	occurrence expr.ValueOccurrence,
	source, target *expr.AttributeExpr,
	value expr.ResolvedValue,
) (any, error) {
	sourceMap := expr.AsMap(source.Type)
	targetMap := expr.AsMap(target.Type)
	if sourceMap == nil || targetMap == nil {
		return nil, errors.New("retained map does not match its protobuf map field")
	}
	elementOccurrence := occurrence.Element()
	projected := make(map[string]any, len(value.Entries()))
	for _, entry := range value.Entries() {
		key, ok := entry.Key.Scalar()
		if !ok {
			return nil, errors.New("retained map entry has a non-scalar key")
		}
		item, err := projectProtoJSONValue(elementOccurrence, sourceMap.ElemType, targetMap.ElemType, entry.Value)
		if err != nil {
			return nil, err
		}
		projected[protoJSONMapKey(key)] = item
	}
	return projected, nil
}

func projectProtoJSONScalar(source *expr.AttributeExpr, value expr.ResolvedValue) (any, error) {
	scalar, ok := value.Scalar()
	if !ok {
		return nil, errors.New("retained scalar has no scalar value")
	}
	switch protoJSONSourceKind(source) {
	case expr.IntKind, expr.Int64Kind, expr.UIntKind, expr.UInt64Kind:
		return fmt.Sprint(scalar), nil
	case expr.Float32Kind:
		if number, ok := scalar.(float32); ok {
			return protoJSONFloat(float64(number), scalar), nil
		}
	case expr.Float64Kind:
		if number, ok := scalar.(float64); ok {
			return protoJSONFloat(number, scalar), nil
		}
	}
	return scalar, nil
}

func protoJSONSourceKind(source *expr.AttributeExpr) expr.Kind {
	seen := make(map[string]struct{})
	for {
		userType, ok := source.Type.(expr.UserType)
		if !ok {
			return source.Type.Kind()
		}
		if _, found := seen[userType.ID()]; found {
			return source.Type.Kind()
		}
		seen[userType.ID()] = struct{}{}
		source = userType.Attribute()
	}
}

func protoJSONFloat(number float64, original any) any {
	switch {
	case math.IsNaN(number):
		return "NaN"
	case math.IsInf(number, 1):
		return "Infinity"
	case math.IsInf(number, -1):
		return "-Infinity"
	default:
		return original
	}
}

// protoJSONMapKey returns the JSON object key of the map key example key.
// The protocol buffer JSON mapping encodes integer and boolean keys as their
// decimal and literal text.
func protoJSONMapKey(key any) string {
	if text, ok := key.(string); ok {
		return text
	}
	return fmt.Sprint(key)
}
