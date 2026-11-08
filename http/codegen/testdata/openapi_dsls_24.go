package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

var (
	// OpenAPISharedErrorHeaderDSL exercises shared status codes with header identity.
	OpenAPISharedErrorHeaderDSL = func() {
		openAPISharedErrorDSL(false, false)
	}
	// OpenAPIStatusErrorNameDSL adds unique-status identity with the same error type.
	OpenAPIStatusErrorNameDSL = func() {
		openAPISharedErrorDSL(true, false)
	}
	// OpenAPIStatusSelectorDSL retains shared error bodies under class and fallback keys.
	OpenAPIStatusSelectorDSL = func() {
		openAPISharedErrorDSL(false, true)
	}
)

func openAPISharedErrorDSL(statusIdentity, selectors bool) {
	var ExceptionResponse = Type("ExceptionResponse", func() {
		ErrorName("loomError", String, "Which declared error this is.", func() {
			if statusIdentity {
				Meta("struct:field:name", "Kind")
			}
		})
		Attribute("error", String, func() {
			Meta("struct:field:name", "ErrorDetail")
		})
		Attribute("message", String, func() {
			if statusIdentity {
				MinLength(1)
			}
		})
		Attribute("path", String)
		if selectors {
			Attribute("status", Int)
			Required("status")
		} else {
			Attribute("status", Int32)
		}
		Attribute("timestamp", Int64)
		Attribute("details", String)
		Required("loomError")
	})

	API("shared-error-responses", func() {
		Meta("openapi:example", "false")
		Title("Shared Error Responses API")
		Description("Exercises repeated error responses whose routing field is carried by a header.")
		Version("1.0.0")
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("shared-error-responses", func() {
			Host("production", func() {
				URI("https://api.loom.design")
			})
		})
	})

	badRequest, forbidden := any(StatusBadRequest), any(StatusForbidden)
	if selectors {
		badRequest, forbidden = Status4XX, StatusFallback
	}
	Service("sharedErrorResponses", func() {
		Description("Returns one shared error body for every declared failure.")
		for _, methodName := range []string{"first", "second"} {
			Method(methodName, func() {
				NoSecurity()
				Error("bad_request", ExceptionResponse)
				Error("invalid_input", ExceptionResponse)
				Error("unauthorized", ExceptionResponse)
				Error("forbidden", ExceptionResponse)
				HTTP(func() {
					sharedErrorHTTPResponses(methodName, badRequest, forbidden, selectors)
				})
			})
		}
		if statusIdentity {
			statusErrorNameMethods(ExceptionResponse)
		}
	})
}

func sharedErrorHTTPResponses(methodName string, badRequest, forbidden any, selectors bool) {
	GET("/" + methodName)
	Response(StatusNoContent)
	Response("bad_request", badRequest, func() {
		Description("Invalid request for " + methodName + ".")
		Header("loomError:loom-error")
		if selectors {
			StatusField("status")
		}
	})
	Response("invalid_input", badRequest, func() {
		Description("Invalid input for " + methodName + ".")
		Header("loomError:loom-error")
		if selectors {
			StatusField("status")
		}
	})
	Response("unauthorized", StatusUnauthorized, func() {
		Header("loomError:loom-error")
		if selectors {
			StatusField("status")
		}
	})
	Response("forbidden", forbidden, func() {
		Header("loomError:loom-error")
		if selectors {
			StatusField("status")
		}
	})
}

func statusErrorNameMethods(errorType expr.UserType) {
	for _, mode := range []string{"body", "selected", "empty"} {
		Method("status_"+mode, func() {
			NoSecurity()
			Error("bad_request", errorType)
			Error("not_found", errorType)
			HTTP(func() {
				GET("/status/" + mode)
				Response(StatusNoContent)
				for _, failure := range []struct {
					name string
					code int
				}{
					{"bad_request", StatusBadRequest},
					{"not_found", StatusNotFound},
				} {
					Response(failure.name, failure.code, func() {
						ErrorNameFromStatus()
						switch mode {
						case "selected":
							Body("message")
						case "empty":
							Body(Empty)
						}
					})
				}
			})
		})
	}
}
