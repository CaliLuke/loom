package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// RecursiveProjectionDSL exercises selected result types through every supported container.
func RecursiveProjectionDSL() {
	RecursiveProjectionTransportDSL("", false)
}

// RecursiveProjectionTransportDSL adds a chosen transport to the projection contract.
func RecursiveProjectionTransportDSL(transport string, unionFirst bool) {
	if transport == "jsonrpc" {
		API("projection", func() {
			JSONRPC(func() {

			})
		})
	}
	wrap := recursiveProjectionResult(unionFirst)
	Service("projection", func() {
		if transport == "jsonrpc" {
			JSONRPC(func() {
				POST("/rpc")
			})
		}
		Method("show", func() {
			Result(wrap)
			switch transport {
			case "http":
				HTTP(func() {
					GET("/show")
				})
			case "grpc":
				GRPC(func() {

				})
			case "jsonrpc":
				JSONRPC(func() {

				})
			}
		})
		if transport != "" {
			Method("watch", func() {
				StreamingResult(wrap)
				switch transport {
				case "http":
					HTTP(func() {
						GET("/watch")
						ServerSentEvents()
					})
				case "grpc":
					GRPC(func() {

					})
				case "jsonrpc":
					JSONRPC(func() {
						ServerSentEvents()
					})
				}
			})
		}
	})
}

func recursiveProjectionResult(unionFirst bool) *expr.ResultTypeExpr {
	owner := ResultType("application/vnd.projection.owner", func() {
		TypeName("Owner")
		Attributes(func() {
			Field(1, "id", String)
			Field(2, "secret", String)
			Required("id", "secret")
		})
		View("default", func() {
			Attribute("id")
		})
		View("full", func() {
			Attribute("id")
			Attribute("secret")
		})
	})
	pick := Type("Pick", func() {
		OneOf("choice", func() {
			Field(1, "owner", owner)
			Field(2, "number", Int)
		})
	})
	wrapper := Type("Holder", func() {
		Field(1, "owner", owner)
	})
	return ResultType("application/vnd.projection.wrap", func() {
		TypeName("Wrap")
		Attributes(func() {
			Field(1, "id", String)
			if unionFirst {
				Field(6, "pick", pick)
			}
			Field(2, "owner", owner)
			Field(3, "owners", CollectionOf(owner))
			Field(4, "items", ArrayOf(owner))
			Field(5, "by_key", MapOf(String, owner))
			if !unionFirst {
				Field(6, "pick", pick)
			}
			Field(7, "holder", wrapper)
			Field(8, "full_owner", owner)
			Field(9, "trailing", String)
			Field(10, "grouped", MapOf(String, ArrayOf(owner)))
			Required("id")
		})
		View("default", func() {
			Attribute("id")
			if unionFirst {
				Attribute("pick")
			}
			Attribute("owner")
			Attribute("owners")
			Attribute("items")
			Attribute("by_key")
			if !unionFirst {
				Attribute("pick")
			}
			Attribute("holder")
			Attribute("full_owner", func() {
				View("full")
			})
			Attribute("trailing")
			Attribute("grouped")
		})
	})
}
