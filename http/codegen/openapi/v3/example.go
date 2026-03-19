package openapiv3

import (
	"goa.design/goa/v3/expr"
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
func initExamples(obj exampler, attr *expr.AttributeExpr, r *expr.ExampleGenerator) {
	if shouldSuppressOpenAPIExamples(attr, false) {
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
	if objectExample, ok := val.(map[string]any); ok && len(objectExample) == 0 && len(attr.AllRequired()) > 0 {
		return nil, false
	}
	return val, true
}

func normalizeOpenAPIExample(val any) any {
	switch actual := val.(type) {
	case []byte:
		return string(actual)
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
