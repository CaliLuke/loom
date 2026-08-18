package ir

import (
	"reflect"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func objectContainsSuppressedOpenAPIExample(attr *expr.AttributeExpr, closeObjects bool, seenUT map[string]struct{}, seenDT map[expr.DataType]struct{}) bool {
	if attr == nil || attr.Type == nil {
		return false
	}
	if _, ok := seenDT[attr.Type]; ok {
		return false
	}
	seenDT[attr.Type] = struct{}{}

	switch actual := attr.Type.(type) {
	case expr.UserType:
		id := actual.ID()
		if _, ok := seenUT[id]; ok {
			return false
		}
		seenUT[id] = struct{}{}
		return objectContainsSuppressedOpenAPIExample(actual.Attribute(), closeObjects, seenUT, seenDT)
	case *expr.Array:
		return objectContainsSuppressedOpenAPIExample(actual.ElemType, closeObjects, seenUT, seenDT)
	case *expr.Map:
		return objectContainsSuppressedOpenAPIExample(actual.KeyType, closeObjects, seenUT, seenDT) ||
			objectContainsSuppressedOpenAPIExample(actual.ElemType, closeObjects, seenUT, seenDT)
	case *expr.Object:
		for _, nat := range *actual {
			if nat == nil || nat.Attribute == nil {
				continue
			}
			if disabled, ok := nat.Attribute.Meta.Last("openapi:example"); ok && disabled == "false" {
				return true
			}
			if isUnionWrapperObjectTypeSeen(nat.Attribute.Type, seenUT, seenDT) {
				return true
			}
			if closeObjects && isUnionTypeSeen(nat.Attribute.Type, seenUT) {
				return true
			}
			if objectContainsSuppressedOpenAPIExample(nat.Attribute, closeObjects, seenUT, seenDT) {
				return true
			}
		}
	}
	return false
}

func isCompleteOpenAPIExample(attr *expr.AttributeExpr, val any) bool {
	if attr == nil {
		return val != nil
	}
	if val == nil {
		return false
	}
	switch actual := attr.Type.(type) {
	case expr.UserType:
		return isCompleteOpenAPIExample(actual.Attribute(), val)
	case *expr.Object:
		return completeOpenAPIObjectExample(attr, val)
	case *expr.Array:
		return completeOpenAPIArrayExample(actual, val)
	case *expr.Map:
		return completeOpenAPIMapExample(actual, val)
	case *expr.Union:
		return completeOpenAPIUnionExample(actual, val)
	default:
		return true
	}
}

func completeOpenAPIObjectExample(attr *expr.AttributeExpr, val any) bool {
	obj, ok := val.(map[string]any)
	if !ok {
		return false
	}
	required := attr.AllRequired()
	if len(obj) == 0 && len(required) > 0 {
		return false
	}
	for _, name := range required {
		if !requiredOpenAPIFieldPresent(attr, obj, name) {
			return false
		}
	}
	return true
}

func requiredOpenAPIFieldPresent(attr *expr.AttributeExpr, obj map[string]any, name string) bool {
	child := attr.Find(name)
	if child == nil || !openapi.MustGenerate(child.Meta) {
		return true
	}
	fieldVal, ok := obj[name]
	return ok && isCompleteOpenAPIExample(child, fieldVal)
}

func completeOpenAPIArrayExample(actual *expr.Array, val any) bool {
	items, ok := val.([]any)
	if !ok {
		return true
	}
	for _, item := range items {
		if !isCompleteOpenAPIExample(actual.ElemType, item) {
			return false
		}
	}
	return true
}

func completeOpenAPIMapExample(actual *expr.Map, val any) bool {
	obj, ok := val.(map[string]any)
	if !ok {
		return true
	}
	for _, item := range obj {
		if !isCompleteOpenAPIExample(actual.ElemType, item) {
			return false
		}
	}
	return true
}

func completeOpenAPIUnionExample(actual *expr.Union, val any) bool {
	if actual.Untagged {
		return untaggedUnionExampleMatches(actual, val)
	}
	example, ok := val.(map[string]any)
	if !ok {
		return false
	}
	tag, rawValue, ok := openAPIUnionTagAndValue(actual, example)
	if !ok {
		return false
	}
	for _, branch := range actual.Values {
		if branch != nil && branch.Attribute != nil && expr.UnionVariantTag(branch) == tag {
			return isCompleteOpenAPIExample(branch.Attribute, rawValue)
		}
	}
	return false
}

func openAPIUnionTagAndValue(actual *expr.Union, example map[string]any) (string, any, bool) {
	rawTag, ok := example[actual.GetTypeKey()]
	if !ok {
		return "", nil, false
	}
	tag, ok := rawTag.(string)
	if !ok || tag == "" {
		return "", nil, false
	}
	rawValue, ok := example[actual.GetValueKey()]
	if !ok {
		return "", nil, false
	}
	return tag, rawValue, true
}

func normalizeOpenAPIExample(val any) any {
	switch actual := val.(type) {
	case []byte:
		return string(actual)
	case expr.Val:
		out := make(map[string]any, len(actual))
		for key, value := range actual {
			out[key] = normalizeOpenAPIExample(value)
		}
		return out
	case expr.ArrayVal:
		out := make([]any, len(actual))
		for i, value := range actual {
			out[i] = normalizeOpenAPIExample(value)
		}
		return out
	case expr.MapVal:
		out := make(map[string]any, len(actual))
		for key, value := range actual {
			stringKey, ok := key.(string)
			if !ok {
				return val
			}
			out[stringKey] = normalizeOpenAPIExample(value)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(actual))
		for key, value := range actual {
			out[key] = normalizeOpenAPIExample(value)
		}
		return out
	case []any:
		out := make([]any, len(actual))
		for i, value := range actual {
			out[i] = normalizeOpenAPIExample(value)
		}
		return out
	default:
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Map:
			out := make(map[string]any, rv.Len())
			iter := rv.MapRange()
			for iter.Next() {
				key := iter.Key()
				if key.Kind() != reflect.String {
					return val
				}
				out[key.String()] = normalizeOpenAPIExample(iter.Value().Interface())
			}
			return out
		case reflect.Slice, reflect.Array:
			out := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out[i] = normalizeOpenAPIExample(rv.Index(i).Interface())
			}
			return out
		default:
			return val
		}
	}
}

