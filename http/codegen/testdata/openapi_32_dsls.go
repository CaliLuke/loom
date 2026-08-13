package testdata

import . "github.com/CaliLuke/loom/dsl"

// OpenAPI32FeaturesDSL exercises OpenAPI 3.2-only path, tag, server, and identity fields.
var OpenAPI32FeaturesDSL = func() {
	var DeviceAuth = OAuth2Security("device", func() {
		Description("Device authorization")
		DeviceAuthorizationFlow("https://auth.example.com/device", "https://auth.example.com/token", "")
		Scope("catalog:read", "Read the catalog")
		Meta("openapi:oauth2MetadataUrl", "https://auth.example.com/.well-known/oauth-authorization-server")
		Meta("openapi:deprecated", "true")
	})
	var ExternalAuth = BasicAuthSecurity("external", func() {
		Meta("openapi:security:uri", "https://auth.example.com/security/external")
	})
	var Created = Type("CatalogCreated", func() {
		Attribute("title", String)
	})
	var Deleted = Type("CatalogDeleted", func() {
		Attribute("reason", String)
	})
	var Event = Type("CatalogEvent", func() {
		Meta("openapi:xml:name", "event")
		Meta("openapi:xml:namespace", "https://api.example.com/catalog")
		Meta("openapi:xml:prefix", "cat")
		Meta("openapi:xml:nodeType", "element")
		Attribute("id", String)
		OneOf("change", func() {
			Meta("openapi:discriminator:optional", "true")
			Meta("openapi:discriminator:defaultMapping", "#/components/schemas/CatalogCreated")
			Attribute("created", Created)
			Attribute("deleted", Deleted)
		})
		Required("id")
		Example("structured", func() {
			Value(Val{"id": "evt-1"})
			Meta("openapi:example:dataValue")
			Meta("openapi:example:serializedValue", `{"id":"evt-1"}`)
		})
	})
	var _ = API("oas32", func() {
		Meta("openapi:self", "https://api.example.com/openapi.json")
		Meta("openapi:tag:catalog:summary", "Catalog")
		Meta("openapi:tag:catalog:kind", "nav")
		Meta("openapi:tag:books:parent", "catalog")
		Server("production", func() {
			Host("primary", func() {
				URI("https://api.example.com")
			})
		})
	})
	Service("catalog", func() {
		Method("search", func() {
			Security(DeviceAuth, ExternalAuth, func() {
				Scope("catalog:read")
			})
			Payload(func() {
				Attribute("query", MapOf(String, String))
				AccessToken("access_token", String)
				Username("username", String)
				Password("password", String)
			})
			Result(Event)
			HTTP(func() {
				QUERY("/books")
				MapParams("query")
				Response(StatusOK, func() {
					ContentType("application/jsonl")
					Meta("openapi:summary", "Event stream")
					Meta("openapi:description:omit", "true")
				})
			})
		})
		Method("purge", func() {
			Result(String)
			HTTP(func() {
				Route("PURGE", "/books")
			})
		})
		Method("connect", func() {
			Result(String)
			HTTP(func() {
				CONNECT("/tunnel")
			})
		})
		Method("multipart", func() {
			Result(ArrayOf(Event), func() {
				Meta("openapi:itemSchema", "true")
				Meta("openapi:component:mediaType", "CatalogEventStream")
				Meta("openapi:prefixEncoding:0:contentType", "application/json")
				Meta("openapi:itemEncoding:contentType", "application/json")
				Meta("openapi:itemEncoding:encoding:id:style", "form")
				Meta("openapi:itemEncoding:prefixEncoding:0:contentType", "application/json")
				Meta("openapi:itemEncoding:itemEncoding:contentType", "application/json")
			})
			HTTP(func() {
				GET("/events")
				Response(StatusOK, func() {
					ContentType("multipart/mixed")
				})
			})
		})
		Method("parameters", func() {
			Payload(func() {
				Attribute("filter", String)
				Attribute("preferences", String)
			})
			Result(func() {
				Attribute("id", String)
				Attribute("cursor", String)
			})
			HTTP(func() {
				GET("/parameters")
				Header("filter:X-Filter", String, func() {
					Meta("openapi:allowReserved", "true")
				})
				Cookie("preferences", String, func() {
					Meta("openapi:allowReserved", "true")
					Meta("openapi:style", "cookie")
				})
				Response(StatusOK, func() {
					Header("cursor:X-Cursor", String, func() {
						Meta("openapi:allowReserved", "true")
					})
				})
			})
		})
		Method("example", func() {
			Result(Event)
			HTTP(func() {
				GET("/example")
			})
		})
	})
}
