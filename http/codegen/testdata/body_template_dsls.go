package testdata

import . "github.com/CaliLuke/loom/dsl"

// BodyTemplateInheritanceDSL selects effective fields from both inline and
// named composed payloads, including an explicit local template override.
var BodyTemplateInheritanceDSL = func() {
	API("template-api", func() {
		Title("Body template inheritance")
		Description("Effective selected body contracts")
	})
	base := Type("TemplateBase", func() {
		Attribute("count", Int, func() {
			Maximum(10)
			Default(7)
		})
		Attribute("name", String)
	})
	middle := Type("TemplateMiddle", func() {
		Extend(base)
		Attribute("count", func() {
			Minimum(1)
		})
	})
	defaults := bodyTemplateDefaultMethods()
	Service("templates", func() {
		for _, name := range []string{"inline", "named", "override"} {
			Method(name, func() {
				if name == "named" {
					Payload(middle)
				} else {
					Payload(func() {
						Extend(middle)
					})
				}
				HTTP(func() {
					POST("/" + name)
					Body(func() {
						Attribute("count:n", func() {
							if name == "override" {
								Minimum(0)
								Default(9)
							}
						})
					})
				})
			})
		}
		defaults()
	})
}

// bodyTemplateDefaultMethods keeps source materialization separate from the
// payload's default policy and supplied-value admission.
func bodyTemplateDefaultMethods() func() {
	scalar := Type("TemplateNestedScalar", func() {
		Attribute("count", Int, func() {
			Default(7)
		})
	})
	return func() {
		bodyTemplateSourceOnlyMethod()
		bodyTemplateNestedMethod()
		Method("nested_same", func() {
			Payload(func() {
				Attribute("object", scalar)
			})
			HTTP(func() {
				POST("/nested_same")
				Body(func() {
					Attribute("object")
				})
			})
		})
		bodyTemplateRequiredMethod()
		bodyTemplateNullableMethod()
	}
}

func bodyTemplateSourceOnlyMethod() {
	Method("source_only", func() {
		Payload(func() {
			Attribute("count", Int)
		})
		HTTP(func() {
			POST("/source_only")
			Body(func() {
				Attribute("count", func() {
					Default(9)
				})
			})
		})
	})
}

func bodyTemplateNestedMethod() {
	Method("nested", func() {
		Payload(func() {
			Attribute("object", func() {
				Attribute("count", Int, func() {
					Default(7)
				})
				Attribute("labels", ArrayOf(String), func() {
					Default([]string{"payload"})
				})
				Default(map[string]any{})
			})
		})
		HTTP(func() {
			POST("/nested")
			Body(func() {
				Attribute("object", func() {
					Attribute("count", func() {
						Default(9)
					})
					Attribute("labels", func() {
						Default([]string{"body"})
					})
				})
			})
		})
	})
}

func bodyTemplateRequiredMethod() {
	Method("required", func() {
		Payload(func() {
			Attribute("count", Int, func() {
				Default(7)
			})
			Required("count")
		})
		HTTP(func() {
			POST("/required")
			Body(func() {
				Attribute("count", func() {
					Default(9)
				})
			})
		})
	})
}

func bodyTemplateNullableMethod() {
	Method("nullable", func() {
		Payload(func() {
			Attribute("value", String, func() {
				Nullable()
				Default("payload")
			})
		})
		HTTP(func() {
			POST("/nullable")
			Body(func() {
				Attribute("value", func() {
					Default("body")
				})
			})
		})
	})
}
