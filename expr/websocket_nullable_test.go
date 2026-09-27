package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestWebSocketRejectsNullablePayloadRoot(t *testing.T) {
	cases := []struct {
		name  string
		shape func() expr.DataType
	}{
		{"string", func() expr.DataType {
			return String
		}},
		{"object", func() expr.DataType {
			return Type("Leaf", func() {
				Attribute("value", String)
			})
		}},
		{"array", func() expr.DataType {
			return ArrayOf(String)
		}},
		{"map", func() expr.DataType {
			return MapOf(String, String)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, alias := range []bool{false, true} {
				for _, bidirectional := range []bool{false, true} {
					err := expr.RunInvalidDSL(t, func() {
						payload := Type("NullablePayload", c.shape(), func() {
							Nullable()
						})
						if alias {
							payload = Type("Alias", payload)
						}
						Service("stream", func() {
							Method("send", func() {
								StreamingPayload(payload)
								if bidirectional {
									StreamingResult(String)
								} else {
									Result(String)
								}
								HTTP(func() {
									GET("/stream")
								})
							})
						})
					})
					require.ErrorContains(t, err, "HTTP WebSocket streaming payload root cannot be nullable", "alias=%t bidirectional=%t", alias, bidirectional)
				}
			}
		})
	}
}

func TestWebSocketAllowsNestedNullablePayload(t *testing.T) {
	expr.RunDSL(t, func() {
		leaf := Type("Leaf", String, func() {
			Nullable()
		})
		Service("stream", func() {
			Method("send", func() {
				StreamingPayload(func() {
					Attribute("value", leaf)
					Attribute("list", ArrayOf(String, func() {
						Nullable()
					}))
					Attribute("map", MapOf(String, String, func() {
						Elem(func() {
							Nullable()
						})
					}))
				})
				StreamingResult(String)
				HTTP(func() {
					GET("/stream")
				})
			})
		})
	})
}
