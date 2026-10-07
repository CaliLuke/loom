package ir

import (
	"encoding/json/v2"
	"math/big"
	"reflect"
	"unicode/utf8"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/unionjson"
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
	plan, err := unionjson.Plan(branch, expr.ElementName, true, false)
	if err != nil {
		return false
	}
	wire, err := json.Marshal(normalizeOpenAPIExample(value), loom.JSONOptions(), json.Deterministic(true))
	if err != nil {
		return false
	}
	matched, err := plan.MatchJSONSchema(wire)
	return err == nil && matched
}

func openAPIFieldExampleMatches(attribute *expr.AttributeExpr, value any) bool {
	if attribute == nil || attribute.Type == nil {
		return false
	}
	if value == nil {
		return expr.AllowsNull(attribute)
	}
	if !attributeValidationMatches(attribute, value) {
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

func primitiveExampleMatches(attribute *expr.AttributeExpr, value any) bool {
	if value == nil {
		return expr.AllowsNull(attribute)
	}
	if attribute == nil || !primitiveTypeMatches(attribute.Type, value) || !attributeValidationMatches(attribute, value) {
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

func attributeValidationMatches(attribute *expr.AttributeExpr, value any) bool {
	validation := attribute.Validation
	if validation != nil && len(validation.Enums()) > 0 {
		// Compare against the same enum representation emitted in the schema.
		// Keep the authored validation intact for runtime and other transports.
		projected := validation.Dup()
		if projected.Values != nil {
			projected.Values = projectOpenAPIValues(attribute, projected.Values)
		}
		for index, values := range projected.EnumClauses {
			projected.EnumClauses[index] = projectOpenAPIValues(attribute, values)
		}
		validation = projected
	}
	return validationMatches(validation, value)
}

func validationMatches(validation *expr.ValidationExpr, value any) bool {
	if validation == nil {
		return true
	}
	for _, values := range validation.Enums() {
		matched := false
		for _, candidate := range values {
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
		for _, pattern := range validation.Patterns() {
			if loom.ValidatePattern("example", stringValue, pattern) != nil {
				return false
			}
		}
		for _, format := range validation.Formats() {
			if loom.ValidateFormat("example", stringValue, loom.Format(format)) != nil {
				return false
			}
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
