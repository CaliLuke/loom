package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/dsl"
)

// TestConvertFilesExternalFieldNames checks the ConvertTo and CreateFrom
// functions of types whose attribute names differ from the Go field names of
// the external types, such as "string" and String. The attributes of the
// external types carry the Go field name as their element name suffix, so the
// conversions must look up their required attributes by attribute name.
func TestConvertFilesExternalFieldNames(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
		Code string
	}{
		{"convert-external-name", testdata.ConvertExternalNameDSL, testdata.ConvertExternalNameCode},
		{"convert-external-name-required", testdata.ConvertExternalNameRequiredDSL, testdata.ConvertExternalNameRequiredCode},
		{"convert-external-name-with-initialism", testdata.ConvertExternalNameWithInitialismDSL, testdata.ConvertExternalNameWithInitialismCode},
		{"create-external-name", testdata.CreateExternalNameDSL, testdata.CreateExternalNameCode},
		{"create-external-name-required", testdata.CreateExternalNameRequiredDSL, testdata.CreateExternalNameRequiredCode},
		{"create-external-name-with-initialism", testdata.CreateExternalNameWithInitialismDSL, testdata.CreateExternalNameWithInitialismCode},
		{"create-mixed-case", testdata.MixedCaseDSL, mixedCaseCreateCode},
		{"convert-struct-field-external", externalFieldDSL, externalFieldConvertCode},
		{"create-struct-field-external", externalFieldDSL, externalFieldCreateCode},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			var code string
			for _, service := range root.Services {
				files, err := ConvertFiles(root, service, services)
				require.NoError(t, err)
				for _, file := range files {
					code += codegen.SectionsCode(t, file.Sections[1:])
				}
			}
			assert.Contains(t, withoutBlankLines(code), withoutBlankLines(c.Code))
		})
	}
}

// externalFieldDSL maps the attribute "text" to the String field of the
// external type with struct:field:external.
func externalFieldDSL() {
	var TextType = dsl.Type("TextType", func() {
		dsl.ConvertTo(testdata.ExternalNameT{})
		dsl.CreateFrom(testdata.ExternalNameT{})
		dsl.Attribute("text", dsl.String, func() {
			dsl.Meta("struct:field:external", "String")
		})
		dsl.Required("text")
	})
	dsl.Service("Service", func() {
		dsl.Method("Method", func() {
			dsl.Payload(TextType)
		})
	})
}

// withoutBlankLines removes the empty lines of code, which the expected code
// omits before the return statements.
func withoutBlankLines(code string) string {
	return strings.ReplaceAll(code, "\n\n", "\n")
}

const mixedCaseCreateCode = `// CreateFromMixedCaseModel initializes t from the fields of v
func (t *StringType) CreateFromMixedCaseModel(v *external.MixedCaseModel) {
	temp := &StringType{
		LowerCamelID: &v.LowerCamelID,
		UpperCamelID: &v.UpperCamelID,
		SnakeID:      &v.SnakeID,
	}
	*t = *temp
}
`

const externalFieldConvertCode = `// ConvertToExternalNameT creates an instance of ExternalNameT initialized from
// t.
func (t *TextType) ConvertToExternalNameT() *testdata.ExternalNameT {
	v := &testdata.ExternalNameT{
		String: t.Text,
	}
	return v
}
`

const externalFieldCreateCode = `// CreateFromExternalNameT initializes t from the fields of v
func (t *TextType) CreateFromExternalNameT(v *testdata.ExternalNameT) {
	temp := &TextType{
		Text: v.String,
	}
	*t = *temp
}
`
