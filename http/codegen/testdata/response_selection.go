package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// ResponseSelectionOrderDSL exercises overlapping tags, default placement, and
// mapped response bodies and headers without changing tagged declaration order.
var ResponseSelectionOrderDSL = func() {
	result := Type("SelectionResult", func() {
		Attribute("first", String)
		Attribute("second", String)
		Attribute("message", String)
		Attribute("trace", String)
		Required("first", "second", "message", "trace")
	})
	Service("selection", func() {
		Method("bodyless", func() {
			Result(func() {
				Attribute("kind", String)
			})
			HTTP(func() {
				GET("/bodyless")
				Response(StatusOK, func() {
					Body(Empty)
				})
				Response(StatusAccepted, func() {
					Tag("kind", "accepted")
					Body(Empty)
				})
			})
		})
		for _, method := range []struct {
			name     string
			statuses []int
		}{
			{"default_first", []int{StatusOK, StatusCreated, StatusAccepted}},
			{"default_middle", []int{StatusCreated, StatusOK, StatusAccepted}},
			{"default_last", []int{StatusCreated, StatusAccepted, StatusOK}},
		} {
			Method(method.name, func() {
				Result(result)
				HTTP(func() {
					GET("/" + method.name)
					for _, status := range method.statuses {
						Response(status, func() {
							switch status {
							case StatusCreated:
								Tag("first", "yes")
							case StatusAccepted:
								Tag("second", "yes")
							}
							Body("message")
							Header("trace:X-Trace")
							Header("first:X-First")
							Header("second:X-Second")
						})
					}
				})
			})
		}
	})
}

// ResponseStatusSelectionDSL exercises final selector ownership without Tags,
// including exact shadows, native class errors and an error-only endpoint.
func ResponseStatusSelectionDSL() {
	result := Type("StatusResult", func() {
		Attribute("status", Int, func() {
			Enum(201, 202, 299, 303, 500)
		})
		Attribute("message", String)
		Attribute("trace", String)
		Required("status", "message", "trace")
	})
	failure := Type("StatusFailure", func() {
		ErrorName("name", String)
		Attribute("status", Int)
		Attribute("message", String)
		Required("name", "status", "message")
	})
	branch := Type("BranchResult", func() {
		Attribute("status", Int, func() {
			Enum(201, 202)
		})
		Attribute("message", String)
		Attribute("entries", ArrayOf(String))
		Attribute("trace", String, func() {
			MinLength(4)
		})
		Attribute("session", String, func() {
			MinLength(2)
		})
		Required("status", "message", "entries", "trace", "session")
	})
	Service("status_selection", func() {
		responseStatusBranchMethod(branch)
		responseStatusCallMethod(result, failure)
		Method("fail", func() {
			Error("class", failure)
			HTTP(func() {
				GET("/fail")
				NoSuccessResponse()
				Response("class", Status4XX, func() {
					StatusField("status")
				})
			})
		})
	})
}

func responseStatusBranchMethod(branch expr.UserType) {
	Method("branch", func() {
		Result(branch)
		HTTP(func() {
			GET("/branch")
			for _, status := range []int{StatusCreated, StatusAccepted} {
				Response(status, func() {
					StatusField("status")
					Header("trace:X-Trace", func() {
						MinLength(5)
					})
					Cookie("session")
					if status == StatusCreated {
						Body("message")
					} else {
						Body("entries")
					}
				})
			}
		})
	})
}

func responseStatusCallMethod(result, failure expr.UserType) {
	Method("call", func() {
		Result(result)
		Error("exact", failure)
		Error("class", failure)
		Error("other", failure)
		HTTP(func() {
			GET("/call")
			for _, selector := range []any{StatusCreated, Status2XX, StatusFallback} {
				Response(selector, func() {
					StatusField("status")
					Header("trace:X-Trace")
				})
			}
			Response("exact", StatusUnauthorized, func() {
				StatusField("status")
			})
			for _, name := range []string{"class", "other"} {
				Response(name, Status4XX, func() {
					StatusField("status")
				})
			}
		})
	})
}
