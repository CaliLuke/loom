package valuecontract

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type (
	independentResolvedValue struct {
		presence string
		kind     string
		scalar   any
		opaque   any
		elements []independentResolvedValue
		fields   []independentResolvedField
		entries  []independentResolvedEntry
	}
	independentResolvedField struct {
		identity int
		name     string
		value    independentResolvedValue
	}
	independentResolvedEntry struct {
		key   independentResolvedValue
		value independentResolvedValue
	}
)

func TestRequiredObservationComparisonRejectsOrderAndDuplicates(t *testing.T) {
	want := []requiredObservation{{field: 2, origin: 2}, {field: 1, origin: 2}}
	for _, candidate := range []struct {
		name  string
		value []requiredObservation
	}{
		{"swapped", []requiredObservation{{field: 1, origin: 2}, {field: 2, origin: 2}}},
		{"duplicate", []requiredObservation{{field: 2, origin: 2}, {field: 1, origin: 2}, {field: 1, origin: 1}}},
		{"ancestor provenance", []requiredObservation{{field: 2, origin: 2}, {field: 1, origin: 1}}},
	} {
		t.Run(candidate.name, func(t *testing.T) {
			require.False(t, slices.Equal(want, candidate.value))
		})
	}
}

func TestIndependentObjectNormalizationRejectsAmbiguousInputs(t *testing.T) {
	for _, test := range []struct {
		name      string
		attribute *expr.AttributeExpr
		value     any
	}{
		{
			name: "invalid authored spelling beside valid wire spelling",
			attribute: &expr.AttributeExpr{Type: &expr.Object{
				{Name: "blob:b", Attribute: &expr.AttributeExpr{Type: expr.String}},
			}},
			value: map[string]any{"blob:b": 1, "b": "valid"},
		},
		{
			name: "duplicate wire alias",
			attribute: &expr.AttributeExpr{Type: &expr.Object{
				{Name: "first:value", Attribute: &expr.AttributeExpr{Type: expr.String}},
				{Name: "second:value", Attribute: &expr.AttributeExpr{Type: expr.String}},
			}},
			value: map[string]any{"value": "ambiguous"},
		},
		{
			name: "closed object extra field",
			attribute: &expr.AttributeExpr{
				Type: &expr.Object{
					{Name: "known", Attribute: &expr.AttributeExpr{Type: expr.String}},
				},
				Meta: expr.MetaExpr{"openapi:additionalProperties": {"false"}},
			},
			value: map[string]any{"known": "value", "extra": true},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := independentResolve(test.attribute, test.value)
			require.Error(t, err)
		})
	}
}

func independentResolve(attribute *expr.AttributeExpr, raw any) (independentResolvedValue, error) {
	snapshot, err := independentSnapshot(raw)
	if err != nil {
		return independentResolvedValue{}, err
	}
	return independentResolveSnapshot(attribute, snapshot)
}

func independentResolveSnapshot(attribute *expr.AttributeExpr, raw any) (independentResolvedValue, error) {
	input := reflect.ValueOf(raw)
	if raw == nil || (input.Kind() == reflect.Pointer && input.IsNil()) {
		if !attribute.Nullable {
			return independentResolvedValue{}, fmt.Errorf("null is not admitted")
		}
		return independentResolvedValue{presence: "null"}, nil
	}
	attribute, err := independentUnderlyingAttribute(attribute)
	if err != nil {
		return independentResolvedValue{}, err
	}
	if primitive, ok := attribute.Type.(expr.Primitive); ok && primitive == expr.Any &&
		(input.Kind() == reflect.Struct || input.Kind() == reflect.Pointer) {
		return independentResolvedValue{presence: "present", kind: "opaque", opaque: raw}, nil
	}
	if independentHasCustomCodec(input.Type()) {
		return independentResolvedValue{presence: "present", kind: "opaque", opaque: raw}, nil
	}
	for input.Kind() == reflect.Interface || input.Kind() == reflect.Pointer {
		if input.IsNil() {
			return independentResolvedValue{}, fmt.Errorf("null is not admitted")
		}
		input = input.Elem()
	}
	raw = input.Interface()
	if _, object := attribute.Type.(*expr.Object); object && input.Kind() == reflect.Struct {
		raw = independentPlainStructMap(input)
	}
	switch actual := attribute.Type.(type) {
	case expr.Primitive:
		return independentResolvePrimitive(actual, raw)
	case *expr.Array:
		return independentResolveArray(actual, raw)
	case *expr.Object:
		return independentResolveObject(attribute, actual, raw)
	case *expr.Map:
		return independentResolveMap(actual, raw)
	default:
		return independentResolvedValue{}, fmt.Errorf("independent normalizer does not model %T", attribute.Type)
	}
}

