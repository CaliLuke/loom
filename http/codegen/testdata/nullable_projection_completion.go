package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// NullableProjectionCompletionDSL covers selected child presence, metadata-only
// reconstruction and viewed source ownership alongside nullable tag fallback.
func NullableProjectionCompletionDSL() {
	metadata := Type("Metadata", func() {
		Attribute("header", String)
		Attribute("payload_error", String)
		Attribute("error_owner", String)
		Attribute("session", String, func() {
			Default("fallback")
		})
	})
	optional := Type("OptionalParent", func() {
		Attribute("value", String)
		Attribute("header", String)
	})
	viewed := ResultType("application/vnd.projection-completion", func() {
		TypeName("Viewed")
		Attributes(func() {
			Attribute("value", String)
			Attribute("header", String)
			Attribute("kind", String)
			Required("value")
		})
		View("default", func() {
			Attribute("value")
			Attribute("header")
			Attribute("kind")
		})
		View("other", func() {
			Attribute("value")
			Attribute("header")
			Attribute("kind")
		})
	})
	Service("completion", func() {
		nullableProjectionMetadataMethods(metadata)
		nullableProjectionOptionalMethods(optional)
		nullableProjectionViewedMethods(viewed)
	})
}

func nullableProjectionMetadataMethods(metadata expr.UserType) {
	for _, required := range []bool{false, true} {
		name := "metadata"
		if required {
			name = "required_metadata"
		}
		Method(name, func() {
			Result(metadata, func() {
				Nullable()
			})
			Error(name, metadata, func() {
				Nullable()
			})
			mapping := func() {
				Headers(func() {
					Header("header:X-Info")
					Header("payload_error:X-Payload")
					Header("error_owner:X-Owner")
					if required {
						Required("header")
					}
				})
				Cookie("session:Session")
			}
			HTTP(func() {
				GET("/" + name)
				Response(StatusOK, mapping)
				Response(name, StatusBadRequest, mapping)
			})
		})
	}
}

func nullableProjectionOptionalMethods(optional expr.UserType) {
	for _, nullable := range []bool{false, true} {
		name := "optional"
		if nullable {
			name = "nullable_optional"
		}
		Method(name, func() {
			Result(optional, func() {
				if nullable {
					Nullable()
				}
			})
			HTTP(func() {
				GET("/" + name)
				Response(StatusOK, func() {
					Header("header:X-Info")
					Body("value")
				})
			})
		})
	}
}

func nullableProjectionViewedMethods(viewed *expr.ResultTypeExpr) {
	for _, mode := range []string{"whole", "selected", "dynamic_selected", "tagged"} {
		Method(mode, func() {
			Result(viewed, func() {
				if mode == "selected" {
					View("default")
				}
				Nullable()
			})
			HTTP(func() {
				GET("/" + mode)
				Response(StatusOK, func() {
					if mode != "tagged" {
						Header("header:X-Info")
					}
					if mode == "selected" || mode == "dynamic_selected" {
						Body("value")
					}
				})
				if mode == "tagged" {
					Response(StatusAccepted, func() {
						Tag("kind", "accepted")
						Body("value")
					})
				}
			})
		})
	}
}
