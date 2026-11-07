package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

type parameterBinding struct {
	method, location, style string
	typ                     expr.DataType
	explode, required       bool
}

// ParameterStylesDSL covers the complete non-union parameter codec surface.
func ParameterStylesDSL() {
	API("parameter-styles", func() {
		Meta("openapi:version", "3.2")
	})
	record := Type("ParameterRange", func() {
		Attribute("R", String)
		Attribute("G", String)
		Required("R", "G")
	})
	bytesRecord := Type("ParameterBytes", func() {
		Attribute("R", String)
		Attribute("data", Bytes)
		Required("R")
	})
	defaults := Type("ParameterDefaults", func() {
		Attribute("R", String)
		Attribute("G", Int, func() { Default(7) })
		Required("R")
	})
	Service("styles", func() {
		for _, binding := range []parameterBinding{
			{"PathSimple", "path", "simple", String, false, true},
			{"PathMatrix", "path", "matrix", String, false, true},
			{"PathLabel", "path", "label", String, false, true},
			{"PathList", "path", "simple", ArrayOf(String), false, true},
			{"PathMatrixList", "path", "matrix", ArrayOf(String), true, true},
			{"PathLabelList", "path", "label", ArrayOf(String), false, true},
			{"PathLabelExploded", "path", "label", ArrayOf(String), true, true},
			{"PathObject", "path", "simple", record, true, true},
			{"PathMap", "path", "matrix", MapOf(String, String), true, true},
			{"QueryList", "query", "form", ArrayOf(String), false, true},
			{"QuerySpace", "query", "spaceDelimited", ArrayOf(String), false, true},
			{"QueryPipe", "query", "pipeDelimited", ArrayOf(String), false, true},
			{"QueryObject", "query", "deepObject", record, true, true},
			{"QueryMap", "query", "deepObject", MapOf(String, String), true, true},
			{"QueryResidual", "query", "form", MapOf(String, String), true, true},
			{"QueryJSON", "query", "", MapOf(String, String), false, true},
			{"QueryOptional", "query", "form", ArrayOf(String), true, false},
			{"QueryRequired", "query", "form", ArrayOf(String), true, true},
			{"QueryObjectPlain", "query", "form", record, false, true},
			{"CookieDefault", "cookie", "form", String, false, false},
			{"QueryListDefault", "query", "form", ArrayOf(String), false, false},
			{"QueryObjectDefault", "query", "deepObject", defaults, true, true},
			{"QueryObjectBytes", "query", "deepObject", bytesRecord, true, true},
			{"HeaderObject", "header", "simple", record, true, true},
			{"HeaderMap", "header", "simple", MapOf(String, String), false, true},
			{"HeaderMapExploded", "header", "simple", MapOf(String, String), true, true},
			{"CookieScalar", "cookie", "cookie", String, true, true},
			{"CookieList", "cookie", "cookie", ArrayOf(String), false, true},
			{"CookieExploded", "cookie", "cookie", ArrayOf(String), true, true},
			{"CookieMap", "cookie", "cookie", MapOf(String, String), true, true},
			{"CookieForm", "cookie", "form", ArrayOf(String), false, true},
		} {
			parameterBindingMethod(binding)
		}
		parameterFormMethod(record)
		Method("ResponseMap", func() {
			Result(func() {
				Attribute("labels", MapOf(String, String), func() {
					Meta("openapi:explode", "true")
				})
				Required("labels")
			})
			HTTP(func() {
				GET("/ResponseMap")
				Response(StatusNoContent, func() {
					Header("labels:X-Labels")
				})
			})
		})
	})
}

// ParameterStyles31DSL covers cookies whose raw and form wire domains coincide.
func ParameterStyles31DSL() {
	API("parameter-styles-31", func() {
		Meta("openapi:version", "3.1")
	})
	Service("styles31", func() {
		Method("Read", func() {
			Payload(func() {
				Attribute("count", Int)
				Attribute("flags", ArrayOf(Boolean))
				Attribute("labels", MapOf(Int, String), func() {
					Elem(func() {
						Enum("compact", "full")
					})
				})
				Required("count", "flags", "labels")
			})
			HTTP(func() {
				GET("/")
				Cookie("count")
				Cookie("flags", func() {
					Meta("openapi:explode", "false")
				})
				Cookie("labels", func() {
					Meta("openapi:explode", "false")
				})
				Response(StatusNoContent)
			})
		})
	})
}

func parameterBindingMethod(binding parameterBinding) {
	Method(binding.method, func() {
		Payload(func() {
			Attribute("value", binding.typ, func() {
				if binding.method == "CookieDefault" {
					Default("blue")
				}
				if binding.method == "QueryListDefault" {
					Default([]string{"blue"})
				}
			})
			if binding.method == "QueryResidual" || binding.method == "CookieMap" {
				Attribute("control", String)
				Required("control")
			}
			if binding.required {
				Required("value")
			}
		})
		HTTP(func() {
			path := "/" + binding.method
			if binding.location == "path" {
				path += "/{value}"
			}
			GET(path)
			metadata := func() {
				if binding.style != "" {
					Meta("openapi:style", binding.style)
					explode := "false"
					if binding.explode {
						explode = "true"
					}
					Meta("openapi:explode", explode)
				}
			}
			switch binding.location {
			case "path", "query":
				Param("value", metadata)
			case "header":
				Header("value:X-Value", metadata)
			case "cookie":
				Cookie("value", metadata)
			}
			if binding.method == "QueryResidual" {
				Param("control")
			}
			if binding.method == "CookieMap" {
				Cookie("control:sid", func() {
					if binding.style == "form" {
						Meta("openapi:style", "form")
					}
				})
			}
			Response(StatusNoContent)
		})
	})
}

func parameterFormMethod(record expr.DataType) {
	Method("FormFields", func() {
		Payload(func() {
			Attribute("filter", record)
			Attribute("tags", ArrayOf(String))
			Attribute("attributes", MapOf(String, String))
			Attribute("flags", MapOf(String, String))
			Attribute("facets", record)
			Attribute("jsonattrs", MapOf(String, String))
			Attribute("owner", String)
			Attribute("search", ArrayOf(String))
			Required("filter", "tags", "attributes", "flags", "facets", "jsonattrs", "owner", "search")
		})
		HTTP(func() {
			POST("/FormFields")
			Param("search", func() { Meta("openapi:explode", "false") })
			FormRequest()
			Body(func() {
				Attribute("filter")
				Attribute("tags")
				Attribute("attributes")
				Attribute("flags")
				Attribute("facets")
				Attribute("jsonattrs")
				Attribute("owner")
				Meta("openapi:encoding:filter:style", "deepObject")
				Meta("openapi:encoding:tags:style", "form")
				Meta("openapi:encoding:tags:explode", "false")
				Meta("openapi:encoding:attributes:contentType", "application/json")
				Meta("openapi:encoding:attributes:explode", "false")
				Meta("openapi:encoding:flags:allowReserved", "false")
				Meta("openapi:encoding:facets:style", "form")
			})
			Response(StatusNoContent)
		})
	})
}
