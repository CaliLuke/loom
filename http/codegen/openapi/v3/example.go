package openapiv3

import (
	"github.com/CaliLuke/loom/expr"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
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
			val, ok := authoredExampleValue(attr, ex)
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
		if val, ok := authoredExampleValue(attr, examples[0]); ok {
			obj.setExample(val)
		}
	default:
		if val, ok := openAPIExampleValue(attr, attr.Example(r)); ok {
			obj.setExample(val)
		}
	}
}

func authoredExampleValue(attr *expr.AttributeExpr, example *expr.ExampleExpr) (any, bool) {
	if example != nil && example.ExplicitNull && expr.AllowsNull(attr) {
		return openapiir.NullExample{}, true
	}
	if example == nil {
		return nil, false
	}
	return openAPIExampleValue(attr, example.Value)
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
	return openapiir.OpenAPIExampleValue(attr, raw)
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
