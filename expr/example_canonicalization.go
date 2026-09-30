package expr

import (
	"bytes"
	"math/big"
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/internal/examplevalue"
	"github.com/CaliLuke/loom/internal/jsonkey"
)

// CanonicalizeExample normalizes example values to their canonical JSON shape:
// objects use the JSON field names of the HTTP and JSON-RPC bodies, which name
// a field declared as "n:m" after its element name "m", Loom unions use their
// discriminator/value shape, and map keys become JSON object member names, so
// that the key true becomes "true" and the key 12 becomes "12".
func CanonicalizeExample(att *AttributeExpr, example any) any {
	if att == nil || att.Type == nil || att.Type == Empty {
		return example
	}
	if nilExample(example) {
		return nil
	}

	switch dt := att.Type.(type) {
	case UserType:
		return CanonicalizeExample(dt.Attribute(), example)
	case *Object:
		return canonicalizeObjectExample(dt, example)
	case *Array:
		return canonicalizeArrayExample(dt, example)
	case *Map:
		return canonicalizeMapExample(dt, example)
	case *Union:
		return canonicalizeUnionExample(dt, example)
	default:
		return example
	}
}

func canonicalizeObjectExample(object *Object, example any) any {
	values, ok := stringMapExample(example)
	if !ok {
		return example
	}
	out := make(map[string]any, len(values))
	recognized := make(map[string]struct{}, len(*object)*2)
	for _, field := range *object {
		if field == nil || field.Attribute == nil {
			continue
		}
		wireName := JSONFieldName(ElementName(field.Name), field.Attribute)
		recognized[field.Name] = struct{}{}
		recognized[wireName] = struct{}{}
		if wireName == "-" {
			continue
		}
		value, present := values[wireName]
		if !present {
			value, present = values[field.Name]
		}
		if present {
			out[wireName] = CanonicalizeExample(field.Attribute, value)
		}
	}
	for key, value := range values {
		if _, known := recognized[key]; !known {
			out[key] = value
		}
	}
	return out
}

func canonicalizeArrayExample(array *Array, example any) any {
	values, ok := sliceExample(example)
	if !ok {
		return example
	}
	out := make([]any, len(values))
	for index, value := range values {
		out[index] = CanonicalizeExample(array.ElemType, value)
	}
	return out
}

func canonicalizeMapExample(object *Map, example any) any {
	values, ok := memberMapExample(example)
	if !ok {
		return example
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = CanonicalizeExample(object.ElemType, value)
	}
	return out
}

func canonicalizeUnionExample(union *Union, example any) any {
	if example == nil || len(union.Values) == 0 {
		return example
	}
	var chosen *NamedAttributeExpr
	if selected, ok := example.(examplevalue.Union); ok {
		if selected.Branch < 0 || selected.Branch >= len(union.Values) {
			return nil
		}
		chosen = union.Values[selected.Branch]
		example = selected.Value
	} else {
		chosen = pickUnionVariantForExample(union, example)
	}
	if chosen == nil {
		return example
	}
	if union.Untagged {
		return CanonicalizeExample(chosen.Attribute, example)
	}
	return map[string]any{
		union.GetTypeKey():  UnionVariantTag(chosen),
		union.GetValueKey(): CanonicalizeExample(chosen.Attribute, example),
	}
}

func pickUnionVariantForExample(u *Union, example any) *NamedAttributeExpr {
	if m, ok := stringMapExample(example); ok {
		matches := make([]*NamedAttributeExpr, 0, len(u.Values))
		preferred := make([]*NamedAttributeExpr, 0, len(u.Values))
		for _, nat := range u.Values {
			if nat == nil || nat.Attribute == nil || !exampleMatchesAttribute(nat.Attribute, m) {
				continue
			}
			matches = append(matches, nat)
			if exampleMatchesKnownObjectField(nat.Attribute, m) {
				preferred = append(preferred, nat)
			}
		}
		if len(preferred) == 1 {
			return preferred[0]
		}
		if len(preferred) == 0 && len(matches) == 1 {
			return matches[0]
		}
		return nil
	}

	var chosen *NamedAttributeExpr
	for _, nat := range u.Values {
		if nat == nil || nat.Attribute == nil || !exampleMatchesAttribute(nat.Attribute, example) {
			continue
		}
		if chosen != nil {
			return nil
		}
		chosen = nat
	}
	return chosen
}

func exampleMatchesAttribute(attribute *AttributeExpr, value any) bool {
	if attribute == nil || attribute.Type == nil {
		return false
	}
	if value == nil {
		return AllowsNull(attribute)
	}
	if !attribute.Type.IsCompatible(value) || !exampleMatchesValidation(attribute, value) {
		return false
	}
	if userType, ok := attribute.Type.(UserType); ok {
		return exampleMatchesAttribute(userType.Attribute(), value)
	}
	switch actual := attribute.Type.(type) {
	case *Object:
		object, ok := stringMapExample(value)
		return ok && exampleMatchesObject(attribute, actual, object)
	case *Array:
		items, ok := sliceExample(value)
		if !ok {
			return false
		}
		for _, item := range items {
			if !exampleMatchesAttribute(actual.ElemType, item) {
				return false
			}
		}
	case *Map:
		items, ok := memberMapExample(value)
		if !ok {
			return false
		}
		for _, item := range items {
			if !exampleMatchesAttribute(actual.ElemType, item) {
				return false
			}
		}
	case *Union:
		return pickUnionVariantForExample(actual, value) != nil
	}
	return true
}

