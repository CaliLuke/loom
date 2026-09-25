package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestOptionalObjectRequestBody asserts that an optional object payload
// attribute selected with Body is sent only when it is not nil, that the
// server decodes the body into a pointer that stays nil when the body is
// empty and validates only a present body, and that the payload constructor
// maps a nil body to a nil attribute. A required object attribute keeps the
// unconditional client body and the by-value server body.
func TestOptionalObjectRequestBody(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Optional bool
		// Encode is the client statement that writes the body.
		Encode string
		// Decode lists server decoder fragments for an optional body.
		Decode []string
		// Validate is the server body validation call, if any.
		Validate string
	}{
		{
			Name:     "json",
			DSL:      optionalObjectRequestBodyDSL(false, false, false, false),
			Optional: true,
			Encode:   "err := encoder(req).Encode(&body)",
			Decode: []string{
				"\t\tvar (\n\t\t\tbody = &FindRequestBody{}\n\t\t\terr  error\n\t\t)\n",
				"\t\terr = decoder(r).Decode(body)\n",
				"\t\t\tif errors.Is(err, io.EOF) {\n\t\t\t\tbody = nil\n\t\t\t\terr = nil\n\t\t\t} else {\n",
			},
		},
		{
			Name:     "json-required-fields",
			DSL:      optionalObjectRequestBodyDSL(true, false, false, false),
			Optional: true,
			Encode:   "err := encoder(req).Encode(&body)",
			Decode: []string{
				"\t\terr = decoder(r).Decode(body)\n",
				"\t\t\t\tbody = nil\n",
			},
			Validate: "\t\tif body != nil {\n\t\t\terr = ValidateFindRequestBody(body)\n\t\t\tif err != nil {\n\t\t\t\treturn payload, err\n\t\t\t}\n\t\t}\n",
		},
		{
			Name:     "optional-request-body",
			DSL:      optionalObjectRequestBodyDSL(false, false, true, false),
			Optional: true,
			Encode:   "err := encoder(req).Encode(&body)",
			Decode: []string{
				"\t\terr = decoder(r).Decode(body)\n",
				"\t\t\t\tbody = nil\n",
			},
		},
		{
			Name:     "form",
			DSL:      optionalObjectRequestBodyDSL(false, true, false, false),
			Optional: true,
			Encode:   "err := loomhttp.SetFormRequest(req, &body)",
			Decode: []string{
				"\t\tif len(r.PostForm) == 0 {\n\t\t\tbody = nil\n\t\t} else {\n",
				`loomhttp.DecodeFormValue(r.PostForm, "", body)`,
			},
		},
		{
			Name:   "required",
			DSL:    optionalObjectRequestBodyDSL(true, false, false, true),
			Encode: "err := encoder(req).Encode(&body)",
			Decode: []string{
				"\t\tvar (\n\t\t\tbody FindRequestBody\n\t\t\terr  error\n\t\t)\n",
				"\t\terr = decoder(r).Decode(&body)\n",
			},
			Validate: "\t\terr = ValidateFindRequestBody(&body)\n\t\tif err != nil {\n\t\t\treturn payload, err\n\t\t}\n",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			data := services.Get("Finder").Endpoint("Find")
			require.NotNil(t, data)
			assert.Equal(t, c.Optional, data.Payload.Request.OptionalBodyAttribute)
			assert.Equal(t, c.Optional, data.Payload.Request.OptionalObjectBody)

			encoder := codegen.SectionCode(t, findFileWithSection(t, ClientFiles("gen", services), "request-encoder").Section("request-encoder")[0])
			assert.Contains(t, encoder, c.Encode)
			guarded := "\t\tif p.O != nil {\n" +
				"\t\t\tbody := NewFindRequestBody(p)\n" +
				"\t\t\tif " + c.Encode + "; err != nil {\n"

			serverFiles := ServerFiles("gen", services)
			decoder := codegen.SectionCode(t, findFileWithSection(t, serverFiles, "request-decoder").Section("request-decoder")[0])
			for _, want := range c.Decode {
				assert.Contains(t, decoder, want)
			}
			if c.Validate != "" {
				assert.Contains(t, decoder, c.Validate)
			}

			initFile := findFileWithSection(t, ServerTypeFiles("gen", services), "server-payload-init")
			init := codegen.SectionCode(t, initFile.Section("server-payload-init")[0])
			if c.Optional {
				assert.Contains(t, encoder, guarded)
				assert.Contains(t, decoder, "payload = NewFindPayload(body, q)")
				assert.Contains(t, init, "\tres := &finder.FindPayload{}\n\tif body != nil {\n")
				assert.Contains(t, init, "\t\tres.O = v\n\t}\n\tres.Q = q\n")
				return
			}
			assert.NotContains(t, encoder, "if p.O != nil")
			assert.Contains(t, decoder, "payload = NewFindPayload(&body, q)")
			assert.NotContains(t, decoder, "body = nil")
			assert.NotContains(t, init, "if body != nil")
			assert.Contains(t, init, "\tres := &finder.FindPayload{\n\t\tO: v,\n\t}\n")
		})
	}
}

// optionalObjectRequestBodyDSL returns a design whose method selects the
// object payload attribute "o" as the request body with Body. The object has
// a required field when requiredFields is true, the request is form encoded
// when form is true, the endpoint sets OptionalRequestBody when optionalBody
// is true, and the attribute is required when required is true.
func optionalObjectRequestBodyDSL(requiredFields, form, optionalBody, required bool) func() {
	return func() {
		var Filters = Type("Filters", func() {
			Attribute("name", String)
			Attribute("limit", Int)
			if requiredFields {
				Required("name")
			}
		})
		Service("Finder", func() {
			Method("Find", func() {
				Payload(func() {
					Attribute("q", String)
					Attribute("o", Filters)
					if required {
						Required("o")
					}
				})
				HTTP(func() {
					POST("/find")
					Param("q")
					Body("o")
					if form {
						FormRequest()
					}
					if optionalBody {
						OptionalRequestBody()
					}
				})
			})
		})
	}
}