func independentUnderlyingAttribute(attribute *expr.AttributeExpr) (*expr.AttributeExpr, error) {
	seen := make(map[expr.UserType]bool)
	for {
		named, ok := attribute.Type.(expr.UserType)
		if !ok {
			return attribute, nil
		}
		if seen[named] {
			return nil, fmt.Errorf("cyclic named type %q", named.Name())
		}
		seen[named] = true
		attribute = named.Attribute()
	}
}

func independentResolvePrimitive(primitive expr.Primitive, raw any) (independentResolvedValue, error) {
	input := reflect.ValueOf(raw)
	value := independentResolvedValue{presence: "present", kind: "scalar"}
	switch primitive {
	case expr.Boolean:
		if input.Kind() == reflect.Bool {
			value.scalar = input.Bool()
		}
	case expr.String:
		if input.Kind() == reflect.String {
			value.scalar = input.String()
		}
	case expr.Bytes:
		if input.Kind() == reflect.String {
			value.scalar = []byte(input.String())
		} else if input.Kind() == reflect.Slice && input.Type().Elem().Kind() == reflect.Uint8 {
			bytes := make([]byte, input.Len())
			reflect.Copy(reflect.ValueOf(bytes), input)
			value.scalar = bytes
		}
	case expr.Int, expr.Int32, expr.Int64, expr.UInt, expr.UInt32, expr.UInt64:
		if independentIntegerAdmitted(primitive, input) {
			value.scalar = raw
		}
	case expr.Float32, expr.Float64:
		if independentFloatAdmitted(primitive, input) {
			bits := 64
			if primitive == expr.Float32 {
				bits = 32
			}
			number, err := strconv.ParseFloat(fmt.Sprint(raw), bits)
			if err == nil && !math.IsInf(number, 0) && !math.IsNaN(number) {
				if bits == 32 {
					value.scalar = float32(number)
				} else {
					value.scalar = number
				}
			}
		}
	case expr.Any:
		if independentBuiltinValue(raw, make(map[independentVisit]bool)) {
			value.kind = "any"
			value.scalar = raw
		}
	}
	if value.scalar == nil {
		return independentResolvedValue{}, fmt.Errorf("%T is incompatible with %s", raw, primitive.Name())
	}
	return value, nil
}

func independentIntegerAdmitted(primitive expr.Primitive, input reflect.Value) bool {
	switch input.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return true
	case reflect.Int64, reflect.Uint64:
		return primitive == expr.Int64 || primitive == expr.UInt64
	default:
		return false
	}
}

func independentFloatAdmitted(primitive expr.Primitive, input reflect.Value) bool {
	switch input.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Int64, reflect.Uint64:
		return primitive == expr.Float32 || primitive == expr.Float64
	default:
		return false
	}
}

func independentResolveArray(array *expr.Array, raw any) (independentResolvedValue, error) {
	input := reflect.ValueOf(raw)
	if input.Kind() != reflect.Array && input.Kind() != reflect.Slice {
		return independentResolvedValue{}, fmt.Errorf("%T is not an array", raw)
	}
	presence := "present"
	if input.Kind() == reflect.Slice && input.IsNil() {
		presence = "nil"
	}
	value := independentResolvedValue{presence: presence, kind: "array"}
	for index := range input.Len() {
		child, err := independentResolve(array.ElemType, input.Index(index).Interface())
		if err != nil {
			return independentResolvedValue{}, fmt.Errorf("element %d: %w", index, err)
		}
		value.elements = append(value.elements, child)
	}
	return value, nil
}

