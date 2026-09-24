package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestTypeFilesNamedUnionUsedTwiceValidatesBothFields checks that the
// generated server and client validation of a type with two fields of the
// same named union validates each field with its own code, as the body and as
// a nested type.
func TestTypeFilesNamedUnionUsedTwiceValidatesBothFields(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		var Code = Type("Code", String, func() {
			MinLength(2)
		})
		var Leaf = Type("Leaf", func() {
			Attribute("name", String)
		})
		var Choice = Type("Choice", OneOf(Code, Leaf))
		var Pair = Type("Pair", func() {
			Attribute("first", Choice)
			Attribute("second", Choice)
			Required("first", "second")
		})
		Service("reuse", func() {
			Method("echo", func() {
				Payload(Pair)
				Result(Pair)
				HTTP(func() {
					POST("/")
				})
			})
			Method("nest", func() {
				Payload(func() {
					Attribute("pair", Pair)
				})
				Result(func() {
					Attribute("pair", Pair)
				})
				HTTP(func() {
					POST("/nest")
				})
			})
		})
	})
	services := CreateHTTPServices(root)
	cases := []struct {
		name  string
		files []*codegen.File
	}{
		{"server", ServerTypeFiles("", services)},
		{"client", ClientTypeFiles("", services)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			for _, f := range c.files {
				for _, s := range f.AllSections() {
					require.NoError(t, s.Write(&buf))
				}
			}
			code := buf.String()
			first := strings.Count(code, `InvalidLengthError("body.first.value"`)
			second := strings.Count(code, `InvalidLengthError("body.second.value"`)
			assert.Positive(t, first)
			assert.Equal(t, first, second, "each field of the named union has its own validation code")
		})
	}
}
