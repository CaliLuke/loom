package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func untaggedMultipartAdditionalMethods() func() {
	filterValue, label, physical, closed, open, numeric, field := untaggedMultipartPhysicalTypes()
	tree, derived, optionalLabel := untaggedMultipartAdditionalTypes(label)
	return func() {
		untaggedMultipartPresenceMethods(optionalLabel, physical)
		for _, item := range []struct {
			name string
			typ  any
		}{
			{"Derived", derived}, {"Physical", physical}, {"Closed", closed}, {"Open", open}, {"Numeric", numeric},
		} {
			Method(item.name, func() {
				Payload(item.typ)
				HTTP(func() {
					POST("/" + item.name)
					MultipartRequest()
					Response(StatusNoContent)
				})
			})
		}
		untaggedMultipartCompositeMethod(field, tree, filterValue)
		Method("Selected", func() {
			Payload(func() {
				Attribute("value", physical)
			})
			HTTP(func() {
				POST("/Selected")
				Body("value")
				MultipartRequest()
				OptionalRequestBody()
				Response(StatusNoContent)
			})
		})
	}
}

func untaggedMultipartPhysicalTypes() (expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType, expr.UserType) {
	filterValue := Type("MultipartFilterValue", func() {
		Attribute("x", String, func() {
			Meta("struct:tag:json", "jsonX")
		})
		Required("x")
	})
	filter := Type("MultipartFilter", func() {
		Attribute("filter", filterValue)
		Required("filter")
		Meta("openapi:encoding:filter:style", "form")
		Meta("openapi:encoding:filter:explode", "true")
	})
	label := Type("MultipartLabel", func() {
		Attribute("label", String)
		Required("label")
	})
	openFilter := Type("MultipartOpenFilter", filter, func() {
		Meta("openapi:additionalProperties", "true")
	})
	physical := Type("MultipartPhysical", OneOf(filter, label), Untagged)
	closed := Type("MultipartClosed", OneOf(filter, label), func() {
		Untagged()
		Meta("openapi:unevaluatedProperties", "false")
	})
	open := Type("MultipartOpen", OneOf(openFilter, label), func() {
		Untagged()
		Meta("openapi:unevaluatedProperties", "false")
	})
	count := Type("MultipartCount", func() {
		Attribute("count", Int32)
		Required("count")
	})
	numeric := Type("MultipartNumeric", OneOf(count, label), Untagged)
	field := Type("MultipartFieldChoice", OneOf(Boolean, filterValue), Untagged)
	return filterValue, label, physical, closed, open, numeric, field
}

func untaggedMultipartAdditionalTypes(label expr.UserType) (expr.UserType, expr.UserType, expr.UserType) {
	tree := Type("MultipartTree", func() {
		Attribute("name", String)
		Attribute("children", ArrayOf("MultipartTree"))
		Required("name")
	})
	metadata := Type("MultipartMetadata", func() {
		Attribute("file", Bytes)
		Attribute("filename", String)
		Required("file", "filename")
		Meta("openapi:additionalProperties", "false")
	})
	derived := Type("MultipartDerived", OneOf(metadata, label), Untagged)
	optionalLabel := Type("MultipartOptionalLabel", func() {
		Attribute("label", String)
	})
	return tree, derived, optionalLabel
}

func untaggedMultipartPresenceMethods(optionalLabel, physical expr.UserType) {
	Method("JSONAny", func() {
		Payload(Any)
		HTTP(func() {
			POST("/JSONAny")
			Response(StatusNoContent)
		})
	})
	for _, item := range []struct {
		name                     string
		typ                      any
		form, optional, nullable bool
	}{
		{"JSONOptionalObject", optionalLabel, false, true, false},
		{"JSONRequiredObject", optionalLabel, false, false, false},
		{"FormOptionalObject", optionalLabel, true, true, false},
		{"JSONOptionalUnion", physical, false, true, false},
		{"JSONRequiredUnion", physical, false, false, false},
		{"FormOptionalUnion", physical, true, true, false},
		{"FormRequiredUnion", physical, true, false, false},
		{"JSONOptionalNullable", optionalLabel, false, true, true},
		{"JSONRequiredNullable", optionalLabel, false, false, true},
	} {
		Method(item.name, func() {
			if item.nullable {
				Payload(item.typ, Nullable)
			} else {
				Payload(item.typ)
			}
			HTTP(func() {
				POST("/" + item.name)
				if item.typ == optionalLabel {
					Body(func() {
						Attribute("label")
						if item.nullable {
							Nullable()
						}
					})
				}
				if item.form {
					FormRequest()
				}
				if item.optional {
					OptionalRequestBody()
				}
				Response(StatusNoContent)
			})
		})
	}
}

func untaggedMultipartCompositeMethod(field, tree, filterValue expr.UserType) {
	Method("Composite", func() {
		Payload(func() {
			Attribute("value", field)
			Attribute("jsonValue", field, Nullable)
			Attribute("any", Any)
			Attribute("tree", tree)
			Attribute("files", ArrayOf(Bytes))
			Attribute("records", ArrayOf(filterValue))
			Required("value", "jsonValue")
			Meta("openapi:encoding:value:style", "form")
			Meta("openapi:encoding:value:explode", "false")
			Meta("openapi:encoding:jsonValue:contentType", "application/json")
			Meta("openapi:encoding:any:contentType", "application/json")
		})
		HTTP(func() {
			POST("/Composite")
			MultipartRequest()
			Response(StatusNoContent)
		})
	})
}
