package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestNullableRequestBody asserts that a nullable object, array, map or
// primitive payload attribute, or an Any attribute, selected with Body is
// declared by the server with the type that the payload constructor takes,
// passed to it by value, and assigned to the payload field without taking
// its address. An optional attribute is sent only when it is present, and
// the server accepts an absent body; a required attribute is always sent
// and its body is required.
func TestNullableRequestBody(t *testing.T) {
	cases := []struct {
		Name string
		// Attribute declares the payload attribute "b" given the user type
		// Leaf.
		Attribute func(leaf any)
		// Body is the declaration of the server body variable.
		Body string
		// Nullable is true when the payload field is a loom.Nullable.
		Nullable bool
		// Validate lists the server body validation fragments.
		Validate []string
	}{
		{
			Name:      "object",
			Attribute: func(leaf any) { Attribute("b", leaf, func() { Nullable() }) },
			Body:      "\t\t\tbody loom.Nullable[PickRequestBody]\n",
			Nullable:  true,
			Validate: []string{
				"\t\terr = ValidatePickRequestBody(body)\n",
			},
		},
		{
			Name: "inline object",
			Attribute: func(any) {
				Attribute("b", func() {
					Attribute("x", String)
					Nullable()
				})
			},
			Body:     "\t\t\tbody loom.Nullable[PickRequestBody]\n",
			Nullable: true,
		},
		{
			Name:      "array",
			Attribute: func(leaf any) { Attribute("b", ArrayOf(leaf), func() { Nullable() }) },
			Body:      "\t\t\tbody loom.Nullable[[]loom.Nullable[*LeafRequestBodyRequestBody]]\n",
			Nullable:  true,
			Validate: []string{
				"loom.InvalidNullElementError(\"body\", i)",
				"ValidateLeafRequestBodyRequestBody(actual)",
			},
		},
		{
			Name:      "string array",
			Attribute: func(any) { Attribute("b", ArrayOf(String), func() { Nullable() }) },
			Body:      "\t\t\tbody loom.Nullable[[]loom.Nullable[string]]\n",
			Nullable:  true,
			Validate: []string{
				"loom.InvalidNullElementError(\"body\", i)",
			},
		},
		{
			Name:      "map",
			Attribute: func(leaf any) { Attribute("b", MapOf(String, leaf), func() { Nullable() }) },
			Body:      "\t\t\tbody loom.Nullable[map[string]loom.Nullable[*LeafRequestBodyRequestBody]]\n",
			Nullable:  true,
			Validate: []string{
				"ValidateLeafRequestBodyRequestBody(actual)",
			},
		},
		{
			Name:      "string",
			Attribute: func(any) { Attribute("b", String, func() { Nullable(); MinLength(2) }) },
			Body:      "\t\t\tbody loom.Nullable[string]\n",
			Nullable:  true,
			Validate: []string{
				"loom.InvalidLengthError(\"body\", actual, utf8.RuneCountInString(actual), 2, true)",
			},
		},
		{
			Name:      "any",
			Attribute: func(any) { Attribute("b", Any) },
			Body:      "\t\t\tbody loom.JSONValue\n",
		},
	}
	for _, c := range cases {
		for _, required := range []bool{false, true} {
			name := c.Name + "/optional"
			if required {
				name = c.Name + "/required"
			}
			t.Run(name, func(t *testing.T) {
				root := RunHTTPDSL(t, nullableRequestBodyDSL(c.Attribute, required))
				services := CreateHTTPServices(root)
				data := services.Get("Picker").Endpoint("Pick")
				require.NotNil(t, data)
				assert.Equal(t, !required, data.Payload.Request.OptionalBodyAttribute)
				assert.Equal(t, !required && c.Nullable, data.Payload.Request.OptionalBodyNullable)
				assert.False(t, data.Payload.Request.OptionalObjectBody)

				encoder := codegen.SectionCode(t, findFileWithSection(t, ClientFiles("gen", services), "request-encoder").Section("request-encoder")[0])
				decoder := codegen.SectionCode(t, findFileWithSection(t, ServerFiles("gen", services), "request-decoder").Section("request-decoder")[0])
				init := codegen.SectionCode(t, findFileWithSection(t, ServerTypeFiles("gen", services), "server-payload-init").Section("server-payload-init")[0])
				serverTypes := codegen.SectionsCode(t, findFileWithSection(t, ServerTypeFiles("gen", services), "server-payload-init").AllSections()[1:])

				assert.Contains(t, decoder, c.Body)
				assert.Contains(t, decoder, "payload = NewPickPayload(body, q)")
				assert.NotContains(t, decoder, "NewPickPayload(&body")
				assert.Contains(t, init, "\t\tB: v,\n")
				assert.NotContains(t, init, "B: &v")
				for _, want := range c.Validate {
					assert.Contains(t, decoder+serverTypes, want)
				}

				guard := "\t\tif p.B != nil {\n"
				if c.Nullable {
					guard = "\t\tif p.B.Present() {\n"
				}
				if required {
					assert.NotContains(t, encoder, guard)
					assert.Contains(t, decoder, "return payload, loom.MissingPayloadError()")
					return
				}
				assert.Contains(t, encoder, guard)
				assert.NotContains(t, decoder, "MissingPayloadError")
				assert.NotContains(t, decoder+serverTypes, "loom.MissingFieldError(\"body\", \"body\")")
			})
		}
	}
}

// nullableRequestBodyDSL returns a design whose method selects the payload
// attribute "b" declared by attribute as the request body with Body. Its
// user type Leaf has a required field so that its request body type has
// validations. The attribute is required when required is true.
func nullableRequestBodyDSL(attribute func(leaf any), required bool) func() {
	return func() {
		leaf := Type("Leaf", func() {
			Attribute("name", String)
			Required("name")
		})
		Service("Picker", func() {
			Method("Pick", func() {
				Payload(func() {
					Attribute("q", String)
					attribute(leaf)
					if required {
						Required("b")
					}
				})
				HTTP(func() {
					POST("/pick")
					Param("q")
					Body("b")
				})
			})
		})
	}
}
