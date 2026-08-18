package ir

import (
	"math"
	"reflect"
	"unicode/utf8"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
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
	for _, field := range *object {
		name := codegen.JSONFieldName(field.Name, field.Attribute)
		fields[name] = field.Attribute
		if branch.IsRequired(field.Name) {
			if _, present := objectValue[name]; !present {
				return false
			}
		}
	}
	closed, _ := branch.Meta.Last("openapi:additionalProperties")
	for name, fieldValue := range objectValue {
		field, defined := fields[name]
		if !defined {
			if closed == "false" {
				return false
			}
			continue
		}
		if !primitiveExampleMatches(field, fieldValue) {
			return false
		}
	}
	return true
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
		_, ok := integerExampleValue(value)
		return ok
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
	if validation.Minimum != nil && number < *validation.Minimum {
		return false
	}
	if validation.ExclusiveMinimum != nil && number <= *validation.ExclusiveMinimum {
		return false
	}
	if validation.Maximum != nil && number > *validation.Maximum {
		return false
	}
	return validation.ExclusiveMaximum == nil || number < *validation.ExclusiveMaximum
}

func exampleValuesEqual(left, right any) bool {
	leftNumber, leftNumeric := numericExampleValue(left)
	rightNumber, rightNumeric := numericExampleValue(right)
	if leftNumeric && rightNumeric {
		return leftNumber == rightNumber
	}
	return reflect.DeepEqual(left, right)
}

func integerExampleValue(value any) (int64, bool) {
	switch actual := value.(type) {
	case int:
		return int64(actual), true
	case int8:
		return int64(actual), true
	case int16:
		return int64(actual), true
	case int32:
		return int64(actual), true
	case int64:
		return actual, true
	case uint:
		return int64(actual), uint64(actual) <= math.MaxInt64
	case uint8:
		return int64(actual), true
	case uint16:
		return int64(actual), true
	case uint32:
		return int64(actual), true
	case uint64:
		return int64(actual), actual <= math.MaxInt64
	default:
		return 0, false
	}
}

func numericExampleValue(value any) (float64, bool) {
	if integer, ok := integerExampleValue(value); ok {
		return float64(integer), true
	}
	switch actual := value.(type) {
	case float32:
		return float64(actual), true
	case float64:
		return actual, true
	default:
		return 0, false
	}
}
