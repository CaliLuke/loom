package codegen

import "github.com/CaliLuke/loom/expr"

func cliJSONPackageName(service *ServiceData) string {
	for _, spec := range clientCLIImports("", service) {
		if spec.Path == "github.com/CaliLuke/loom/http/cli" {
			return spec.Name
		}
	}
	panic("missing HTTP CLI JSON decoder import")
}

func containsBooleanMapKeys(datatype expr.DataType) bool {
	seen := make(map[expr.DataType]bool)
	var visit func(expr.DataType) bool
	// visitAdditional visits the typed additional members of the object
	// attribute declares, which its data type does not carry.
	visitAdditional := func(attribute *expr.AttributeExpr) bool {
		return attribute.AdditionalAttributes != nil && visit(attribute.AdditionalAttributes.Type)
	}
	visit = func(datatype expr.DataType) bool {
		if seen[datatype] {
			return false
		}
		seen[datatype] = true
		switch actual := datatype.(type) {
		case expr.UserType:
			return visit(actual.Attribute().Type) || visitAdditional(actual.Attribute())
		case *expr.Map:
			key := actual.KeyType.Type
			for {
				alias, ok := key.(expr.UserType)
				if !ok {
					break
				}
				key = alias.Attribute().Type
			}
			return key == expr.Boolean || visit(actual.ElemType.Type)
		case *expr.Array:
			return visit(actual.ElemType.Type)
		case *expr.Object:
			for _, field := range *actual {
				if visit(field.Attribute.Type) || visitAdditional(field.Attribute) {
					return true
				}
			}
		case *expr.Union:
			for _, variant := range actual.Values {
				if visit(variant.Attribute.Type) {
					return true
				}
			}
		}
		return false
	}
	return visit(datatype)
}
