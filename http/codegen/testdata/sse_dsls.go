package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var SSEStringDSL = func() {
	Service("SSEStringService", func() {
		Method("SSEStringMethod", func() {
			StreamingResult(String)
			HTTP(func() {
				GET("/string")
				ServerSentEvents()
			})
		})
	})
}

var SSEIntDSL = func() {
	Service("SSEIntService", func() {
		Method("SSEIntMethod", func() {
			StreamingResult(Int)
			HTTP(func() {
				GET("/int")
				ServerSentEvents()
			})
		})
	})
}

var SSEBoolDSL = func() {
	Service("SSEBoolService", func() {
		Method("SSEBoolMethod", func() {
			StreamingResult(Boolean)
			HTTP(func() {
				GET("/bool")
				ServerSentEvents()
			})
		})
	})
}

var SSEObjectDSL = func() {
	Service("SSEObjectService", func() {
		Method("SSEObjectMethod", func() {
			StreamingResult(func() {
				Attribute("id", String)
				Attribute("value", Int)
				Attribute("flag", Boolean)
			})
			HTTP(func() {
				GET("/object")
				ServerSentEvents()
			})
		})
	})
}

var SSEDataFieldDSL = func() {
	Service("SSEDataFieldService", func() {
		Method("SSEDataFieldMethod", func() {
			StreamingResult(func() {
				Attribute("data", String)
				Attribute("flag", Boolean)
			})
			HTTP(func() {
				GET("/data-field")
				ServerSentEvents("data")
			})
		})
	})
}

var SSEDataIDFieldDSL = func() {
	Service("SSEDataIDFieldService", func() {
		Method("SSEDataIDFieldMethod", func() {
			StreamingResult(func() {
				Attribute("data", String)
				Attribute("id", String)
			})
			HTTP(func() {
				GET("/data-id-field")
				ServerSentEvents("data", func() {
					SSEEventID("id")
				})
			})
		})
	})
}

var SSERequestIDDSL = func() {
	Service("SSERequestIDService", func() {
		Method("SSERequestIDMethod", func() {
			Payload(func() {
				Attribute("id", String)
			})
			StreamingResult(String)
			HTTP(func() {
				GET("/request-id")
				ServerSentEvents(func() {
					SSERequestID("id")
				})
			})
		})
	})
}

var SSEAllFieldsDSL = func() {
	Service("SSEAllFieldsService", func() {
		Method("SSEAllFieldsMethod", func() {
			Payload(func() {
				Attribute("id", String)
			})
			StreamingResult(func() {
				Attribute("id", String, func() {
					Example("123")
				})
				Attribute("event", String, func() {
					Example("update")
				})
				Attribute("retry", Int, func() {
					Example(3000)
				})
				Attribute("data", func() {
					Attribute("message", String, func() {
						Example("Hello, world!")
					})
				})
			})
			HTTP(func() {
				GET("/all-fields")
				ServerSentEvents(func() {
					SSERequestID("id")
					SSEEventID("id")
					SSEEventType("event")
					SSEEventRetry("retry")
					SSEEventData("data")
				})
			})
		})
	})
}

// SSEResultTypesDSL declares one SSE method per primitive, collection, and
// user-type streaming result so generated server and client code can be
// compiled and exercised for each event shape.
var SSEResultTypesDSL = func() {
	var Note = Type("Note", func() {
		Attribute("text", String)
		Required("text")
	})
	Service("SSEResultTypes", func() {
		Method("StreamString", func() {
			StreamingResult(String)
			HTTP(func() {
				GET("/string")
				ServerSentEvents()
			})
		})
		Method("StreamInt", func() {
			StreamingResult(Int)
			HTTP(func() {
				GET("/int")
				ServerSentEvents()
			})
		})
		Method("StreamBool", func() {
			StreamingResult(Boolean)
			HTTP(func() {
				GET("/bool")
				ServerSentEvents()
			})
		})
		Method("StreamBytes", func() {
			StreamingResult(Bytes)
			HTTP(func() {
				GET("/bytes")
				ServerSentEvents()
			})
		})
		Method("StreamAny", func() {
			StreamingResult(Any)
			HTTP(func() {
				GET("/any")
				ServerSentEvents()
			})
		})
		Method("StreamArray", func() {
			StreamingResult(ArrayOf(String))
			HTTP(func() {
				GET("/array")
				ServerSentEvents()
			})
		})
		Method("StreamMap", func() {
			StreamingResult(MapOf(String, Int))
			HTTP(func() {
				GET("/map")
				ServerSentEvents()
			})
		})
		Method("StreamNote", func() {
			StreamingResult(Note)
			HTTP(func() {
				GET("/note")
				ServerSentEvents()
			})
		})
	})
}

// SSEFieldPresenceDSL maps the SSE id, event, and retry fields from required,
// optional, and defaulted result attributes so generated server encoding and
// client decoding can be compiled and exercised for each presence mode.
var SSEFieldPresenceDSL = func() {
	Service("SSEFieldPresence", func() {
		Method("StreamRequired", func() {
			StreamingResult(func() {
				Attribute("id", String)
				Attribute("event", String)
				Attribute("retry", Int)
				Attribute("text", String)
				Required("id", "event", "retry", "text")
			})
			HTTP(func() {
				GET("/required")
				ServerSentEvents(func() {
					SSEEventID("id")
					SSEEventType("event")
					SSEEventRetry("retry")
					SSEEventData("text")
				})
			})
		})
		Method("StreamOptional", func() {
			StreamingResult(func() {
				Attribute("id", String)
				Attribute("event", String)
				Attribute("retry", Int64)
				Attribute("text", String)
				Required("text")
			})
			HTTP(func() {
				GET("/optional")
				ServerSentEvents(func() {
					SSEEventID("id")
					SSEEventType("event")
					SSEEventRetry("retry")
					SSEEventData("text")
				})
			})
		})
		Method("StreamDefaulted", func() {
			StreamingResult(func() {
				Attribute("id", String, func() {
					Default("default-id")
				})
				Attribute("event", String, func() {
					Default("default-event")
				})
				Attribute("retry", UInt, func() {
					Default(1500)
				})
				Attribute("text", String)
				Required("text")
			})
			HTTP(func() {
				GET("/defaulted")
				ServerSentEvents(func() {
					SSEEventID("id")
					SSEEventType("event")
					SSEEventRetry("retry")
					SSEEventData("text")
				})
			})
		})
	})
}
