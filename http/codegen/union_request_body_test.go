package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestUnionRequestBodyDeclaration asserts that the server declares a request
// body whose type is a union by value and decodes into its address, for
// required and optional JSON, form-encoded and WebSocket streaming payloads.
func TestUnionRequestBodyDeclaration(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Section  string
		Contains []string
	}{
		{
			Name:    "json-body",
			DSL:     unionRequestBodyDSL(func() { POST("/pick") }, false),
			Section: "request-decoder",
			Contains: []string{
				"body PickRequestBody\n",
				"err = decoder(r).Decode(&body)",
				"payload = NewPickLeafOrOther(&body)",
			},
		},
		{
			Name:    "get-body",
			DSL:     unionRequestBodyDSL(func() { GET("/pick") }, false),
			Section: "request-decoder",
			Contains: []string{
				"body PickRequestBody\n",
				"err = decoder(r).Decode(&body)",
				"payload = NewPickLeafOrOther(&body)",
			},
		},
		{
			Name:    "optional-body",
			DSL:     optionalUnionRequestBodyDSL,
			Section: "request-decoder",
			Contains: []string{
				"body PickRequestBody\n",
				"err = decoder(r).Decode(&body)",
				"payload = NewPickPayload(&body, q)",
			},
		},
		{
			Name: "form-body",
			DSL: unionRequestBodyDSL(func() {
				POST("/pick")
				FormRequest()
			}, false),
			Section: "request-decoder",
			Contains: []string{
				"body PickRequestBody\n",
				`loomhttp.DecodeFormValue(r.PostForm, "", &body)`,
				"payload = NewPickLeafOrOther(&body)",
			},
		},
		{
			Name:    "websocket-streaming-payload",
			DSL:     unionRequestBodyDSL(func() { GET("/pick") }, true),
			Section: "server-websocket-recv",
			Contains: []string{
				"msg *PickStreamingBody\n",
				"s.conn.ReadJSON(ctx, &msg)",
				"return NewPickStreamingBody(msg), nil",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			file := findFileWithSection(t, ServerFiles("gen", CreateHTTPServices(root)), c.Section)
			sections := file.Section(c.Section)
			require.NotEmpty(t, sections)
			code := codegen.SectionCode(t, sections[0])
			for _, want := range c.Contains {
				assert.Contains(t, code, want)
			}
			assert.NotContains(t, code, "body *Pick")
			assert.NotContains(t, code, "msg **")
		})
	}
}

// unionRequestBodyDSL returns a design whose only method takes a constructor
// OneOf union as its payload, or as its streaming payload when streaming is
// true, mapped by the given HTTP endpoint DSL.
func unionRequestBodyDSL(endpoint func(), streaming bool) func() {
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
				if streaming {
					StreamingPayload(OneOf(Leaf, Other))
				} else {
					Payload(OneOf(Leaf, Other))
				}
				HTTP(endpoint)
			})
		})
	}
}

// optionalUnionRequestBodyDSL returns a design whose method maps an optional
// constructor OneOf union payload attribute to the request body.
func optionalUnionRequestBodyDSL() {
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
				Attribute("u", OneOf(Leaf, Other))
			})
			HTTP(func() {
				POST("/pick")
				Param("q")
				Body("u")
			})
		})
	})
}
