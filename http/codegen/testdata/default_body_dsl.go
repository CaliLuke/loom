package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// SelectedBodyDefaultsDSL exercises optional and required defaulted primitive
// attributes selected as HTTP bodies, including named and mapped attributes.
var SelectedBodyDefaultsDSL = func() {
	alias := Type("DefaultText", String, func() {
		Default("inherited")
	})
	Service("defaults", func() {
		Method("path", func() {
			Payload(func() {
				Attribute("value", String, func() {
					Default("path-default")
				})
			})
			HTTP(func() {
				GET("/path/{value}")
			})
		})

		for _, c := range []struct {
			name     string
			datatype any
			value    any
		}{
			{"text", String, "fallback"},
			{"empty", String, ""},
			{"number", Int, 7},
			{"zero", Int, 0},
			{"enabled", Boolean, true},
			{"disabled", Boolean, false},
			{"precise", Float32, 1.25},
			{"bytes", Bytes, []byte{0, 255}},
			{"alias", alias, nil},
			{"override", alias, "override"},
		} {
			for _, required := range []bool{false, true} {
				name := c.name
				if required {
					name += "_required"
				}
				Method(name, func() {
					Payload(func() {
						Attribute("value:wire", c.datatype, func() {
							if c.value != nil {
								Default(c.value)
							}
						})
						Attribute("q", String)
						if required {
							Required("value:wire")
						}
					})
					HTTP(func() {
						POST("/" + name)
						Param("q")
						Body("value")
					})
				})
			}
		}
	})
}

// CollectionBodyDefaultsDSL exercises collection defaults on CLI body flags,
// including nested maps, arrays, and a named boolean map key.
var CollectionBodyDefaultsDSL = func() {
	collectionBodyDefaultsDSL(false)
}

// JSONRPCCollectionBodyDefaultsDSL exercises collection defaults on JSON-RPC CLI flags.
var JSONRPCCollectionBodyDefaultsDSL = func() {
	collectionBodyDefaultsDSL(true)
}

func collectionBodyDefaultsDSL(rpc bool) {
	API("collectionDefaults", func() {
		if rpc {
			JSONRPC(func() {
			})
		}
	})
	key := Type("FlagKey", Boolean)
	Service("collections", func() {
		if rpc {
			JSONRPC(func() {
				POST("/rpc")
			})
		}
		for _, c := range []struct {
			name     string
			datatype any
			value    any
		}{
			{"array", ArrayOf(String), []string{"a", "b"}},
			{"string_map", MapOf(String, Int), map[string]int{"a": 1}},
			{"boolean_map", MapOf(Boolean, String), map[bool]string{true: "on"}},
			{"nested_map", MapOf(String, MapOf(Boolean, Int)), map[string]map[bool]int{"team": {false: 9}}},
			{"nested_array", ArrayOf(MapOf(Boolean, Boolean)), []map[bool]bool{{true: false}}},
			{"named_key", MapOf(key, String), map[bool]string{true: "on"}},
		} {
			Method(c.name, func() {
				Payload(func() {
					if rpc {
						ID("id", String)
					}
					Attribute("value", c.datatype, func() {
						Default(c.value)
					})
				})
				if rpc {
					JSONRPC(func() {
						Body("value")
					})
				} else {
					HTTP(func() {
						POST("/" + c.name)
						Body("value")
					})
				}
			})
		}
	})
}

// SelectedResponseStorageDSL exercises selected response reconstruction into
// service and view fields through both HTTP and JSON-RPC transports.
var SelectedResponseStorageDSL = func() {
	API("responsestorage", func() {
		JSONRPC(func() {
		})
	})
	text := map[string]any{
		"named":                 Type("Text", String),
		"namedoptional":         Type("OptionalText", String),
		"ordinarynamedoptional": Type("ServiceOptionalText", String),
	}
	names := []string{"required", "optional", "defaulted", "named", "namedoptional", "nullable", "nullableoptional", "number", "custom", "array", "bytes", "ordinary", "ordinaryoptional", "ordinarynullable", "ordinarynamedoptional", "whole"}
	for _, transport := range []string{"http", "jsonrpc"} {
		results := selectedResponseStorageTypes(transport, names, text)
		Service(transport+"fields", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			for _, name := range names {
				Method(name, func() {
					Result(results[name])
					response := func() {
						Response(StatusOK, func() {
							if name != "whole" {
								Body("value")
							}
						})
					}
					if transport == "jsonrpc" {
						JSONRPC(response)
					} else {
						HTTP(func() {
							GET("/" + name)
							response()
						})
					}
				})
			}
		})
	}
}

func selectedResponseStorageTypes(transport string, names []string, text map[string]any) map[string]any {
	results := make(map[string]any, len(names))
	for _, name := range names {
		fields := func() {
			selectedResponseStorageFields(name, text)
		}
		if name == "ordinary" || name == "ordinaryoptional" || name == "ordinarynullable" || name == "ordinarynamedoptional" {
			results[name] = Type(transport+name, fields)
		} else {
			results[name] = ResultType("application/vnd."+transport+name, func() {
				TypeName(transport + name)
				fields()
				View("default", func() {
					Attribute("value")
				})
			})
		}
	}
	return results
}

func selectedResponseStorageFields(name string, text map[string]any) {
	datatype := any(String)
	switch name {
	case "named", "namedoptional", "ordinarynamedoptional":
		datatype = text[name]
	case "number":
		datatype = Int
	case "array":
		datatype = ArrayOf(String)
	case "bytes":
		datatype = Bytes
	}
	Attribute("value", datatype, func() {
		if name == "nullable" || name == "nullableoptional" || name == "ordinarynullable" {
			Nullable()
		}
		if name == "defaulted" {
			Default("fallback")
		}
		if name == "custom" {
			Meta("struct:field:type", "time.Time", "time")
		}
	})
	if name != "ordinarynamedoptional" && name != "namedoptional" && name != "optional" && name != "defaulted" && name != "nullableoptional" && name != "ordinaryoptional" && name != "ordinarynullable" {
		Required("value")
	}
}
