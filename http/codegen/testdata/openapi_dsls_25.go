package testdata

import . "github.com/CaliLuke/loom/dsl"

// OpenAPIVendorExtensionScopeDSL declares one vendor extension in every scope
// the OpenAPI importer emits, using the exact metadata keys it renders.
var OpenAPIVendorExtensionScopeDSL = func() {
	var Item = Type("Item", func() {
		Attribute("id", String)
		Required("id")
		Meta("openapi:schema:extension:x-schema-note", `{"audience":"public"}`)
	})

	API("vendorExtensions", func() {
		Title("Vendor Extensions API")
		Version("1.0.0")
		Meta("openapi:document:extension:x-migration-review", `{"status":"reviewed"}`)
		Meta("openapi:tag:catalog")
		Meta("openapi:tag:catalog:extension:x-color", `"blue"`)
	})

	Service("vendorExtensions", func() {
		Method("create", func() {
			NoSecurity()
			Payload(func() {
				Attribute("body", Item, func() {
					Meta("openapi:requestBody:extension:x-request-note", `"body-gated"`)
					Meta("openapi:mediaType:extension:x-media-note", `"json-only"`)
				})
				Required("body")
			})
			Result(Item)
			HTTP(func() {
				Meta("openapi:tag:catalog")
				POST("/items")
				Body("body")
				Response(StatusCreated)
			})
		})
		Method("show", func() {
			NoSecurity()
			Meta("openapi:extension:x-operation-note", `"gated"`)
			Payload(func() {
				Attribute("id", String, func() {
					Meta("openapi:parameter:extension:x-param-note", `"identifier"`)
					Meta("openapi:schema:extension:x-param-schema-note", `"uuid-ish"`)
				})
				Required("id")
			})
			Result(func() {
				Attribute("cursor", String, func() {
					Meta("openapi:header:extension:x-header-note", `"pagination"`)
				})
				Attribute("body", Item)
				Required("body")
			})
			HTTP(func() {
				Meta("openapi:tag:catalog")
				GET("/items/{id}")
				Response(StatusOK, func() {
					Meta("openapi:response:extension:x-response-note", `{"state":"primary"}`)
					Header("cursor:X-Cursor")
					Body("body")
				})
			})
		})
	})
}