func independentResolveObject(attribute *expr.AttributeExpr, object *expr.Object, raw any) (independentResolvedValue, error) {
	entries, ok := independentStringMap(raw)
	if !ok || entries == nil {
		return independentResolvedValue{}, fmt.Errorf("%T is not a non-nil object", raw)
	}
	value := independentResolvedValue{presence: "present", kind: "object"}
	known := make(map[string]bool)
	aliases := make(map[string]bool)
	for _, member := range *object {
		wire := independentElementName(member.Name)
		if aliases[wire] {
			if _, supplied := entries[wire]; supplied {
				return independentResolvedValue{}, fmt.Errorf("wire name %q identifies multiple fields", wire)
			}
		}
		aliases[wire] = true
	}
	for index, member := range *object {
		name := member.Name
		wire := independentElementName(member.Name)
		known[name], known[wire] = true, true
		var authored, wired *independentResolvedValue
		if rawAuthored, present := entries[name]; present {
			child, err := independentResolve(member.Attribute, rawAuthored)
			if err != nil {
				return independentResolvedValue{}, fmt.Errorf("field %q: %w", name, err)
			}
			authored = &child
		}
		if wire != name {
			if rawWired, present := entries[wire]; present {
				child, err := independentResolve(member.Attribute, rawWired)
				if err != nil {
					return independentResolvedValue{}, fmt.Errorf("field %q: %w", wire, err)
				}
				wired = &child
			}
		}
		child := independentResolvedValue{presence: "absent"}
		if authored != nil {
			child = *authored
		}
		if wired != nil {
			child = *wired
		}
		if child.presence == "absent" && independentObjectFieldRequired(attribute, name, wire) {
			return independentResolvedValue{}, fmt.Errorf("required field %q is absent", name)
		}
		value.fields = append(value.fields, independentResolvedField{identity: index + 1, name: name, value: child})
	}
	var extraNames []string
	for name := range entries {
		if !known[name] {
			extraNames = append(extraNames, name)
		}
	}
	slices.Sort(extraNames)
	if len(extraNames) > 0 && independentObjectClosed(attribute) {
		return independentResolvedValue{}, fmt.Errorf("undeclared object field %q", extraNames[0])
	}
	for _, name := range extraNames {
		child, err := independentResolvePrimitive(expr.Any, entries[name])
		if err != nil {
			return independentResolvedValue{}, fmt.Errorf("field %q: %w", name, err)
		}
		value.fields = append(value.fields, independentResolvedField{name: name, value: child})
	}
	return value, nil
}

func independentObjectFieldRequired(attribute *expr.AttributeExpr, name, wire string) bool {
	if attribute.Validation == nil {
		return false
	}
	return slices.Contains(attribute.Validation.Required, name) || slices.Contains(attribute.Validation.Required, wire)
}

func independentObjectClosed(attribute *expr.AttributeExpr) bool {
	values, present := attribute.Meta["openapi:additionalProperties"]
	return present && len(values) > 0 && values[len(values)-1] == "false"
}

func independentResolveMap(mapping *expr.Map, raw any) (independentResolvedValue, error) {
	input := reflect.ValueOf(raw)
	if input.Kind() != reflect.Map {
		return independentResolvedValue{}, fmt.Errorf("%T is not a map", raw)
	}
	presence := "present"
	if input.IsNil() {
		presence = "nil"
	}
	value := independentResolvedValue{presence: presence, kind: "map"}
	iterator := input.MapRange()
	for iterator.Next() {
		key, err := independentResolve(mapping.KeyType, iterator.Key().Interface())
		if err != nil {
			return independentResolvedValue{}, fmt.Errorf("map key: %w", err)
		}
		child, err := independentResolve(mapping.ElemType, iterator.Value().Interface())
		if err != nil {
			return independentResolvedValue{}, fmt.Errorf("map value: %w", err)
		}
		value.entries = append(value.entries, independentResolvedEntry{key: key, value: child})
	}
	return value, nil
}

func independentResolvedEqual(left, right independentResolvedValue) bool {
	if left.presence != right.presence {
		compatibleNil := (left.presence == "nil" || left.presence == "present") &&
			(right.presence == "nil" || right.presence == "present")
		if !compatibleNil {
			return false
		}
	}
	if left.kind != right.kind {
		return false
	}
	if left.presence == "absent" || left.presence == "null" {
		return left.presence == right.presence
	}
	switch left.kind {
	case "scalar", "any":
		return independentExampleEqual(left.scalar, right.scalar)
	case "opaque":
		return reflect.TypeOf(left.opaque) == reflect.TypeOf(right.opaque) &&
			reflect.DeepEqual(left.opaque, right.opaque)
	case "array":
		return slices.EqualFunc(left.elements, right.elements, independentResolvedEqual)
	case "object":
		return independentFieldsEqual(left.fields, right.fields)
	case "map":
		return independentEntriesEqual(left.entries, right.entries)
	default:
		return true
	}
}

