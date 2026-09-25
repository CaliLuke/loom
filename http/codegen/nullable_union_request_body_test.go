package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestNullableUnionRequestBody asserts that a nullable union payload
// attribute selected with Body is passed by value to the server payload
// constructor, which assigns it without dereferencing it. An optional
// attribute is sent only when it is present, and the server accepts an
// absent body; a required attribute is always sent and its body is
// required.
func TestNullableUnionRequestBody(t *testing.T) {
	cases := []struct {
		Name     string
		Required bool
	}{
		{Name: "optional"},
		{Name: "required", Required: true},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, nullableUnionRequestBodyDSL(c.Required))
			services := CreateHTTPServices(root)
			data := services.Get("Picker").Endpoint("Pick")
			require.NotNil(t, data)
			assert.Equal(t, !c.Required, data.Payload.Request.OptionalBodyAttribute)
			assert.Equal(t, !c.Required, data.Payload.Request.OptionalBodyNullable)
			assert.False(t, data.Payload.Request.OptionalObjectBody)

			encoder := codegen.SectionCode(t, findFileWithSection(t, ClientFiles("gen", services), "request-encoder").Section("request-encoder")[0])
			decoder := codegen.SectionCode(t, findFileWithSection(t, ServerFiles("gen", services), "request-decoder").Section("request-decoder")[0])
			init := codegen.SectionCode(t, findFileWithSection(t, ServerTypeFiles("gen", services), "server-payload-init").Section("server-payload-init")[0])

			assert.Contains(t, decoder, "\t\t\tbody loom.Nullable[PickRequestBody]\n")
			assert.Contains(t, decoder, "err = decoder(r).Decode(&body)")
			assert.Contains(t, decoder, "payload = NewPickPayload(body, q)")
			assert.NotContains(t, decoder, "NewPickPayload(&body")
			assert.Contains(t, init, "func NewPickPayload(body loom.Nullable[PickRequestBody], q *string) *picker.PickPayload {")
			assert.Contains(t, init, "\tres := &picker.PickPayload{\n\t\tU: v,\n\t}\n")
			assert.NotContains(t, init, "U: *v")

			guard := "\t\tif p.U.Present() {\n\t\t\tbody := NewLoomNullablePickRequestBody(p)\n"
			if c.Required {
				assert.NotContains(t, encoder, "p.U.Present()")
				assert.Contains(t, decoder, "return payload, loom.MissingPayloadError()")
				assert.Contains(t, decoder, "if !body.Present() {")
				return
			}
			assert.Contains(t, encoder, guard)
			assert.NotContains(t, decoder, "MissingPayloadError")
			assert.NotContains(t, decoder, "if !body.Present() {")
		})
	}
}

// nullableUnionRequestBodyDSL returns a design whose method selects a
// nullable constructor OneOf union payload attribute as the request body with
// Body. The attribute is required when required is true.
func nullableUnionRequestBodyDSL(required bool) func() {
	return func() {
		var Leaf = Type("Leaf", func() {
			Attribute("name", String)
			Required("name")
		})
		var Other = Type("Other", func() {
			Attribute("count", Int)
		})
		Service("Picker", func() {
			Method("Pick", func() {
				Payload(func() {
					Attribute("q", String)
					Attribute("u", OneOf(Leaf, Other), func() {
						Nullable()
					})
					if required {
						Required("u")
					}
				})
				HTTP(func() {
					POST("/pick")
					Param("q")
					Body("u")
				})
			})
		})
	}
}
