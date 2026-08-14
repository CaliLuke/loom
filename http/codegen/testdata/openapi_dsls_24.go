package testdata

import . "github.com/CaliLuke/loom/dsl"

var OpenAPISharedErrorHeaderDSL = func() {
	var ExceptionResponse = Type("ExceptionResponse", func() {
		ErrorName("loomError", String, "Which declared error this is.")
		Attribute("error", String)
		Attribute("message", String)
		Attribute("path", String)
		Attribute("status", Int32)
		Attribute("timestamp", Int64)
		Attribute("details", String)
		Required("loomError")
	})

	API("shared-error-responses", func() {
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

	Service("sharedErrorResponses", func() {
		Description("Returns one shared error body for every declared failure.")
		for _, methodName := range []string{"first", "second"} {
			Method(methodName, func() {
				NoSecurity()
				Error("bad_request", ExceptionResponse)
				Error("unauthorized", ExceptionResponse)
				Error("forbidden", ExceptionResponse)
				HTTP(func() {
					GET("/" + methodName)
					Response(StatusNoContent)
					Response("bad_request", StatusBadRequest, func() {
						Header("loomError:loom-error")
					})
					Response("unauthorized", StatusUnauthorized, func() {
						Header("loomError:loom-error")
					})
					Response("forbidden", StatusForbidden, func() {
						Header("loomError:loom-error")
					})
				})
			})
		}
	})
}
