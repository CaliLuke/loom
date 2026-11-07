package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// FormMapFieldsDSL covers the default field codecs of dynamically named entries.
func FormMapFieldsDSL() {
	record, arrays, words := formMapValueTypes()
	type entry struct {
		name  string
		value any
	}
	entries := []entry{
		{"scalar", String}, {"strings", ArrayOf(String)}, {"object", record},
		{"records", ArrayOf(record)}, {"nested", ArrayOf(ArrayOf(Int))},
		{"choice", arrays}, {"word", words},
	}
	for index := range entries {
		entries[index].value = Type("Map"+entries[index].name, MapOf(String, entries[index].value))
	}
	recordMap := entries[3].value
	root, media := formMapRootTypes(recordMap)
	Service("mapfields", func() {
		for _, entry := range entries {
			Method(entry.name, func() {
				Payload(entry.value)
				HTTP(func() {
					POST("/" + entry.name)
					FormRequest()
					Response(StatusNoContent)
				})
			})
		}
		Method("root", func() {
			Payload(root)
			HTTP(func() {
				POST("/root")
				FormRequest()
				Response(StatusNoContent)
			})
		})
		Method("optional", func() {
			Payload(func() {
				Attribute("value", entries[0].value)
			})
			HTTP(func() {
				POST("/optional")
				Body("value")
				FormRequest()
				Response(StatusNoContent)
			})
		})
		Method("media", func() {
			Payload(media)
			Result(media)
			HTTP(func() {
				POST("/media")
				MediaType("application/x-www-form-urlencoded", "Form")
				MediaType("text/plain", "Text")
				Response(StatusOK, func() {
					MediaType("application/x-www-form-urlencoded", "Form")
					MediaType("text/plain", "Text")
				})
			})
		})
	})
}

func formMapValueTypes() (expr.UserType, expr.UserType, expr.UserType) {
	record := Type("MapRecord", func() {
		Attribute("x", String)
		Required("x")
	})
	arrays := Type("MapArrays", OneOf(func() {
		Untagged()
		Attribute("Strings", ArrayOf(String, func() {
			Enum("plain")
		}))
		Attribute("Records", ArrayOf(record))
	}), func() {
		Meta("http:encoding:sourceType", "array")
	})
	words := Type("MapWords", OneOf(func() {
		Untagged()
		Attribute("First", String, func() {
			Enum("first")
		})
		Attribute("Second", String, func() {
			Enum("second")
		})
	}), func() {
		Meta("http:encoding:sourceType", "string")
	})
	return record, arrays, words
}

func formMapRootTypes(recordMap any) (expr.UserType, expr.UserType) {
	flag := Type("MapFlag", func() {
		Attribute("flag", Boolean)
		Required("flag")
	})
	root := Type("MapRoot", OneOf(func() {
		Untagged()
		Attribute("Maps", recordMap)
		Attribute("Flag", flag)
	}))
	media := Type("MapMedia", OneOf(func() {
		Attribute("Form", recordMap)
		Attribute("Text", String)
	}))
	return root, media
}
