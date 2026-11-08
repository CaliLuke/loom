package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// DiscriminatorHintDSL exercises validation-neutral hints and the distinct
// captured JSON-field, styled-field and native tagged form-root representations.
func DiscriminatorHintDSL() {
	discriminatorHintDSL(true)
}

// DiscriminatorNoHintDSL uses the same wire contract without OpenAPI hints.
func DiscriminatorNoHintDSL() {
	discriminatorHintDSL(false)
}

func discriminatorHintDSL(hints bool) {
	API("discriminator-hints", func() {
		Description("Discriminator hints and captured form representations")
	})
	code := Type("DiscriminatorCode", func() {
		Attribute("code", String)
		Required("code")
	})
	grant := discriminatorHintGrant(code, hints)
	jsonField := Type("DiscriminatorJSONField", func() {
		Attribute("Values", grant)
		Required("Values")
	})
	objectField := Type("DiscriminatorObjectField", func() {
		Attribute("Values", code)
		Required("Values")
	})
	mapField := Type("DiscriminatorMapField", func() {
		Attribute("Values", MapOf(String, String))
		Required("Values")
	})
	hinted := discriminatorHintChoice(hints)
	fullObjectField := Type("DiscriminatorFullObjectField", func() {
		Attribute("Values", hinted)
		Required("Values")
	})
	Service("hint", func() {
		for _, method := range []struct {
			name     string
			typ      any
			encoding string
		}{
			{"FlatRoot", grant, "form"},
			{"NestedRoot", grant, "form"},
			{"JSONRoot", grant, "json"},
			{"JSONField", jsonField, "form"},
			{"ObjectField", objectField, "form"},
			{"MapField", mapField, "form"},
			{"FullObjectField", fullObjectField, "form"},
			{"MultipartField", jsonField, "multipart"},
			{"HintNeutral", hinted, "json"},
		} {
			Method(method.name, func() {
				Payload(method.typ)
				Result(method.typ)
				HTTP(func() {
					POST("/" + method.name)
					if method.name == "ObjectField" || method.name == "MapField" || method.name == "FullObjectField" {
						Body(func() {
							Attribute("Values")
							Meta("openapi:encoding:Values:style", "deepObject")
						})
					}
					switch method.encoding {
					case "form":
						FormRequest()
					case "multipart":
						MultipartRequest()
					}
					Response(StatusOK)
				})
			})
		}
	})
}

func discriminatorHintGrant(code expr.UserType, hints bool) expr.UserType {
	nested := Type("DiscriminatorNested", func() {
		Attribute("details", code)
		Required("details")
	})
	nullableRecord := Type("DiscriminatorNullableRecord", func() {
		Attribute("details", code, func() {
			Nullable()
		})
		Required("details")
	})
	arrayRecord := Type("DiscriminatorArrayRecord", func() {
		Attribute("records", ArrayOf(code))
		Required("records")
	})
	return Type("DiscriminatorGrant", OneOf(func() {
		Attribute("AuthorizationCode", code, func() {
			Meta("oneof:type:tag", "authorization_code")
			if hints {
				Meta("openapi:discriminator:mapping", "code")
			}
		})
		Attribute("NestedCode", nested, func() {
			Meta("oneof:type:tag", "nested_code")
		})
		Attribute("NullableRecord", nullableRecord, func() {
			Meta("oneof:type:tag", "nullable_record")
		})
		Attribute("Records", ArrayOf(code), func() {
			Meta("oneof:type:tag", "records")
		})
		Attribute("NonEmptyRecords", ArrayOf(code), func() {
			Meta("oneof:type:tag", "nonempty_records")
			MinLength(1)
		})
		Attribute("Matrix", ArrayOf(ArrayOf(Int)), func() {
			Meta("oneof:type:tag", "matrix")
		})
		Attribute("ArrayRecord", arrayRecord, func() {
			Meta("oneof:type:tag", "array_record")
		})
		Attribute("Labels", MapOf(String, String), func() {
			Meta("oneof:type:tag", "labels")
		})
		Attribute("NullableLabels", MapOf(String, String), func() {
			Meta("oneof:type:tag", "nullable_labels")
			Nullable()
		})
		Attribute("Tags", ArrayOf(String), func() {
			Meta("oneof:type:tag", "tags")
		})
		Attribute("NonEmptyTags", ArrayOf(String), func() {
			Meta("oneof:type:tag", "nonempty_tags")
			MinLength(1)
		})
		Attribute("Text", String, func() {
			Meta("oneof:type:tag", "text")
		})
	}), func() {
		Meta("oneof:type:field", "grant_type")
		Meta("oneof:value:field", "data")
		Example(struct {
			Code string
		}{Code: "abc"})
	})
}

func discriminatorHintChoice(hints bool) expr.UserType {
	cat := Type("DiscriminatorCat", func() {
		Attribute("kind", String, func() {
			Enum("cat")
		})
		Required("kind")
	})
	dog := Type("DiscriminatorDog", func() {
		Attribute("kind", String, func() {
			Enum("dog")
		})
		Required("kind")
	})
	return Type("DiscriminatorChoice", OneOf(func() {
		Untagged()
		Attribute("Cat", cat, func() {
			if hints {
				Meta("openapi:discriminator:mapping", "cat", "kitty")
			}
		})
		Attribute("Dog", dog)
	}), func() {
		if hints {
			Meta("openapi:discriminator:propertyName", "kind")
			Meta("openapi:discriminator:extension:x-audit", `{"v":1}`)
		}
	})
}
