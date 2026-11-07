package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// untaggedJSONProfileMethods mirrors the runtime closure table at generated boundaries.
func untaggedJSONProfileMethods() func() {
	x := Type("ProfileX", func() {
		Attribute("x", Int)
		Required("x")
	})
	y := Type("ProfileY", func() {
		Attribute("y", Int)
	})
	kind := Type("ProfileKind", func() {
		Attribute("kind", String, func() {
			Enum("a")
		})
		Required("kind")
	})
	additional := Type("ProfileAdditional", func() {
		Attribute("x", Int)
		Required("x")
		AdditionalAttributes(Int)
	})
	explicitOpen := Type("ProfileExplicitOpen", func() {
		AdditionalAttributes(Any)
	})
	closed := Type("ProfileClosed", func() {
		Attribute("x", Int)
		Required("x")
		Meta("openapi:additionalProperties", "false")
	})
	nested := Type("ProfileNested", OneOf(x, String), Untagged)
	nestedNumber := Type("ProfileNestedNumber", OneOf(Int, String), Untagged)
	profiles := []struct {
		name string
		typ  expr.DataType
	}{
		{"nested_three", Type("ProfileThree", OneOf(nestedNumber, Int, Float64), Untagged)},
		{"closure_open_pair", Type("ProfileOpenPair", OneOf(x, y), Untagged)},
		{"closure_discriminated", Type("ProfileDiscriminated", OneOf(x, kind), Untagged)},
		{"closure_map", Type("ProfileMap", OneOf(func() {
			Untagged()
			Attribute("Map", MapOf(String, Int))
			Attribute("Text", String)
		}))},
		{"closure_any", Type("ProfileAny", OneOf(Any, String), Untagged)},
		{"closure_nested", Type("ProfileOuter", OneOf(nested, Boolean), Untagged)},
		{"closure_additional", Type("ProfileAdditionalChoice", OneOf(additional, String), Untagged)},
		{"closure_explicit_open", Type("ProfileExplicitOpenChoice", OneOf(explicitOpen, String), Untagged)},
		{"closure_closed", Type("ProfileClosedChoice", OneOf(closed, String), Untagged)},
	}
	errorMethod := untaggedErrorEnvelopeMethod()
	return func() {
		for _, profile := range profiles {
			Method(profile.name, func() {
				occurrence := func() {
					if profile.name != "nested_three" {
						Meta("openapi:unevaluatedProperties", "false")
					}
				}
				Payload(profile.typ, occurrence)
				Result(profile.typ, occurrence)
				HTTP(func() {
					POST("/" + profile.name)
				})
			})
		}
		errorMethod()
	}
}

func untaggedErrorEnvelopeMethod() func() {
	choice := Type("ProfileErrorChoice", OneOf(String, Int), Untagged)
	envelope := Type("ProfileErrorEnvelope", func() {
		Attribute("body", choice)
		Attribute("reason", String)
		Required("body", "reason")
	})
	return func() {
		Method("error_envelope", func() {
			Payload(String)
			Error("profile_error", envelope)
			HTTP(func() {
				POST("/error_envelope")
				Response("profile_error", StatusBadRequest, func() {
					Body("body")
					Header("reason:X-Reason")
				})
			})
		})
	}
}
