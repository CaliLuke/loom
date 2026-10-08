package testdata

import . "github.com/CaliLuke/loom/dsl"

// PathParamsDSL combines typed URL captures with JSON-RPC body parameters and
// includes a path-only method whose request must omit params.
var PathParamsDSL = func() {
	API("paths", func() { JSONRPC(func() {}) })
	Service("paths", func() {
		JSONRPC(func() {
			POST("/orgs/{organization}/regions/{region}/labels/{labels}/rpc")
			Param("organization_id:organization")
			Param("region")
			Param("labels")
		})
		for _, name := range []string{"body", "path"} {
			Method(name, func() {
				Payload(func() {
					if name == "body" {
						ID("id", String)
					}
					Attribute("organization_id", Int, func() { Meta("struct:field:name", "Org") })
					Attribute("region", String)
					Attribute("labels", ArrayOf(String))
					Required("organization_id", "region", "labels")
					if name == "body" {
						Attribute("name", String)
						Required("name")
					}
				})
				Result(String)
				JSONRPC(func() {})
			})
		}
	})
}
