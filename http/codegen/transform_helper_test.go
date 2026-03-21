package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/http/codegen/testdata"
)

func TestTransformHelperServer(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Offset   int
		Expected string
	}{
		{
			Name:     "body-user-inner-default-1",
			DSL:      testdata.PayloadBodyUserInnerDefaultDSL,
			Offset:   1,
			Expected: testdata.PayloadBodyUserInnerDefaultTransformCode1,
		},
		{
			Name:     "body-user-recursive-default-1",
			DSL:      testdata.PayloadBodyInlineRecursiveUserDSL,
			Offset:   1,
			Expected: testdata.PayloadBodyInlineRecursiveUserTransformCode1,
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			f := ServerEncodeDecodeFile("", root.API.HTTP.Services[0], services)
			sections := f.SectionTemplates
			require.Greater(t, len(sections), c.Offset)
			code := codegen.SectionCode(t, sections[len(sections)-c.Offset])
			require.Equal(t, c.Expected, code)
		})
	}
}

func TestTransformHelperCLI(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Offset   int
		Expected string
	}{
		{
			Name:     "cli-body-user-inner-default-1",
			DSL:      testdata.PayloadBodyUserInnerDefaultDSL,
			Offset:   1,
			Expected: testdata.PayloadBodyUserInnerDefaultTransformCodeCLI1,
		},
		{
			Name:     "cli-body-user-inner-default-2",
			DSL:      testdata.PayloadBodyUserInnerDefaultDSL,
			Offset:   2,
			Expected: testdata.PayloadBodyUserInnerDefaultTransformCodeCLI2,
		},
		{
			Name:     "cli-body-user-recursive-default-1",
			DSL:      testdata.PayloadBodyInlineRecursiveUserDSL,
			Offset:   1,
			Expected: testdata.PayloadBodyInlineRecursiveUserTransformCodeCLI1,
		},
		{
			Name:     "cli-body-user-recursive-default-2",
			DSL:      testdata.PayloadBodyInlineRecursiveUserDSL,
			Offset:   2,
			Expected: testdata.PayloadBodyInlineRecursiveUserTransformCodeCLI2,
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			f := ClientEncodeDecodeFile("", root.API.HTTP.Services[0], services)
			sections := f.AllSections()
			require.Greater(t, len(sections), c.Offset)
			code := codegen.SectionCode(t, sections[len(sections)-c.Offset])
			require.Equal(t, c.Expected, code)
		})
	}
}
