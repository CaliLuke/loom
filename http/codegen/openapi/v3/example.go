package openapiv3

import (
	"reflect"

	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/openapi"
)

type (
	// exampler is the interface used to initialize the example of an
	// OpenAPI object.
	exampler interface {
		setExample(any)
		setExamples(map[string]*ExampleRef)
	}
)

// initExample sets the example or examples of the given object.
func initExamples(obj exampler, attr *expr.AttributeExpr, r *expr.ExampleGenerator, closeObjects bool) {
	if shouldSuppressOpenAPIExamples(attr, closeObjects) {
		return
	}
	examples := attr.ExtractUserExamples()
	switch {
	case len(examples) > 1:
		refs := make(map[string]*ExampleRef, len(examples))
		for _, ex := range examples {
			val, ok := openAPIExampleValue(attr, ex.Value)
			if !ok {
				continue
			}
			example := &Example{
				Summary:     ex.Summary,
				Description: ex.Description,
				Value:       val,
			}
			refs[ex.Summary] = &ExampleRef{Value: example}
		}
		if len(refs) == 0 {
			return
		}
		obj.setExamples(refs)
		return
	case len(examples) > 0:
		if val, ok := openAPIExampleValue(attr, examples[0].Value); ok {
			obj.setExample(val)
		}
	default:
		if val, ok := openAPIExampleValue(attr, attr.Example(r)); ok {
			obj.setExample(val)
		}
	}
}

func shouldSuppressOpenAPIExamples(attr *expr.AttributeExpr, closeObjects bool) bool {
	if attr == nil {
		return true
	}
	if disabled, ok := attr.Meta.Last("openapi:example"); ok && disabled == "false" {
		return true
	}
	if objectContainsSuppressedOpenAPIExample(attr, closeObjects, map[string]struct{}{}, map[expr.DataType]struct{}{}) {
		return true
	}
	if isUnionWrapperObjectType(attr.Type) {
		return true
	}
	if closeObjects && isUnionType(attr.Type) {
		return true
	}
	return false
}

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

func openAPIExampleValue(attr *expr.AttributeExpr, raw any) (any, bool) {
	if raw == nil {
		return nil, false
	}
	val := normalizeOpenAPIExample(expr.CanonicalizeExample(attr, raw))
	if !isCompleteOpenAPIExample(attr, val) {
		return nil, false
	}
	return val, true
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
		obj, ok := val.(map[string]any)
		if !ok {
			return false
		}
		if len(obj) == 0 && len(attr.AllRequired()) > 0 {
			return false
		}
		for _, name := range attr.AllRequired() {
			child := attr.Find(name)
			if child == nil {
				continue
			}
			if !openapi.MustGenerate(child.Meta) {
				continue
			}
			fieldVal, ok := obj[name]
			if !ok {
				return false
			}
			if !isCompleteOpenAPIExample(child, fieldVal) {
				return false
			}
		}
		return true
	case *expr.Array:
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
	case *expr.Map:
		m, ok := val.(map[string]any)
		if !ok {
			return true
		}
		for _, item := range m {
			if !isCompleteOpenAPIExample(actual.ElemType, item) {
				return false
			}
		}
		return true
	case *expr.Union:
		example, ok := val.(map[string]any)
		if !ok {
			return false
		}
		typeKey := actual.GetTypeKey()
		valueKey := actual.GetValueKey()
		rawTag, ok := example[typeKey]
		if !ok {
			return false
		}
		tag, ok := rawTag.(string)
		if !ok || tag == "" {
			return false
		}
		rawValue, ok := example[valueKey]
		if !ok {
			return false
		}
		for _, branch := range actual.Values {
			if branch == nil || branch.Attribute == nil {
				continue
			}
			if expr.UnionVariantTag(branch) == tag {
				return isCompleteOpenAPIExample(branch.Attribute, rawValue)
			}
		}
		return false
	default:
		return true
	}
}

func normalizeOpenAPIExample(val any) any {
	switch actual := val.(type) {
	case []byte:
		return string(actual)
	case expr.Val:
		out := make(map[string]any, len(actual))
		for k, v := range actual {
			out[k] = normalizeOpenAPIExample(v)
		}
		return out
	case expr.ArrayVal:
		out := make([]any, len(actual))
		for i, v := range actual {
			out[i] = normalizeOpenAPIExample(v)
		}
		return out
	case expr.MapVal:
		out := make(map[string]any, len(actual))
		for k, v := range actual {
			key, ok := k.(string)
			if !ok {
				return val
			}
			out[key] = normalizeOpenAPIExample(v)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(actual))
		for k, v := range actual {
			out[k] = normalizeOpenAPIExample(v)
		}
		return out
	case []any:
		out := make([]any, len(actual))
		for i, v := range actual {
			out[i] = normalizeOpenAPIExample(v)
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
		}
		return val
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