func exampleMatchesObject(attribute *AttributeExpr, object *Object, example map[string]any) bool {
	fields := make(map[string]*AttributeExpr, len(*object)*2)
	for _, field := range *object {
		if field == nil || field.Attribute == nil {
			continue
		}
		wireName := JSONFieldName(ElementName(field.Name), field.Attribute)
		if wireName == "-" {
			continue
		}
		fields[field.Name] = field.Attribute
		fields[wireName] = field.Attribute
	}
	additionalProperties, explicitAdditionalProperties := attribute.Meta.Last("openapi:additionalProperties")
	allowsUnknown := !explicitAdditionalProperties || additionalProperties != "false"
	if len(fields) == 0 {
		return len(example) == 0 || allowsUnknown
	}
	for key, value := range example {
		field, ok := fields[key]
		if !ok {
			if !allowsUnknown {
				return false
			}
			continue
		}
		if !exampleMatchesAttribute(field, value) {
			return false
		}
	}
	if attribute.Validation == nil {
		return true
	}
	for _, name := range attribute.Validation.Required {
		fieldName, field := objectExampleField(object, name)
		if field == nil {
			return false
		}
		wireName := JSONFieldName(ElementName(fieldName), field)
		_, authoredPresent := example[fieldName]
		_, wirePresent := example[wireName]
		if !authoredPresent && !wirePresent {
			return false
		}
	}
	return true
}

func exampleMatchesKnownObjectField(attribute *AttributeExpr, example map[string]any) bool {
	if userType, ok := attribute.Type.(UserType); ok {
		return exampleMatchesKnownObjectField(userType.Attribute(), example)
	}
	object, ok := attribute.Type.(*Object)
	if !ok {
		return true
	}
	for _, field := range *object {
		if field == nil || field.Attribute == nil {
			continue
		}
		if _, exists := example[field.Name]; exists {
			return true
		}
		if _, exists := example[JSONFieldName(ElementName(field.Name), field.Attribute)]; exists {
			return true
		}
	}
	return false
}

