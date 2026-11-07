package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// EncodingSourceDSL retains named content application through normalization,
// including root alternatives, source-array item defaults and typeless values.
func EncodingSourceDSL() {
	record := Type("EncodingRecord", func() {
		Attribute("x", String)
		Required("x")
	})
	arrays := Type("EncodingArrays", OneOf(func() {
		Untagged()
		Attribute("strings", ArrayOf(String))
		Attribute("records", ArrayOf(record))
	}))
	fields := encodingSourceFields(arrays)
	roots := encodingSourceRoots(record)
	Service("encoding_source", func() {
		for _, media := range []string{"form", "multipart"} {
			for _, entry := range []struct {
				name string
				body any
			}{{"fields", fields}, {"roots", roots}} {
				Method(media+"_"+entry.name, func() {
					Payload(entry.body)
					HTTP(func() {
						POST("/" + media + "/" + entry.name)
						if media == "form" {
							FormRequest()
						} else {
							MultipartRequest()
						}
						if entry.name == "fields" {
							Body(func() {
								for _, name := range []string{"mixed", "whole", "singleton", "nested", "styled", "renamed"} {
									Attribute(name)
								}
								Meta("openapi:encoding:whole:contentType", "application/example+json; charset=utf-8")
								Meta("openapi:encoding:singleton:contentType", "application/json")
								Meta("openapi:encoding:styled:contentType", "application/json")
								Meta("openapi:encoding:styled:allowReserved", "false")
								Meta("openapi:encoding:renamed:contentType", "text/plain; charset=utf-8")
							})
						}
						Response(StatusNoContent)
					})
				})
			}
		}
	})
}

func encodingSourceFields(arrays expr.UserType) expr.UserType {
	fields := Type("EncodingFields", func() {
		Attribute("mixed", arrays, func() {
			Meta("http:encoding:sourceType", "array")
		})
		Attribute("whole", arrays)
		Attribute("singleton", ArrayOf(String), func() {
			Meta("http:encoding:sourceType", "absent")
		})
		Attribute("nested", ArrayOf(ArrayOf(Int)))
		Attribute("styled", ArrayOf(Int))
		Attribute("renamed", String, func() {
			Meta("struct:tag:form", "wire")
			Meta("struct:tag:json", "jsonName")
		})
	})
	return fields
}

func encodingSourceRoots(record expr.UserType) expr.UserType {
	left := Type("EncodingLeft", func() {
		Attribute("kind", String, func() {
			Enum("left")
		})
		Required("kind")
		Attribute("value", ArrayOf(String))
	})
	right := Type("EncodingRight", func() {
		Attribute("kind", String, func() {
			Enum("right")
		})
		Required("kind")
		Attribute("value", ArrayOf(record))
	})
	roots := Type("EncodingRoots", OneOf(func() {
		Untagged()
		Attribute("nested", OneOf(func() {
			Untagged()
			Attribute("left", left)
			Attribute("right", right)
		}))
	}))
	return roots
}
