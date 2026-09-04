package expr

import (
	"math/big"
	"reflect"
)

// CanonicalizeExample normalizes example values so Loom unions use their
// canonical discriminator/value JSON shape.
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
		wireName := JSONFieldName(field.Name, field.Attribute)
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
	values, ok := stringMapExample(example)
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
	chosen := pickUnionVariantForExample(union, example)
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
		items, ok := stringMapExample(value)
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
		wireName := JSONFieldName(field.Name, field.Attribute)
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
		wireName := JSONFieldName(fieldName, field)
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
		if _, exists := example[JSONFieldName(field.Name, field.Attribute)]; exists {
			return true
		}
	}
	return false
}

func exampleMatchesValidation(attribute *AttributeExpr, value any) bool {
	validation := attribute.Validation
	if validation != nil && len(validation.Values) > 0 {
		matched := false
		for _, allowed := range validation.Values {
			if exampleValuesEqual(allowed, value) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return checkLength(attribute, value) && checkPattern(attribute, value) && checkMinMaxValue(attribute, value)
}

func objectExampleField(object *Object, name string) (string, *AttributeExpr) {
	if object == nil {
		return "", nil
	}
	for _, field := range *object {
		if field == nil || field.Attribute == nil {
			continue
		}
		if field.Name == name || JSONFieldName(field.Name, field.Attribute) == name {
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
	if !actual.IsValid() || actual.Kind() != reflect.Map {
		return nil, false
	}
	if actual.IsNil() {
		return nil, true
	}
	out := make(map[string]any, actual.Len())
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
		out[key.String()] = iterator.Value().Interface()
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
