package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestOptionalValueRequestBody asserts that an optional primitive, array or
// map payload attribute selected with Body is sent only when it is not nil,
// that the server leaves it nil for an empty body and validates only a
// present body, and that the client CLI body flag of such an attribute is
// optional and leaves it nil when it is empty. A primitive is decoded into a
// pointer, and the client builds the body of an alias of a primitive from
// the dereferenced field. A primitive with a default value is a value field: it keeps the
// unconditional client body and a required CLI flag, and so does a required
// attribute.
func TestOptionalValueRequestBody(t *testing.T) {
	cases := []struct {
		Name      string
		Attribute func(leaf, plain any)
		Optional  bool
		Primitive bool
		// Encode is the client statement that builds the body.
		Encode string
		// Decode lists fragments of the server request decoder.
		Decode []string
		// Init lists fragments of the server payload constructor.
		Init []string
		// Build lists fragments of the CLI payload builder.
		Build []string
		// ClientInit is a fragment of the client body builder, if any.
		ClientInit string
	}{
		{
			Name:      "string",
			Attribute: func(_, _ any) { Attribute("v", String, func() { MinLength(2) }) },
			Optional:  true,
			Primitive: true,
			Encode:    "\t\tif p.V != nil {\n\t\t\tbody := p.V\n",
			Decode: []string{
				"\t\tvar (\n\t\t\tbody = new(string)\n\t\t\terr  error\n\t\t)\n",
				"\t\terr = decoder(r).Decode(body)\n",
				"\t\t\tif errors.Is(err, io.EOF) {\n\t\t\t\tbody = nil\n\t\t\t\terr = nil\n\t\t\t} else {\n",
				"\t\tif body != nil {\n\t\t\tif utf8.RuneCountInString(*body) < 2 {\n",
				"payload = NewSendPayload(body, q)",
			},
			Init: []string{
				"func NewSendPayload(body *string, q *string) *picker.SendPayload {\n\tres := &picker.SendPayload{}\n\tif body != nil {\n\t\tv := *body\n\t\tres.V = &v\n\t}\n",
			},
			Build: []string{
				"\tvar body *string\n\t{\n\t\tif pickerSendBody != \"\" {\n\t\t\tbody = &pickerSendBody\n",
				"\tres := &picker.SendPayload{}\n\tif pickerSendBody != \"\" {\n\t\tv := *body\n\t\tres.V = &v\n\t}\n",
			},
		},
		{
			Name:      "int",
			Attribute: func(_, _ any) { Attribute("v", Int) },
			Optional:  true,
			Primitive: true,
			Encode:    "\t\tif p.V != nil {\n\t\t\tbody := p.V\n",
			Decode: []string{
				"\t\t\tbody = new(int)\n",
				"\t\terr = decoder(r).Decode(body)\n",
				"\t\t\t\tbody = nil\n",
			},
			Init: []string{
				"func NewSendPayload(body *int, q *string) *picker.SendPayload {\n\tres := &picker.SendPayload{}\n\tif body != nil {\n\t\tv := *body\n\t\tres.V = &v\n\t}\n",
			},
			Build: []string{
				"\tvar body *int\n\t{\n\t\tif pickerSendBody != \"\" {\n",
				"\tif pickerSendBody != \"\" {\n\t\tv := *body\n\t\tres.V = &v\n\t}\n",
			},
		},
		{
			Name:      "array",
			Attribute: func(_, _ any) { Attribute("v", ArrayOf(String), func() { MinLength(1) }) },
			Optional:  true,
			Encode:    "\t\tif p.V != nil {\n\t\t\tbody := p.V\n",
			Decode: []string{
				"\t\terr = decoder(r).Decode(&body)\n",
				"\t\tif body != nil {\n\t\t\tif len(body) < 1 {\n",
				"payload = NewSendPayload(body, q)",
			},
			Init: []string{
				"\tres := &picker.SendPayload{}\n\tif body != nil {\n\t\tv := make([]string, len(body))\n",
				"\t\tres.V = v\n\t}\n",
			},
			Build: []string{
				"\t\tif pickerSendBody != \"\" {\n\t\t\terr = json.Unmarshal([]byte(pickerSendBody), &body)\n",
				"\tres := &picker.SendPayload{}\n\tif pickerSendBody != \"\" {\n\t\tv := make([]string, len(body))\n",
			},
		},
		{
			Name:      "array of objects",
			Attribute: func(leaf, _ any) { Attribute("v", ArrayOf(leaf)) },
			Optional:  true,
			Encode:    "\t\tif p.V != nil {\n\t\t\tbody := NewLeafRequestBodyRequestBody(p)\n",
			Decode: []string{
				"\t\tif body != nil {\n\t\t\tfor i, e := range body {\n",
			},
			Init: []string{
				"\tres := &picker.SendPayload{}\n\tif body != nil {\n\t\tv := make([]*picker.Leaf, len(body))\n",
			},
			Build: []string{
				"\tif pickerSendBody != \"\" {\n\t\tv := make([]*picker.Leaf, len(body))\n",
			},
		},
		{
			Name:      "map",
			Attribute: func(_, _ any) { Attribute("v", MapOf(String, Int)) },
			Optional:  true,
			Encode:    "\t\tif p.V != nil {\n\t\t\tbody := p.V\n",
			Decode: []string{
				"\t\terr = decoder(r).Decode(&body)\n",
			},
			Init: []string{
				"\tres := &picker.SendPayload{}\n\tif body != nil {\n\t\tv := make(map[string]int, len(body))\n",
			},
			Build: []string{
				"\tif pickerSendBody != \"\" {\n\t\tv := make(map[string]int, len(body))\n",
			},
		},
		{
			Name:      "bytes",
			Attribute: func(_, _ any) { Attribute("v", Bytes) },
			Optional:  true,
			Encode:    "\t\tif p.V != nil {\n\t\t\tbody := p.V\n",
			Decode: []string{
				"\t\t\tbody []byte\n",
				"\t\terr = decoder(r).Decode(&body)\n",
			},
			Init: []string{
				"\tres := &picker.SendPayload{}\n\tif body != nil {\n\t\tv := body\n\t\tres.V = v\n\t}\n",
			},
			Build: []string{
				"\tif pickerSendBody != \"\" {\n\t\tv := body\n\t\tres.V = v\n\t}\n",
			},
		},
		{
			Name:      "alias",
			Attribute: func(_, plain any) { Attribute("v", plain) },
			Optional:  true,
			Primitive: true,
			Encode:    "\t\tif p.V != nil {\n\t\t\tbody := NewSendRequestBody(p)\n",
			Decode: []string{
				"\t\t\tbody = new(SendRequestBody)\n",
				"\t\terr = decoder(r).Decode(body)\n",
			},
			Init: []string{
				"\tres := &picker.SendPayload{}\n\tif body != nil {\n\t\tv := picker.Plain(*body)\n\t\tres.V = &v\n\t}\n",
			},
			Build: []string{
				"\tvar body SendRequestBody\n",
				"\tif pickerSendBody != \"\" {\n\t\tv := picker.Plain(body)\n\t\tres.V = &v\n\t}\n",
			},
			ClientInit: "\tbody := SendRequestBody(*p.V)\n",
		},
		{
			Name:      "default",
			Attribute: func(_, _ any) { Attribute("v", String, func() { Default("d") }) },
			Encode:    "\t\tbody := p.V\n",
			Decode: []string{
				"\t\t\tbody string\n",
				"\t\terr = decoder(r).Decode(&body)\n",
			},
			Init: []string{
				"\tv := body\n\tres := &picker.SendPayload{\n\t\tV: v,\n\t}\n",
			},
			Build: []string{
				"\tv := body\n\tres := &picker.SendPayload{V: v}\n",
			},
		},
	}
	for _, c := range cases {
		for _, required := range []bool{false, true} {
			name := c.Name + "/optional"
			if required {
				name = c.Name + "/required"
			}
			t.Run(name, func(t *testing.T) {
				root := RunHTTPDSL(t, optionalValueRequestBodyDSL(c.Attribute, required))
				services := CreateHTTPServices(root)
				data := services.Get("Picker").Endpoint("Send")
				require.NotNil(t, data)
				optional := c.Optional && !required
				assert.Equal(t, optional, data.Payload.Request.OptionalBodyAttribute)
				assert.Equal(t, optional && c.Primitive, data.Payload.Request.OptionalPrimitiveBody)
				assert.False(t, data.Payload.Request.OptionalObjectBody)

				encoder := codegen.SectionCode(t, findFileWithSection(t, ClientFiles("gen", services), "request-encoder").Section("request-encoder")[0])
				decoder := codegen.SectionCode(t, findFileWithSection(t, ServerFiles("gen", services), "request-decoder").Section("request-decoder")[0])
				init := codegen.SectionCode(t, findFileWithSection(t, ServerTypeFiles("gen", services), "server-payload-init").Section("server-payload-init")[0])
				cliFiles := ClientCLIFiles("gen", services)
				build := codegen.SectionCode(t, findFileWithSection(t, cliFiles, "cli-build-payload").Section("cli-build-payload")[0])
				parse := codegen.SectionCode(t, findFileWithSection(t, cliFiles, "parse-endpoint").Section("parse-endpoint")[0])
				if !optional {
					assert.NotContains(t, encoder, "if p.V != nil {")
					assert.NotContains(t, decoder, "body = nil")
					assert.NotContains(t, init, "if body != nil {")
					assert.NotContains(t, build, "if pickerSendBody != \"\"")
					assert.Contains(t, parse, "\t\t\tBody string `help:\"\" name:\"body\" required:\"\"`\n")
					if !c.Optional {
						for _, want := range c.Decode {
							assert.Contains(t, decoder, want)
						}
						for _, want := range c.Init {
							assert.Contains(t, init, want)
						}
						for _, want := range c.Build {
							assert.Contains(t, build, want)
						}
					}
					return
				}
				assert.Contains(t, encoder, c.Encode)
				for _, want := range c.Decode {
					assert.Contains(t, decoder, want)
				}
				for _, want := range c.Init {
					assert.Contains(t, init, want)
				}
				for _, want := range c.Build {
					assert.Contains(t, build, want)
				}
				assert.Contains(t, parse, "\t\t\tBody string `help:\"\" name:\"body\"`\n")
				if c.ClientInit != "" {
					clientInit := codegen.SectionCode(t, findFileWithSection(t, ClientTypeFiles("gen", services), "client-body-init").Section("client-body-init")[0])
					assert.Contains(t, clientInit, c.ClientInit)
				}
			})
		}
	}
}

// optionalValueRequestBodyDSL returns a design whose method selects the
// payload attribute "v" declared by attribute as the request body with Body.
// attribute receives an object type with a required field and an alias of
// String. The attribute is required when required is true.
func optionalValueRequestBodyDSL(attribute func(leaf, plain any), required bool) func() {
	return func() {
		leaf := Type("Leaf", func() {
			Attribute("name", String)
			Required("name")
		})
		plain := Type("Plain", String)
		Service("Picker", func() {
			Method("Send", func() {
				Payload(func() {
					Attribute("q", String)
					attribute(leaf, plain)
					if required {
						Required("v")
					}
				})
				HTTP(func() {
					POST("/send")
					Param("q")
					Body("v")
				})
			})
		})
	}
}
