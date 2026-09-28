package testdata

import . "github.com/CaliLuke/loom/dsl"

// ProtoGoFieldNamesDSL exercises method and getter reservations, optional
// field synthetic oneofs, map-entry wrapper names, and declaration order.
var ProtoGoFieldNamesDSL = func() {
	text := func(tag int, name string) {
		Field(tag, name, String, func() {
			MinLength(2)
		})
	}
	value := Type("NameCases", func() {
		for i, name := range []string{"reset", "descriptor", "string", "proto_message", "marshal", "unmarshal", "extension_range_array", "extension_map", "label", "get_label", "OAuth2Token", "name", "x_name", "get_pick"} {
			text(i+1, name)
		}
		Field(15, "foo", MapOf(String, String))
		OneOf("pick", func() {
			text(16, "foo_entry")
			text(17, "code")
		})
		text(18, "proto_reflect")
		text(19, "proto_reflect_field")
		Required("reset", "label", "OAuth2Token")
	})
	reverse := Type("ReverseCases", func() {
		OneOf("pick", func() {
			text(1, "code")
			text(2, "reset")
		})
		text(3, "get_pick")
		text(4, "get_label")
		text(5, "label")
		text(6, "name")
		text(7, "x_name")
		Required("name")
	})
	reserved := Type("ReservedChoice", func() {
		OneOf("reset", func() {
			text(1, "descriptor")
			text(2, "proto_message")
		})
		text(3, "label")
		Required("reset", "label")
	})
	Service("names", func() {
		for _, method := range []struct {
			name  string
			value any
		}{{"echo", value}, {"reverse", reverse}, {"choose", reserved}} {
			Method(method.name, func() {
				Payload(method.value)
				Result(method.value)
				GRPC(func() {})
			})
		}
	})
}
