package testdata

import . "github.com/CaliLuke/loom/dsl"

// UntaggedMultipartDSL exercises object alternatives and captured part codecs.
var UntaggedMultipartDSL = func() {
	additional := untaggedMultipartAdditionalMethods()
	empty := Type("EmptyUpload", func() {
		Meta("openapi:additionalProperties", "false")
	})
	record := Type("NamedUpload", func() {
		Attribute("name", String)
		Attribute("file", Bytes)
		Required("name")
		Meta("openapi:additionalProperties", "false")
	})
	choice := Type("UploadChoice", OneOf(empty, record), func() {
		Untagged()
	})
	texts := Type("TextItems", ArrayOf(String))
	numbers := Type("NumberItems", ArrayOf(Int64))
	items := Type("ItemChoice", OneOf(texts, numbers), func() {
		Untagged()
		Meta("http:encoding:sourceType", "array")
	})
	flag := Type("FlagUpload", func() {
		Attribute("flag", Boolean)
		Required("flag")
		Meta("openapi:additionalProperties", "false")
	})
	nested := Type("NestedUpload", OneOf(choice, flag), func() {
		Untagged()
	})
	media := Type("MediaUpload", OneOf(nested, String))
	Service("multipartchoices", func() {
		additional()
		Method("Nested", func() {
			Payload(nested)
			HTTP(func() {
				POST("/nested")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
		Method("Items", func() {
			Payload(func() {
				Attribute("value", items)
				Required("value")
			})
			HTTP(func() {
				POST("/items")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
		Method("WholeItems", func() {
			Payload(func() {
				Attribute("value", items, func() {
					Meta("http:encoding:sourceType", "absent")
				})
				Required("value")
				Meta("openapi:encoding:value:contentType", "application/json")
			})
			HTTP(func() {
				POST("/WholeItems")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
		Method("Metadata", func() {
			Payload(func() {
				Attribute("file", Bytes)
				Attribute("filename", String)
				Attribute("content_type", String)
				Required("file", "filename")
			})
			HTTP(func() {
				POST("/metadata")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
		Method("Media", func() {
			Payload(media)
			Result(media)
			HTTP(func() {
				POST("/media")
				MediaType("multipart/form-data", "NestedUpload")
				MediaType("application/json", "String")
				Response(StatusOK, func() {
					MediaType("multipart/form-data", "NestedUpload")
					MediaType("application/json", "String")
				})
			})
		})
		Method("Root", func() {
			Payload(choice)
			HTTP(func() {
				POST("/root")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
		Method("Optional", func() {
			Payload(choice)
			HTTP(func() {
				POST("/optional")
				MultipartRequest()
				OptionalRequestBody()
				Response(StatusNoContent)
			})
		})
		Method("Fields", func() {
			Payload(func() {
				Attribute("names", ArrayOf(String))
				Attribute("json", ArrayOf(String))
				Attribute("file", Bytes)
				Attribute("styled", ArrayOf(String))
				Meta("openapi:encoding:json:contentType", "application/json")
				Meta("openapi:encoding:styled:style", "form")
				Meta("openapi:encoding:styled:explode", "false")
			})
			HTTP(func() {
				POST("/fields")
				MultipartRequest()
				Response(StatusNoContent)
			})
		})
	})
}
