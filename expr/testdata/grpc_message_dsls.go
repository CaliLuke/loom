package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// GRPCRequestMessageWithMissingSecondAttribute lists a second request message attribute that the payload does not define.
var GRPCRequestMessageWithMissingSecondAttribute = func() {
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "first", String)
			})
			GRPC(func() {
				Message(func() {
					Attribute("first")
					Attribute("missing")
				})
			})
		})
	})
}

// GRPCRequestMessageWithUntaggedSecondAttribute lists a second request message attribute that has no rpc:tag.
var GRPCRequestMessageWithUntaggedSecondAttribute = func() {
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "first", String)
				Attribute("second", String)
			})
			GRPC(func() {
				Message(func() {
					Attribute("first")
					Attribute("second")
				})
			})
		})
	})
}

// GRPCRequestMessageWithDuplicateSecondTag lists a second request message attribute that reuses the first field number.
var GRPCRequestMessageWithDuplicateSecondTag = func() {
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "first", String)
				Field(1, "second", String)
			})
			GRPC(func() {
				Message(func() {
					Attribute("first")
					Attribute("second")
				})
			})
		})
	})
}

// GRPCRequestMessageWithMultipleAttributes lists several valid request message attributes.
var GRPCRequestMessageWithMultipleAttributes = func() {
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "first", String)
				Field(2, "second", String)
				Field(3, "third", String)
			})
			GRPC(func() {
				Message(func() {
					Attribute("first")
					Attribute("second")
					Attribute("third")
				})
			})
		})
	})
}

// GRPCResponseMessageWithMissingSecondAttribute lists a second response message attribute that the result does not define.
var GRPCResponseMessageWithMissingSecondAttribute = func() {
	Service("Service", func() {
		Method("Method", func() {
			Result(func() {
				Field(1, "first", String)
			})
			GRPC(func() {
				Response(CodeOK, func() {
					Message(func() {
						Attribute("first")
						Attribute("missing")
					})
				})
			})
		})
	})
}

// GRPCResponseMessageWithUntaggedSecondAttribute lists a second response message attribute that has no rpc:tag.
var GRPCResponseMessageWithUntaggedSecondAttribute = func() {
	Service("Service", func() {
		Method("Method", func() {
			Result(func() {
				Field(1, "first", String)
				Attribute("second", String)
			})
			GRPC(func() {
				Response(CodeOK, func() {
					Message(func() {
						Attribute("first")
						Attribute("second")
					})
				})
			})
		})
	})
}

// GRPCResponseMessageWithDuplicateSecondTag lists a second response message attribute that reuses the first field number.
var GRPCResponseMessageWithDuplicateSecondTag = func() {
	Service("Service", func() {
		Method("Method", func() {
			Result(func() {
				Field(1, "first", String)
				Field(1, "second", String)
			})
			GRPC(func() {
				Response(CodeOK, func() {
					Message(func() {
						Attribute("first")
						Attribute("second")
					})
				})
			})
		})
	})
}

// GRPCResponseMessageWithMultipleAttributes lists several valid response message attributes.
var GRPCResponseMessageWithMultipleAttributes = func() {
	Service("Service", func() {
		Method("Method", func() {
			Result(func() {
				Field(1, "first", String)
				Field(2, "second", String)
				Field(3, "third", String)
			})
			GRPC(func() {
				Response(CodeOK, func() {
					Message(func() {
						Attribute("first")
						Attribute("second")
						Attribute("third")
					})
				})
			})
		})
	})
}

var GRPCEndpointWithMixedResults = func() {
	Service("Service", func() {
		Method("Method", func() {
			Result(func() {
				Field(1, "done", Boolean)
			})
			StreamingResult(func() {
				Field(1, "line", String)
			})
			GRPC(func() {})
		})
	})
}

var GRPCEndpointWithStreamingResultOnly = func() {
	Service("Service", func() {
		Method("Method", func() {
			StreamingResult(func() {
				Field(1, "line", String)
			})
			GRPC(func() {})
		})
	})
}
