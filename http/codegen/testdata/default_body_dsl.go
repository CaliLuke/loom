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
