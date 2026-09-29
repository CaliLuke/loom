package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// BytesRepresentationDSL reuses one canonical named Bytes contract across
// builtin JSON, multipart, location and documentation-only body occurrences.
// Its recursive wrapper and named alias require complete schema variants;
// authored enum/default/example values remain the same service contract.
func BytesRepresentationDSL() {
	API("byte-representations", func() {
	})
	blob := Type("RepresentationBlob", Bytes, func() {
		Meta("openapi:typename", "PublicRepresentationBlob")
		Meta("openapi:typename:canonical", "true")
		MinLength(1)
		MaxLength(3)
		Enum([]byte("hi"), []byte("bye"))
		Default([]byte("hi"))
		Example([]byte("hi"))
	})
	alias := Type("RepresentationBlobAlias", blob)
	node := Type("RepresentationNode", func() {
		Attribute("data", blob)
		Attribute("next", "RepresentationNode")
		Example(map[string]any{"data": []byte("hi")})
	})
	Service("byte-representations", func() {
		bytesRepresentationJSON(blob, alias, node)
		bytesRepresentationLocations(blob)
		bytesRepresentationDocuments(blob)
		bytesRepresentationStreams(blob)
		bytesRepresentationResponseCodecs(blob)
	})
}

func bytesRepresentationJSON(blob, alias, node expr.UserType) {
	Method("json", func() {
		Payload(func() {
			Attribute("data", blob)
			Attribute("alias", alias)
		})
		Result(func() {
			Attribute("data", blob)
		})
		HTTP(func() {
			POST("/json")
			Response(StatusOK)
		})
	})
	Method("recursive", func() {
		Payload(node)
		Result(node)
		HTTP(func() {
			POST("/recursive")
			Response(StatusOK)
		})
	})
}

func bytesRepresentationLocations(blob expr.UserType) {
	Method("multipart", func() {
		Payload(func() {
			Attribute("data", blob)
		})
		HTTP(func() {
			POST("/multipart")
			MultipartRequest()
		})
	})
	Method("locations", func() {
		Payload(func() {
			Attribute("query", blob)
			Attribute("header", blob)
			Attribute("data", blob)
		})
		Result(func() {
			Attribute("header", blob)
			Attribute("data", blob)
		})
		HTTP(func() {
			POST("/locations")
			Param("query")
			Header("header:X-Blob")
			Response(StatusOK, func() {
				Header("header:X-Blob")
			})
		})
	})
}

func bytesRepresentationDocuments(blob expr.UserType) {
	for _, document := range []struct{ name, media string }{
		{"json_document", "application/json"},
		{"binary_document", "application/octet-stream"},
	} {
		Method(document.name, func() {
			HTTP(func() {
				POST("/" + document.name)
				SkipRequestBodyEncodeDecode()
				SkipResponseBodyEncodeDecode()
				OpenAPIRequestBody(blob, document.media, true)
				Response(StatusOK, func() {
					ContentType(document.media)
					OpenAPIBody(blob)
				})
			})
		})
	}
}

func bytesRepresentationStreams(blob expr.UserType) {
	Method("mapped_bytes", func() {
		StreamingResult(func() {
			Attribute("data", Bytes, func() {
				MaxLength(2)
			})
			Attribute("blob", blob)
			Attribute("event", String)
			Required("data", "blob", "event")
		})
		HTTP(func() {
			GET("/mapped-bytes")
			ServerSentEvents(func() {
				SSEEventData("data")
				SSEEventType("event")
			})
		})
	})
	Method("whole_bytes", func() {
		StreamingResult(Bytes, func() {
			MaxLength(2)
		})
		HTTP(func() {
			GET("/whole-bytes")
			ServerSentEvents()
		})
	})
}

func bytesRepresentationResponseCodecs(blob expr.UserType) {
	for _, response := range []struct{ name, media string }{
		{"static_json", "application/json"}, {"static_text", "text/plain"},
	} {
		Method(response.name, func() {
			if response.media == "text/plain" {
				// The public text-response DSL accepts native Bytes, not a named
				// Bytes declaration. Keep that authoring boundary unchanged.
				Result(Bytes, func() {
					MinLength(1)
					MaxLength(3)
				})
			} else {
				Result(blob)
			}
			HTTP(func() {
				GET("/" + response.name)
				Response(StatusOK, func() {
					ContentType(response.media)
				})
			})
		})
	}
	Method("media_header", func() {
		Result(func() {
			Attribute("data", blob)
			Attribute("media", String, func() {
				Enum("application/json", "text/plain", "application/gob")
			})
			Required("data", "media")
		})
		HTTP(func() {
			GET("/media-header")
			Response(StatusOK, func() {
				Body("data")
				Header("media:Content-Type")
			})
		})
	})
}
