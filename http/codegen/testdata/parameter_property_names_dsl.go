package testdata

import . "github.com/CaliLuke/loom/dsl"

// ParameterPropertyNamesDSL uses one record in text and JSON representations.
func ParameterPropertyNamesDSL() {
	parameterPropertyNamesDSL(true)
}

// ParameterPropertyNamesImportDSL isolates wire names from example annotations.
func ParameterPropertyNamesImportDSL() {
	parameterPropertyNamesDSL(false)
}

func parameterPropertyNamesDSL(examples bool) {
	API("parameter-property-names", func() {
		Meta("openapi:version", "3.2")
		if !examples {
			Meta("openapi:example", "false")
		}
	})
	record := Type("ParameterRecord", func() {
		Meta("openapi:typename", "PublicParameterRecord")
		Meta("openapi:typename:canonical", "true")
		Attribute("count", Int, func() {
			Meta("struct:tag:json:name", "wire")
			Meta("struct:tag:xml", "x")
		})
		Attribute("other", Int, func() {
			Meta("struct:tag:json", "alternate")
		})
		Required("count")
		if examples {
			Example(map[string]any{"count": 3, "other": 5})
		}
	})
	hidden := Type("TextOnlyRecord", func() {
		Attribute("count", Int)
		Attribute("hidden", Int, func() {
			Meta("struct:tag:json", "-")
		})
		Required("count")
	})
	Service("propertynames", func() {
		Method("hidden", func() {
			Payload(func() {
				Attribute("filter", hidden)
				Required("filter")
			})
			HTTP(func() {
				GET("/hidden")
				Param("filter", func() {
					Meta("openapi:style", "deepObject")
				})
			})
		})
		Method("exchange", func() {
			Payload(func() {
				Attribute("filter", record)
				Attribute("data", record)
				Required("filter", "data")
			})
			Result(func() {
				Attribute("metadata", record)
				Attribute("data", record)
				Required("metadata", "data")
			})
			HTTP(func() {
				POST("/exchange")
				Param("filter", func() {
					Meta("openapi:style", "deepObject")
				})
				Body("data")
				Response(StatusOK, func() {
					Header("metadata:X-Record")
					Body("data")
				})
			})
		})
		Method("form", func() {
			Payload(func() {
				Attribute("styled", record)
				Attribute("ordinary", record)
				Required("styled", "ordinary")
			})
			HTTP(func() {
				POST("/form")
				FormRequest()
				Body(func() {
					Attribute("styled")
					Attribute("ordinary")
					Meta("openapi:encoding:styled:style", "deepObject")
				})
			})
		})
	})
}
