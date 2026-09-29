package testdata

import . "github.com/CaliLuke/loom/dsl"

// BytesSchemaHTTPDSL exercises actual JSON byte decoders at every short length
// residue, one-sided bounds and unconstrained values. Byte fields are optional
// so present empty bytes exercise length rather than required-field validation.
func BytesSchemaHTTPDSL() {
	API("byteschema", func() {
	})
	blob := Type("SchemaBlob", Bytes, func() {
		MinLength(2)
		MaxLength(3)
		Meta("openapi:typename:canonical", "true")
	})
	chain := Type("SchemaBlobAlias", blob)
	Service("byteschema", func() {
		for _, name := range []string{"unbounded", "minimum", "maximum", "zero", "one", "two", "three", "four"} {
			Method(name, func() {
				Payload(func() {
					Attribute("data", Bytes, func() {
						switch name {
						case "minimum":
							MinLength(2)
						case "maximum":
							MaxLength(2)
						case "zero", "one", "two", "three", "four":
							length := map[string]int{"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4}[name]
							MinLength(length)
							MaxLength(length)
						}
					})
				})
				HTTP(func() {
					POST("/" + name)
				})
			})
		}
		for _, name := range []string{"named", "chain", "array", "map"} {
			Method(name, func() {
				Payload(func() {
					dataType := chain
					if name == "named" {
						dataType = blob
					}
					switch name {
					case "array":
						Attribute("data", ArrayOf(dataType))
					case "map":
						Attribute("data", MapOf(String, dataType))
					default:
						Attribute("data", dataType)
					}
				})
				HTTP(func() {
					POST("/" + name)
				})
			})
		}
	})
}
