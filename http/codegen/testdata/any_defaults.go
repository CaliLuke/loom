package testdata

import . "github.com/CaliLuke/loom/dsl"

// HTTPAnyDefaultsDSL exercises raw JSON defaults in HTTP object properties.
var HTTPAnyDefaultsDSL = func() {
	anyDefaultsDSL(false)
}

// JSONRPCAnyDefaultsDSL exercises the same properties in JSON-RPC params.
var JSONRPCAnyDefaultsDSL = func() {
	anyDefaultsDSL(true)
}

func anyDefaultsDSL(rpc bool) {
	API("anydefaults", func() {
		if rpc {
			JSONRPC(func() {})
		}
	})
	alias := Type("Metadata", Any)
	Service("defaults", func() {
		if rpc {
			JSONRPC(func() {
				POST("/rpc")
			})
		}
		for _, c := range []struct {
			name     string
			datatype any
			value    any
			required bool
			nullable bool
		}{
			{"text", Any, "fallback", false, false},
			{"named", alias, "fallback", false, false},
			{"boolean", Any, false, false, false},
			{"number", Any, 0, false, false},
			{"empty", Any, "", false, false},
			{"object", Any, map[string]any{"key": "value"}, false, false},
			{"array", Any, []any{"value"}, false, false},
			{"nullable", Any, "fallback", false, true},
			{"nodefault", Any, nil, false, false},
			{"required", Any, "fallback", true, false},
		} {
			Method(c.name, func() {
				Payload(func() {
					if rpc {
						ID("id", String)
					}
					Attribute("metadata:wire", c.datatype, func() {
						if c.value != nil {
							Default(c.value)
						}
						if c.nullable {
							Nullable()
						}
					})
					if c.required {
						Required("metadata:wire")
					}
				})
				if rpc {
					JSONRPC(func() {})
				} else {
					HTTP(func() {
						POST("/" + c.name)
					})
				}
			})
		}
	})
}