func independentFieldsEqual(left, right []independentResolvedField) bool {
	if len(left) != len(right) {
		return false
	}
	matched := make([]bool, len(right))
	for _, field := range left {
		found := false
		for index, other := range right {
			if !matched[index] && field.identity == other.identity && field.name == other.name &&
				independentResolvedEqual(field.value, other.value) {
				matched[index], found = true, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func independentEntriesEqual(left, right []independentResolvedEntry) bool {
	if len(left) != len(right) {
		return false
	}
	matched := make([]bool, len(right))
	for _, entry := range left {
		found := false
		for index, other := range right {
			if !matched[index] && independentResolvedEqual(entry.key, other.key) &&
				independentResolvedEqual(entry.value, other.value) {
				matched[index], found = true, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func independentDecimal(value independentResolvedValue) *referenceDecimal {
	if value.kind != "scalar" || value.presence == "null" || value.presence == "absent" {
		return nil
	}
	switch number := value.scalar.(type) {
	case int:
		return decimalFromInteger(int64(number))
	case int32:
		return decimalFromInteger(int64(number))
	case int64:
		return decimalFromInteger(number)
	case uint:
		return decimalFromUnsigned(uint64(number))
	case uint32:
		return decimalFromUnsigned(uint64(number))
	case uint64:
		return decimalFromUnsigned(number)
	case float32:
		decimal := referenceBinaryDecimal(float64(number))
		return &decimal
	case float64:
		decimal := referenceBinaryDecimal(number)
		return &decimal
	default:
		return nil
	}
}

type independentVisit struct {
	typeOf  reflect.Type
	pointer uintptr
}

func independentBuiltinValue(raw any, active map[independentVisit]bool) bool {
	value := reflect.ValueOf(raw)
	if !value.IsValid() {
		return true
	}
	for value.Kind() == reflect.Interface {
		if value.IsNil() {
			return true
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Float32, reflect.Float64:
		return !math.IsInf(value.Float(), 0) && !math.IsNaN(value.Float())
	case reflect.Array:
		for index := range value.Len() {
			if !independentBuiltinValue(value.Index(index).Interface(), active) {
				return false
			}
		}
		return true
	case reflect.Slice:
		if value.IsNil() {
			return true
		}
		visit := independentVisit{typeOf: value.Type(), pointer: value.Pointer()}
		if active[visit] {
			return false
		}
		active[visit] = true
		defer delete(active, visit)
		for index := range value.Len() {
			if !independentBuiltinValue(value.Index(index).Interface(), active) {
				return false
			}
		}
		return true
	case reflect.Map:
		if value.IsNil() {
			return true
		}
		visit := independentVisit{typeOf: value.Type(), pointer: value.Pointer()}
		if active[visit] {
			return false
		}
		active[visit] = true
		defer delete(active, visit)
		iterator := value.MapRange()
		for iterator.Next() {
			if !independentBuiltinValue(iterator.Key().Interface(), active) ||
				!independentBuiltinValue(iterator.Value().Interface(), active) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func independentExampleEqual(left, right any) bool {
	if reflect.DeepEqual(left, right) {
		return true
	}
	leftNil, rightNil := independentNil(left), independentNil(right)
	if leftNil || rightNil {
		return leftNil && rightNil
	}
	if leftNumber, ok := independentNumber(left); ok {
		rightNumber, rightOK := independentNumber(right)
		return rightOK && leftNumber.Cmp(rightNumber) == 0
	}
	if leftMap, ok := independentStringMap(left); ok {
		rightMap, rightOK := independentStringMap(right)
		if !rightOK || len(leftMap) != len(rightMap) {
			return false
		}
		for key, leftValue := range leftMap {
			rightValue, present := rightMap[key]
			if !present || !independentExampleEqual(leftValue, rightValue) {
				return false
			}
		}
		return true
	}
	leftSlice, leftOK := independentSlice(left)
	rightSlice, rightOK := independentSlice(right)
	return leftOK && rightOK && slices.EqualFunc(leftSlice, rightSlice, independentExampleEqual)
}

func independentNumber(value any) (*big.Rat, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() {
		return nil, false
	}
	number := new(big.Rat)
	switch actual.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return number.SetInt64(actual.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return number.SetUint64(actual.Uint()), true
	case reflect.Float32, reflect.Float64:
		if number.SetFloat64(actual.Float()) == nil {
			return nil, false
		}
		return number, true
	default:
		return nil, false
	}
}

func independentStringMap(value any) (map[string]any, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || actual.Kind() != reflect.Map {
		return nil, false
	}
	if actual.IsNil() {
		return nil, true
	}
	result := make(map[string]any, actual.Len())
	iterator := actual.MapRange()
	for iterator.Next() {
		key := iterator.Key()
		for key.IsValid() && key.Kind() == reflect.Interface {
			if key.IsNil() {
				return nil, false
			}
			key = key.Elem()
		}
		if !key.IsValid() || key.Kind() != reflect.String {
			return nil, false
		}
		result[key.String()] = iterator.Value().Interface()
	}
	return result, true
}

func independentSlice(value any) ([]any, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || actual.Kind() != reflect.Array && actual.Kind() != reflect.Slice {
		return nil, false
	}
	result := make([]any, actual.Len())
	for index := range actual.Len() {
		result[index] = actual.Index(index).Interface()
	}
	return result, true
}

func independentNil(value any) bool {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() {
		return true
	}
	switch actual.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return actual.IsNil()
	default:
		return false
	}
}
