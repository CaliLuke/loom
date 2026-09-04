package ir

import (
	"math/big"
	"reflect"
	"unicode/utf8"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	loom "github.com/CaliLuke/loom/pkg"
)

func untaggedUnionExampleMatches(union *expr.Union, value any) bool {
	matches := 0
	for _, branch := range union.Values {
		if branch != nil && branch.Attribute != nil && untaggedBranchExampleMatches(branch.Attribute, value) {
			matches++
		}
	}
	return matches == 1
}

func untaggedBranchExampleMatches(branch *expr.AttributeExpr, value any) bool {
	objectValue, ok := value.(map[string]any)
	if !ok {
		return false
	}
	branch = unwrapExampleAttribute(branch)
	if branch == nil {
		return false
	}
	object := expr.AsObject(branch.Type)
	if object == nil {
		return false
	}
	fields := make(map[string]*expr.AttributeExpr, len(*object))
	suppressed := make(map[string]struct{})
	for _, field := range *object {
		if field == nil || field.Attribute == nil {
			continue
		}
		name := codegen.JSONFieldName(field.Name, field.Attribute)
		if !openapi.MustGenerate(field.Attribute.Meta) {
			suppressed[name] = struct{}{}
			continue
		}
		fields[name] = field.Attribute
		if branch.IsRequired(field.Name) {
			if _, present := objectValue[name]; !present {
				return false
			}
		}
	}
	closed, _ := branch.Meta.Last("openapi:additionalProperties")
	for name, fieldValue := range objectValue {
		if _, ignored := suppressed[name]; ignored {
			continue
		}
		field, defined := fields[name]
		if !defined {
			if closed == "false" {
				return false
			}
			continue
		}
		if !openAPIFieldExampleMatches(field, fieldValue) {
			return false
		}
	}
	return true
}

func openAPIFieldExampleMatches(attribute *expr.AttributeExpr, value any) bool {
	if attribute == nil || attribute.Type == nil {
		return false
	}
	if value == nil {
		return expr.AllowsNull(attribute)
	}
	if !validationMatches(attribute.Validation, value) {
		return false
	}
	if userType, ok := attribute.Type.(expr.UserType); ok {
		return openAPIFieldExampleMatches(userType.Attribute(), value)
	}
	if object := expr.AsObject(attribute.Type); object != nil {
		objectValue, ok := value.(map[string]any)
		return ok && untaggedBranchExampleMatches(attribute, objectValue)
	}
	if array := expr.AsArray(attribute.Type); array != nil {
		items, ok := value.([]any)
		if !ok || !containerLengthMatches(attribute.Validation, len(items)) {
			return false
		}
		for _, item := range items {
			if !openAPIFieldExampleMatches(array.ElemType, item) {
				return false
			}
		}
		return true
	}
	if mapping := expr.AsMap(attribute.Type); mapping != nil {
		items, ok := value.(map[string]any)
		if !ok || !containerLengthMatches(attribute.Validation, len(items)) {
			return false
		}
		for _, item := range items {
			if !openAPIFieldExampleMatches(mapping.ElemType, item) {
				return false
			}
		}
		return true
	}
	if union := expr.AsUnion(attribute.Type); union != nil {
		if union.Untagged {
			return untaggedUnionExampleMatches(union, value)
		}
		example, ok := value.(map[string]any)
		if !ok {
			return false
		}
		tag, branchValue, ok := openAPIUnionTagAndValue(union, example)
		if !ok {
			return false
		}
		for _, branch := range union.Values {
			if branch != nil && branch.Attribute != nil && expr.UnionVariantTag(branch) == tag {
				return openAPIFieldExampleMatches(branch.Attribute, branchValue)
			}
		}
		return false
	}
	return primitiveExampleMatches(attribute, value)
}

func containerLengthMatches(validation *expr.ValidationExpr, length int) bool {
	if validation == nil {
		return true
	}
	if validation.MinLength != nil && length < *validation.MinLength {
		return false
	}
	return validation.MaxLength == nil || length <= *validation.MaxLength
}

func unwrapExampleAttribute(attribute *expr.AttributeExpr) *expr.AttributeExpr {
	seen := make(map[string]struct{})
	for attribute != nil {
		userType, ok := attribute.Type.(expr.UserType)
		if !ok {
			return attribute
		}
		if _, exists := seen[userType.ID()]; exists {
			return nil
		}
		seen[userType.ID()] = struct{}{}
		attribute = userType.Attribute()
	}
	return nil
}

