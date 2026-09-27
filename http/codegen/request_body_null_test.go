package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestRequestBodyNullability checks that requiredness does not change the
// emitted root null policy, including for named aliases.
func TestRequestBodyNullability(t *testing.T) {
	for _, tc := range []struct {
		name      string
		attribute func()
		allows    bool
	}{
		{"string", func() {
			Attribute("body", String)
		}, false},
		{"array", func() {
			Attribute("body", ArrayOf(String))
		}, false},
		{"map", func() {
			Attribute("body", MapOf(String, Int))
		}, false},
		{"object", func() {
			Attribute("body", func() {
				Attribute("name", String)
			})
		}, false},
		{"nullable", func() {
			Attribute("body", String, func() {
				Nullable()
			})
		}, true},
		{"any", func() {
			Attribute("body", Any)
		}, true},
		{"any alias", func() {
			Attribute("body", "AnyValue")
		}, true},
		{"nullable alias", func() {
			Attribute("body", "NullableValue")
		}, true},
	} {
		for _, required := range []bool{false, true} {
			name := tc.name + "/optional"
			if required {
				name = tc.name + "/required"
			}
			t.Run(name, func(t *testing.T) {
				root := RunHTTPDSL(t, func() {
					Type("AnyValue", Any)
					Type("NullableValue", String, func() {
						Nullable()
					})
					Service("sender", func() {
						Method("send", func() {
							Payload(func() {
								tc.attribute()
								if required {
									Required("body")
								}
							})
							HTTP(func() {
								POST("/")
								Body("body")
							})
						})
					})
				})
				services := CreateHTTPServices(root)
				require.Equal(t, tc.allows, services.Get("sender").Endpoints[0].Payload.Request.BodyAllowsNull)
				file := findFileWithSection(t, ServerFiles("gen", services), "request-decoder")
				source := codegen.SectionCode(t, file.Section("request-decoder")[0])
				if tc.allows {
					require.Contains(t, source, "decoder(r).Decode")
					require.NotContains(t, source, "WithNonNullableBody")
				} else {
					require.Contains(t, source, "decoder(loomhttp.WithNonNullableBody(r)).Decode")
				}
			})
		}
	}
}
