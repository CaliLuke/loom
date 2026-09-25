package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestOptionalBodyCLI asserts that the client CLI makes the body flag of an
// optional union, object or nullable payload attribute selected with Body
// optional, decodes it only when it is set, and leaves the attribute nil or
// absent when it is not. The body flag example of a union holds the union
// discriminator and value. A required attribute keeps a required flag.
func TestOptionalBodyCLI(t *testing.T) {
	cases := []struct {
		Name      string
		Attribute func(leaf, other any)
		// Build lists fragments of the payload builder of an optional body.
		Build []string
		// Example is a fragment of the body flag example.
		Example string
	}{
		{
			Name:      "union",
			Attribute: func(leaf, other any) { Attribute("b", OneOf(leaf, other)) },
			Build: []string{
				"\tvar body PickRequestBody\n\t{\n\t\tif pickerPickBody != \"\" {\n\t\t\terr = json.Unmarshal([]byte(pickerPickBody), &body)\n",
				"\tres := &picker.PickPayload{B: v}\n",
			},
			Example: `\"type\": \"`,
		},
		{
			Name:      "object",
			Attribute: func(leaf, _ any) { Attribute("b", leaf) },
			Build: []string{
				"\t\tif pickerPickBody != \"\" {\n\t\t\terr = json.Unmarshal([]byte(pickerPickBody), &body)\n",
				"\tres := &picker.PickPayload{}\n\tif pickerPickBody != \"\" {\n",
				"\t\tres.B = v\n\t}\n",
			},
			Example: `\"name\": \"`,
		},
		{
			Name:      "nullable object",
			Attribute: func(leaf, _ any) { Attribute("b", leaf, func() { Nullable() }) },
			Build: []string{
				"\t\tif pickerPickBody != \"\" {\n\t\t\terr = json.Unmarshal([]byte(pickerPickBody), &body)\n",
				"\tres := &picker.PickPayload{B: v}\n",
			},
			Example: `\"name\": \"`,
		},
	}
	for _, c := range cases {
		for _, required := range []bool{false, true} {
			name := c.Name + "/optional"
			if required {
				name = c.Name + "/required"
			}
			t.Run(name, func(t *testing.T) {
				root := RunHTTPDSL(t, optionalBodyCLIDSL(c.Attribute, required))
				services := CreateHTTPServices(root)
				files := ClientCLIFiles("gen", services)
				build := codegen.SectionCode(t, findFileWithSection(t, files, "cli-build-payload").Section("cli-build-payload")[0])
				parse := codegen.SectionCode(t, findFileWithSection(t, files, "parse-endpoint").Section("parse-endpoint")[0])
				assert.Contains(t, build, c.Example)
				if required {
					assert.NotContains(t, build, "if pickerPickBody != \"\"")
					assert.Contains(t, parse, "\t\t\tBody string `help:\"\" name:\"body\" required:\"\"`\n")
					return
				}
				for _, want := range c.Build {
					assert.Contains(t, build, want)
				}
				assert.Contains(t, parse, "\t\t\tBody string `help:\"\" name:\"body\"`\n")
			})
		}
	}
}

// optionalBodyCLIDSL returns a design whose method selects the payload
// attribute "b" declared by attribute as the request body with Body. The
// attribute is required when required is true.
func optionalBodyCLIDSL(attribute func(leaf, other any), required bool) func() {
	return func() {
		leaf := Type("Leaf", func() {
			Attribute("name", String)
			Required("name")
		})
		other := Type("Other", func() {
			Attribute("count", Int)
		})
		Service("Picker", func() {
			Method("Pick", func() {
				Payload(func() {
					Attribute("q", String)
					attribute(leaf, other)
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