func primitiveExampleMatches(attribute *expr.AttributeExpr, value any) bool {
	if value == nil {
		return expr.AllowsNull(attribute)
	}
	if attribute == nil || !primitiveTypeMatches(attribute.Type, value) || !validationMatches(attribute.Validation, value) {
		return false
	}
	if userType, ok := attribute.Type.(expr.UserType); ok {
		return primitiveExampleMatches(userType.Attribute(), value)
	}
	return true
}

func primitiveTypeMatches(dataType expr.DataType, value any) bool {
	for {
		userType, ok := dataType.(expr.UserType)
		if !ok {
			break
		}
		dataType = userType.Attribute().Type
	}
	switch dataType.Kind() {
	case expr.AnyKind:
		return true
	case expr.BooleanKind:
		_, ok := value.(bool)
		return ok
	case expr.StringKind, expr.BytesKind:
		_, ok := value.(string)
		return ok
	case expr.IntKind, expr.Int32Kind, expr.Int64Kind, expr.UIntKind, expr.UInt32Kind, expr.UInt64Kind:
		return integerExampleValue(value)
	case expr.Float32Kind, expr.Float64Kind:
		_, ok := numericExampleValue(value)
		return ok
	default:
		return false
	}
}

func validationMatches(validation *expr.ValidationExpr, value any) bool {
	if validation == nil {
		return true
	}
	if len(validation.Values) > 0 {
		matched := false
		for _, candidate := range validation.Values {
			if exampleValuesEqual(candidate, value) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if stringValue, ok := value.(string); ok {
		if validation.Pattern != "" && loom.ValidatePattern("example", stringValue, validation.Pattern) != nil {
			return false
		}
		if validation.Format != "" && loom.ValidateFormat("example", stringValue, loom.Format(validation.Format)) != nil {
			return false
		}
		if validation.MinLength != nil && utf8.RuneCountInString(stringValue) < *validation.MinLength {
			return false
		}
		if validation.MaxLength != nil && utf8.RuneCountInString(stringValue) > *validation.MaxLength {
			return false
		}
	}
	number, numeric := numericExampleValue(value)
	if !numeric {
		return true
	}
	if validation.Minimum != nil && number.Cmp(new(big.Rat).SetFloat64(*validation.Minimum)) < 0 {
		return false
	}
	if validation.ExclusiveMinimum != nil && number.Cmp(new(big.Rat).SetFloat64(*validation.ExclusiveMinimum)) <= 0 {
		return false
	}
	if validation.Maximum != nil && number.Cmp(new(big.Rat).SetFloat64(*validation.Maximum)) > 0 {
		return false
	}
	return validation.ExclusiveMaximum == nil || number.Cmp(new(big.Rat).SetFloat64(*validation.ExclusiveMaximum)) < 0
}

func exampleValuesEqual(left, right any) bool {
	if reflect.DeepEqual(left, right) {
		return true
	}
	leftNumber, leftNumeric := numericExampleValue(left)
	rightNumber, rightNumeric := numericExampleValue(right)
	if leftNumeric || rightNumeric {
		return leftNumeric && rightNumeric && leftNumber.Cmp(rightNumber) == 0
	}
	if leftMap, ok := openAPIStringMap(left); ok {
		rightMap, rightOK := openAPIStringMap(right)
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
	if leftSlice, ok := openAPISlice(left); ok {
		rightSlice, rightOK := openAPISlice(right)
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

func integerExampleValue(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func numericExampleValue(value any) (*big.Rat, bool) {
	number := new(big.Rat)
	switch actual := value.(type) {
	case int:
		return number.SetInt64(int64(actual)), true
	case int8:
		return number.SetInt64(int64(actual)), true
	case int16:
		return number.SetInt64(int64(actual)), true
	case int32:
		return number.SetInt64(int64(actual)), true
	case int64:
		return number.SetInt64(actual), true
	case uint:
		return number.SetInt(new(big.Int).SetUint64(uint64(actual))), true
	case uint8:
		return number.SetInt64(int64(actual)), true
	case uint16:
		return number.SetInt64(int64(actual)), true
	case uint32:
		return number.SetInt64(int64(actual)), true
	case uint64:
		return number.SetInt(new(big.Int).SetUint64(actual)), true
	case float32:
		return number.SetFloat64(float64(actual)), true
	case float64:
		return number.SetFloat64(actual), true
	case openAPIJSONNumber:
		parsed, ok := number.SetString(string(actual))
		return parsed, ok
	default:
		return nil, false
	}
}
