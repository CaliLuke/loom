package testdata

import . "github.com/CaliLuke/loom/dsl"

// OpenAPIScalarMapKeysDSL declares maps keyed by booleans and integers in
// query parameters, request bodies, and results, with synthesized, authored,
// and default examples. OpenAPI renders their keys as JSON object member
// names.
var OpenAPIScalarMapKeysDSL = func() {
	var Rule = Type("Rule", func() {
		Attribute("max_count", Int, func() {
			Meta("struct:tag:json:name", "maxCount")
		})
		Required("max_count")
	})

	API("scalarMapKeys", func() {
		Title("Scalar Map Keys API")
		Version("1.0.0")
	})

	Service("flags", func() {
		Method("list", func() {
			NoSecurity()
			Payload(func() {
				Attribute("enabled", MapOf(Boolean, ArrayOf(Boolean)))
				Attribute("labels", MapOf(Boolean, String), func() {
					Default(map[bool]string{true: "on", false: "off"})
				})
				Attribute("groups", MapOf(String, MapOf(Boolean, Int)))
			})
			Result(MapOf(Int, String), func() {
				Example(map[int]string{1: "one", 20: "twenty"})
			})
			HTTP(func() {
				GET("/flags")
				Param("enabled")
				Param("labels")
				Param("groups")
				Response(StatusOK)
			})
		})
		Method("update", func() {
			NoSecurity()
			Payload(func() {
				Attribute("rules", MapOf(Boolean, Rule), func() {
					Example(map[bool]map[string]int{true: {"max_count": 3}})
				})
				Required("rules")
			})
			HTTP(func() {
				PUT("/flags")
				Response(StatusNoContent)
			})
		})
	})
}
