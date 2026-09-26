package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestAliasRequestBodyValidation asserts that the server validates a request
// body selected with Body from an alias of a primitive, an array or a map
// with validations by value: the generated Validate function reads the
// value without a nil check or a dereference, and the decoder passes it the
// body instead of its address. An optional alias of a primitive is decoded
// into a pointer and validated inline when it is not nil. A union with an
// alias of a primitive branch validates the branch inline.
func TestAliasRequestBodyValidation(t *testing.T) {
	cases := []struct {
		Name      string
		Attribute func(code, amount, tags, dict, leaf any)
		// Validate lists fragments of the Validate function of the body,
		// if any.
		Validate []string
		// Optional lists fragments of the server request decoder when the
		// attribute is optional.
		Optional []string
		// Required lists fragments of the server request decoder when the
		// attribute is required.
		Required []string
	}{
		{
			Name: "string",
			Attribute: func(code, _, _, _, _ any) {
				Attribute("v", code)
			},
			Validate: []string{
				"(body SendRequestBody) (err error) {\n\tif utf8.RuneCountInString(string(body)) < 1 {\n",
			},
			Optional: []string{
				"\t\tif body != nil {\n\t\t\tif utf8.RuneCountInString(string(*body)) < 1 {\n",
			},
			Required: []string{"\t\terr = ValidateSendRequestBody(body)\n"},
		},
		{
			Name: "int",
			Attribute: func(_, amount, _, _, _ any) {
				Attribute("v", amount)
			},
			Validate: []string{
				"(body SendRequestBody) (err error) {\n\tif int(body) < 1 {\n",
			},
			Optional: []string{"\t\tif body != nil {\n\t\t\tif int(*body) < 1 {\n"},
			Required: []string{"\t\terr = ValidateSendRequestBody(body)\n"},
		},
		{
			Name: "array",
			Attribute: func(_, _, tags, _, _ any) {
				Attribute("v", tags)
			},
			Validate: []string{
				"(body SendRequestBody) (err error) {\n\tif len(body) < 1 {\n",
			},
			Optional: []string{"\t\tif body != nil {\n\t\t\terr = ValidateSendRequestBody(body)\n\t\t}\n"},
			Required: []string{"\t\terr = ValidateSendRequestBody(body)\n"},
		},
		{
			Name: "map",
			Attribute: func(_, _, _, dict, _ any) {
				Attribute("v", dict)
			},
			Validate: []string{
				"(body SendRequestBody) (err error) {\n\tif len(body) > 3 {\n",
			},
			Optional: []string{"\t\tif body != nil {\n\t\t\terr = ValidateSendRequestBody(body)\n\t\t}\n"},
			Required: []string{"\t\terr = ValidateSendRequestBody(body)\n"},
		},
		{
			Name: "union",
			Attribute: func(code, _, _, _, leaf any) {
				Attribute("v", OneOf(leaf, code))
			},
			Optional: []string{
				"\t\tcase \"Code\":\n\t\t\tactual, _ := body.AsCode()\n\t\t\tif utf8.RuneCountInString(string(actual)) < 1 {\n",
			},
			Required: []string{
				"\t\tcase \"Code\":\n\t\t\tactual, _ := body.AsCode()\n\t\t\tif utf8.RuneCountInString(string(actual)) < 1 {\n",
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
				root := RunHTTPDSL(t, aliasRequestBodyDSL(c.Attribute, required))
				services := CreateHTTPServices(root)
				serverFiles := ServerFiles("gen", services)
				decoder := codegen.SectionCode(t, findFileWithSection(t, serverFiles, "request-decoder").Section("request-decoder")[0])
				want := c.Optional
				if required {
					want = c.Required
				}
				for _, fragment := range want {
					require.Contains(t, decoder, fragment)
				}
				require.NotContains(t, decoder, "RequestBody(&body)")
				var validate string
				for _, f := range ServerTypeFiles("gen", services) {
					for _, s := range f.Section("server-validate") {
						code := codegen.SectionCode(t, s)
						if strings.Contains(code, "func ValidateSendRequestBody(") {
							validate = code
						}
					}
				}
				if len(c.Validate) == 0 {
					require.Empty(t, validate)
					return
				}
				require.NotEmpty(t, validate)
				for _, fragment := range c.Validate {
					require.Contains(t, validate, fragment)
				}
				require.NotContains(t, validate, "body != nil")
				require.NotContains(t, validate, "*body")
			})
		}
	}
}

// aliasRequestBodyDSL returns a design whose method selects the payload
// attribute "v" declared by attribute as the request body with Body.
// attribute receives an alias of String with a minimum length, an alias of
// Int with a minimum, an alias of an array with a minimum length, an alias of
// a map with a maximum length and an object type. The attribute is required
// when required is true.
func aliasRequestBodyDSL(attribute func(code, amount, tags, dict, leaf any), required bool) func() {
	return func() {
		code := Type("Code", String, func() {
			MinLength(1)
		})
		amount := Type("Amount", Int, func() {
			Minimum(1)
		})
		tags := Type("Tags", ArrayOf(String), func() {
			MinLength(1)
		})
		dict := Type("Dict", MapOf(String, Int), func() {
			MaxLength(3)
		})
		leaf := Type("Leaf", func() {
			Attribute("name", String)
			Required("name")
		})
		Service("Picker", func() {
			Method("Send", func() {
				Payload(func() {
					Attribute("q", String)
					attribute(code, amount, tags, dict, leaf)
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
