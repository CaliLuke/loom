package testdata

import . "github.com/CaliLuke/loom/dsl"

// AccessLogCredentialsDSL exercises query and path credentials on mixed and
// disjoint HTTP/JSON-RPC example servers, including an ordinary control service.
func AccessLogCredentialsDSL() {
	key := APIKeySecurity("permit")
	bearer := JWTSecurity("bearer")
	API("accesslogs", func() {
		JSONRPC(func() {
		})
		for _, server := range []struct {
			name     string
			services []string
		}{
			{"mixed", []string{"web", "rpc"}},
			{"webonly", []string{"web"}},
			{"rpconly", []string{"rpc"}},
			{"ordinary", []string{"ordinary"}},
		} {
			Server(server.name, func() {
				Services(server.services...)
				Host("local", func() {
					URI("http://localhost:8080")
				})
			})
		}
	})
	Service("web", func() {
		for _, mode := range []string{"query", "path"} {
			Method(mode, func() {
				if mode == "path" {
					Meta("openapi:generate", "false")
					Security(bearer)
					Payload(func() {
						Token("credential", String)
						Required("credential")
					})
				} else {
					Security(key)
					Payload(func() {
						APIKey("permit", "credential", String)
						Required("credential")
					})
				}
				HTTP(func() {
					if mode == "path" {
						GET("/path/{credential}")
					} else {
						GET("/query")
						Param("credential:ticket")
					}
				})
			})
		}
	})
	Service("rpc", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("query", func() {
			Security(key)
			Payload(func() {
				ID("id")
				APIKey("permit", "credential", String)
				Required("credential")
			})
			Result(func() {
				ID("id")
			})
			JSONRPC(func() {
				Param("credential:proof")
			})
		})
	})
	Service("ordinary", func() {
		Method("show", func() {
			HTTP(func() {
				GET("/ordinary")
			})
		})
	})
}
