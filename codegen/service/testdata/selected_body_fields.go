package testdata

import . "github.com/CaliLuke/loom/dsl"

// SelectedBodyFieldsDSL exercises one declared field selected as a transport
// body, keeping its logical DSL name distinct from the generated Go member.
func SelectedBodyFieldsDSL(transport, direction, shape string, renamed bool) func() {
	return func() {
		fields := selectedBodyFields(shape, renamed)
		failure := Type("Failure", fields)
		var viewed any
		if direction == "viewed" {
			viewed = ResultType("application/vnd.fields", func() {
				fields()
				View("default", func() {
					Attribute("value")
				})
			})
		}
		API("fields", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
				})
			}
		})
		Service("fields", func() {
			if transport == "jsonrpc" {
				JSONRPC(func() {
					POST("/")
				})
			}
			Method("call", func() {
				switch direction {
				case "request", "whole":
					Payload(fields)
				case "response":
					Result(fields)
				case "viewed":
					Result(viewed)
				case "error":
					Error("failed", failure)
				}
				wire := selectedBodyWire(direction)
				if transport == "jsonrpc" {
					JSONRPC(wire)
				} else {
					HTTP(func() {
						POST("/")
						wire()
					})
				}
			})
		})
	}
}

func selectedBodyFields(shape string, renamed bool) func() {
	named := Type("Text", String)
	value := any(String)
	switch shape {
	case "named":
		value = named
	case "array":
		value = ArrayOf(String)
	case "object":
		value = Type("Record", func() {
			Attribute("text", String)
		})
	}
	return func() {
		Attribute("value", value, func() {
			if renamed {
				Meta("struct:field:name", "ActualValue")
			}
			if shape == "nullable" {
				Nullable()
			}
			if shape == "defaulted" {
				Default("fallback")
			}
		})
		if shape != "optional" && shape != "defaulted" {
			Required("value")
		}
	}
}

func selectedBodyWire(direction string) func() {
	return func() {
		switch direction {
		case "request":
			Body("value")
		case "response", "viewed":
			Response(StatusOK, func() {
				Body("value")
			})
		case "error":
			Response("failed", StatusBadRequest, func() {
				Body("value")
			})
		}
	}
}