func exampleMatchesValidation(attribute *AttributeExpr, value any) bool {
	validation := attribute.Validation
	if validation != nil {
		for _, values := range validation.Enums() {
			matched := false
			for _, allowed := range values {
				if exampleEnumValuesEqual(attribute.Type, allowed, value) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}
	return checkLength(attribute, value) && checkPattern(attribute, value) && checkFormat(attribute, value) &&
		checkMinMaxValue(attribute, value)
}

func exampleEnumValuesEqual(datatype DataType, allowed, value any) bool {
	if named, ok := datatype.(UserType); ok {
		return exampleEnumValuesEqual(named.Attribute().Type, allowed, value)
	}
	if datatype == Bytes {
		// Bytes accepts authored text. Compare both representations as bytes
		// before choosing a union branch; String and Any retain their own
		// distinct string/byte semantics.
		if text, ok := allowed.(string); ok {
			allowed = []byte(text)
		}
		if text, ok := value.(string); ok {
			value = []byte(text)
		}
		if left, ok := allowed.([]byte); ok {
			if right, ok := value.([]byte); ok {
				return bytes.Equal(left, right)
			}
		}
	}
	return exampleValuesEqual(allowed, value)
}

func objectExampleField(object *Object, name string) (string, *AttributeExpr) {
	if object == nil {
		return "", nil
	}
	for _, field := range *object {
		if field == nil || field.Attribute == nil {
			continue
		}
		if field.Name == name || JSONFieldName(ElementName(field.Name), field.Attribute) == name {
			return field.Name, field.Attribute
		}
	}
	return "", nil
}

func exampleValuesEqual(left, right any) bool {
	if reflect.DeepEqual(left, right) {
		return true
	}
	leftNil, rightNil := nilExample(left), nilExample(right)
	if leftNil || rightNil {
		return leftNil && rightNil
	}
	if leftNumber, ok := numericExampleRat(left); ok {
		rightNumber, rightOK := numericExampleRat(right)
		return rightOK && leftNumber.Cmp(rightNumber) == 0
	}
	if leftMap, ok := stringMapExample(left); ok {
		rightMap, rightOK := stringMapExample(right)
		if !rightOK || len(leftMap) != len(rightMap) {
			return false
		}
		for key, leftValue := range leftMap {
			rightValue, exists := rightMap[key]
			if !exists || !exampleValuesEqual(leftValue, rightValue) {
				return false
			}
		}
		return true
	}
	if leftSlice, ok := sliceExample(left); ok {
		rightSlice, rightOK := sliceExample(right)
		if !rightOK || len(leftSlice) != len(rightSlice) {
			return false
		}
		for index, leftValue := range leftSlice {
			if !exampleValuesEqual(leftValue, rightSlice[index]) {
				return false
			}
		}
		return true
	}
	return false
}

func numericExampleRat(value any) (*big.Rat, bool) {
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

func stringMapExample(value any) (map[string]any, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() {
		return nil, false
	}
	if actual.Kind() == reflect.Struct {
		return plainStructMap(actual)
	}
	if actual.Kind() != reflect.Map {
		return nil, false
	}
	if actual.IsNil() {
		return nil, true
	}
	out := make(map[string]any, actual.Len())
	for _, entry := range valueSortedMapEntries(actual) {
		key := entry.key
		for key.IsValid() && key.Kind() == reflect.Interface {
			if key.IsNil() {
				return nil, false
			}
			key = key.Elem()
		}
		if !key.IsValid() || key.Kind() != reflect.String {
			return nil, false
		}
		name := key.String()
		if _, exists := out[name]; exists {
			return nil, false
		}
		out[name] = entry.value.Interface()
	}
	return out, true
}

type plainStructField struct {
	value  reflect.Value
	depth  int
	tagged bool
}

func plainStructMap(value reflect.Value) (map[string]any, bool) {
	if value.Kind() != reflect.Struct || valueHasCustomCodec(value.Type()) {
		return nil, false
	}
	candidates := make(map[string][]plainStructField)
	collectPlainStructFields(value, 0, candidates, make(map[uintptr]bool))
	result := make(map[string]any, len(candidates))
	for name, fields := range candidates {
		bestDepth := fields[0].depth
		for _, field := range fields[1:] {
			if field.depth < bestDepth {
				bestDepth = field.depth
			}
		}
		var best []plainStructField
		for _, field := range fields {
			if field.depth == bestDepth {
				best = append(best, field)
			}
		}
		var tagged []plainStructField
		for _, field := range best {
			if field.tagged {
				tagged = append(tagged, field)
			}
		}
		if len(tagged) == 1 {
			best = tagged
		} else if len(tagged) > 1 || len(best) != 1 {
			continue
		}
		result[name] = best[0].value.Interface()
	}
	return result, true
}

func collectPlainStructFields(
	value reflect.Value,
	depth int,
	fields map[string][]plainStructField,
	active map[uintptr]bool,
) {
	for index := range value.NumField() {
		fieldType := value.Type().Field(index)
		if !fieldType.IsExported() {
			continue
		}
		fieldValue := value.Field(index)
		name, _, _ := strings.Cut(fieldType.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		tagged := name != ""
		if fieldType.Anonymous && !tagged {
			embedded := fieldValue
			var pointer uintptr
			for embedded.Kind() == reflect.Pointer && !valueHasCustomCodec(embedded.Type()) {
				if embedded.IsNil() {
					embedded = reflect.Value{}
					break
				}
				pointer = embedded.Pointer()
				if active[pointer] {
					embedded = reflect.Value{}
					break
				}
				active[pointer] = true
				embedded = embedded.Elem()
			}
			if embedded.IsValid() && embedded.Kind() == reflect.Struct && !valueHasCustomCodec(embedded.Type()) {
				collectPlainStructFields(embedded, depth+1, fields, active)
				if pointer != 0 {
					delete(active, pointer)
				}
				continue
			}
			if pointer != 0 {
				delete(active, pointer)
			}
			if !embedded.IsValid() {
				continue
			}
		}
		if !fieldType.IsExported() {
			continue
		}
		if name == "" {
			name = fieldType.Name
		}
		fields[name] = append(fields[name], plainStructField{value: fieldValue, depth: depth, tagged: tagged})
	}
}

// memberMapExample returns a copy of the map example value keyed by the JSON
// object member names of its keys. A string key is its own name. A bool or
// number key is named by its JSON text, as encoding/json/v2 names the numeric
// keys it encodes. It reports false when value is not a map, when a key has
// another kind or no JSON text, or when two keys have the same name.
func memberMapExample(value any) (map[string]any, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || actual.Kind() != reflect.Map {
		return nil, false
	}
	if actual.IsNil() {
		return nil, true
	}
	out := make(map[string]any, actual.Len())
	iterator := actual.MapRange()
	for iterator.Next() {
		name, ok := jsonkey.Name(iterator.Key())
		if !ok {
			return nil, false
		}
		if _, exists := out[name]; exists {
			return nil, false
		}
		out[name] = iterator.Value().Interface()
	}
	return out, true
}

func sliceExample(value any) ([]any, bool) {
	actual := reflect.ValueOf(value)
	if !actual.IsValid() || actual.Kind() != reflect.Array && actual.Kind() != reflect.Slice {
		return nil, false
	}
	if actual.Kind() == reflect.Slice && actual.IsNil() {
		return nil, true
	}
	out := make([]any, actual.Len())
	for index := range actual.Len() {
		out[index] = actual.Index(index).Interface()
	}
	return out, true
}

func nilExample(value any) bool {
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
func unwrapUserTypeAttr(att *AttributeExpr) *AttributeExpr {
	if att == nil || att.Type == nil {
		return att
	}
	if ut, ok := att.Type.(UserType); ok {
		return unwrapUserTypeAttr(ut.Attribute())
	}
	return att
}
