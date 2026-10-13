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
				Attribute("metadata", Any, func() {
					Default(map[any]any{int64(7): "seven", uint64(18446744073709551615): map[any]any{float32(1.2): "nested"}})
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

// OpenAPIAuthoredJSONTagsDSL extends the scalar-map specimen with explicit JSON
// tag omission choices and authored empty object-member examples.
var OpenAPIAuthoredJSONTagsDSL = func() {
	OpenAPIScalarMapKeysDSL()
	record := Type("TagRecord", func() {
		Attribute("plain", String, func() {
			Meta("struct:tag:json", "plain")
		})
		Attribute("items", ArrayOf(String), func() {
			Meta("struct:tag:json", "items,omitzero")
		})
		Attribute("omitted", String, func() {
			Meta("struct:tag:json", "omitted,omitempty")
		})
		Attribute("empty", String)
		Attribute("names", ArrayOf(String))
		Attribute("labels", MapOf(String, String))
		Attribute("flag", Boolean, func() {
			Default(true)
		})
		Attribute("count", Int, func() {
			Default(7)
		})
		Attribute("text", String, func() {
			Default("fallback")
		})
		Example(map[string]any{"plain": "", "items": []string{}, "omitted": "", "empty": "", "names": []string{}, "labels": map[string]string{}, "flag": false, "count": 0, "text": ""})
	})
	Service("tagged", func() {
		Method("echo", func() {
			NoSecurity()
			Payload(record)
			Result(record)
			HTTP(func() {
				POST("/tagged")
				Response(StatusOK)
			})
		})
	})
}