func isUnionWrapperObjectType(dt expr.DataType) bool {
	return isUnionWrapperObjectTypeSeen(dt, map[string]struct{}{}, map[expr.DataType]struct{}{})
}

func isUnionWrapperObjectTypeSeen(dt expr.DataType, seenUT map[string]struct{}, seenDT map[expr.DataType]struct{}) bool {
	if dt == nil {
		return false
	}
	if _, ok := seenDT[dt]; ok {
		return false
	}
	seenDT[dt] = struct{}{}
	obj, ok := unwrapExampleDataType(dt, seenUT).(*expr.Object)
	if !ok || len(*obj) != 1 {
		return false
	}
	fieldType := (*obj)[0].Attribute.Type
	return isUnionTypeSeen(fieldType, seenUT) || isUnionWrapperObjectTypeSeen(fieldType, seenUT, seenDT)
}

func isUnionType(dt expr.DataType) bool {
	return isUnionTypeSeen(dt, map[string]struct{}{})
}

func isUnionTypeSeen(dt expr.DataType, seen map[string]struct{}) bool {
	_, ok := unwrapExampleDataType(dt, seen).(*expr.Union)
	return ok
}

func unwrapExampleDataType(dt expr.DataType, seen map[string]struct{}) expr.DataType {
	for {
		ut, ok := dt.(expr.UserType)
		if !ok {
			return dt
		}
		id := ut.ID()
		if _, ok := seen[id]; ok {
			return nil
		}
		seen[id] = struct{}{}
		attr := ut.Attribute()
		if attr == nil {
			return nil
		}
		dt = attr.Type
	}
}
