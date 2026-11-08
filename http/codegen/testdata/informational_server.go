package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// InformationalServerDSL extends the response-selection specimen with typed
// interim responses, an exact shadow of a class, and a headerless emission.
func InformationalServerDSL() {
	InformationalServer(true)
}

// InformationalServer selects either class-based progress responses or an
// exact-status control while retaining the same native and header contracts.
func InformationalServer(class bool) {
	label := informationalLabel("Label")
	nativeLabel := informationalLabel("LabelRequestBody")
	choice := Type("Choice", OneOf(nativeLabel, String), Untagged)
	input := informationalInput(label)
	hints := informationalHints(choice)
	progress := Type("Progress", func() {
		Attribute("status", Int, func() {
			Enum(100, 104, 105)
		})
		Attribute("trace", String, func() {
			MinLength(4)
		})
		Required("status", "trace")
	})
	marker := Type("Marker", func() {
	})
	continued := Type("Continued", func() {
		Attribute("status", Int, func() {
			Enum(100)
		})
		Attribute("trace", String, func() {
			MinLength(4)
		})
		Required("status")
	})
	result := Type("SelectionResult", func() {
		Attribute("message", String)
		Attribute("trace", String, func() {
			MinLength(4)
		})
		Required("message", "trace")
	})
	mapped := Type("MappedResult", func() {
		Attribute("trace", String, func() {
			MinLength(2)
		})
		Required("trace")
	})
	Service("selection", func() {
		informationalMappedMethod(mapped)
		Method("body", func() {
			Payload(input)
			HTTP(func() {
				POST("/body")
				Informational("continue", StatusContinue, marker)
				Response(StatusNoContent)
			})
		})
		Method("read", func() {
			Result(result)
			HTTP(func() {
				GET("/read")
				informationalResponses(hints, progress, marker, class)
				Informational("continue", StatusContinue, continued, func() {
					StatusField("status")
					Header("trace:X-Trace")
				})
				Response(StatusOK, func() {
					Body("message")
					Header("trace:X-Trace", func() {
						MinLength(2)
					})
				})
			})
		})
	})
}

func informationalMappedMethod(result expr.DataType) {
	Method("mapped", func() {
		Result(result)
		HTTP(func() {
			GET("/mapped")
			Response(StatusNoContent, func() {
				Header("trace:X-Trace", func() {
					MinLength(4)
				})
			})
		})
	})
}

func informationalInput(label expr.DataType) expr.UserType {
	headerLabel := Type("HTTPHeaderLabel", func() {
		Attribute("text", String, func() {
			MinLength(2)
		})
		Required("text")
	})
	return Type("Input", func() {
		Attribute("label", label)
		Attribute("headerLabel", headerLabel)
		Required("label", "headerLabel")
	})
}

func informationalHints(choice expr.DataType) expr.UserType {
	return Type("Hints", func() {
		Attribute("status", Int)
		Attribute("trace", String, func() {
			MinLength(2)
		})
		Attribute("labels", ArrayOf(String))
		Attribute("metadata", MapOf(String, String))
		Attribute("choice", choice)
		Required("status", "trace", "labels")
	})
}

func informationalResponses(hints, progress, marker expr.DataType, class bool) {
	Informational("hints", StatusEarlyHints, hints, func() {
		StatusField("status")
		Header("trace:X-Trace", func() {
			MinLength(4)
		})
		Header("labels:X-Labels")
		Header("metadata:X-Metadata")
		Header("choice:X-Choice")
	})
	selector := any(104)
	if class {
		selector = Status1XX
	}
	Informational("progress", selector, progress, func() {
		StatusField("status")
		Header("trace:X-Trace", func() {
			MinLength(2)
		})
	})
	Informational("marker", StatusProcessing, marker)
}

func informationalLabel(name string) expr.UserType {
	return Type(name, func() {
		Attribute("text", String, func() {
			MinLength(2)
		})
		Required("text")
	})
}
