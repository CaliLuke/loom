package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestTransformHelperServer(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Section  string
		Helper   string
		Expected string
	}{
		{
			Name:     "body-user-inner-default-1",
			DSL:      testdata.PayloadBodyUserInnerDefaultDSL,
			Section:  "server-transform-helper",
			Helper:   "unmarshalInnerTypeRequestBodyToServicebodyuserinnerdefaultInnerType",
			Expected: testdata.PayloadBodyUserInnerDefaultTransformCode1,
		},
		{
			Name:     "body-user-recursive-default-1",
			DSL:      testdata.PayloadBodyInlineRecursiveUserDSL,
			Section:  "server-transform-helper",
			Helper:   "unmarshalPayloadTypeRequestBodyToServicebodyinlinerecursiveuserPayloadType",
			Expected: testdata.PayloadBodyInlineRecursiveUserTransformCode1,
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			f := ServerEncodeDecodeFile("", root.API.HTTP.Services[0], services)
			code := namedTransformHelper(t, []*codegen.File{f}, c.Section, c.Helper)
			require.Equal(t, c.Expected, code)
		})
	}
}

func TestTransformHelperCLI(t *testing.T) {
	cases := []struct {
		Name     string
		DSL      func()
		Section  string
		Helper   string
		Expected string
	}{
		{
			Name:     "cli-body-user-inner-default-1",
			DSL:      testdata.PayloadBodyUserInnerDefaultDSL,
			Section:  "cli-input-transform",
			Helper:   "unmarshalCLIInnerTypeRequestBodyToServicebodyuserinnerdefaultInnerType",
			Expected: testdata.PayloadBodyUserInnerDefaultTransformCodeCLI1,
		},
		{
			Name:     "cli-body-user-inner-default-2",
			DSL:      testdata.PayloadBodyUserInnerDefaultDSL,
			Section:  "client-transform-helper",
			Helper:   "marshalServicebodyuserinnerdefaultInnerTypeToInnerTypeRequestBody",
			Expected: testdata.PayloadBodyUserInnerDefaultTransformCodeCLI2,
		},
		{
			Name:     "cli-body-user-recursive-default-1",
			DSL:      testdata.PayloadBodyInlineRecursiveUserDSL,
			Section:  "cli-input-transform",
			Helper:   "unmarshalCLIPayloadTypeRequestBodyToServicebodyinlinerecursiveuserPayloadType",
			Expected: testdata.PayloadBodyInlineRecursiveUserTransformCodeCLI1,
		},
		{
			Name:     "cli-body-user-recursive-default-2",
			DSL:      testdata.PayloadBodyInlineRecursiveUserDSL,
			Section:  "client-transform-helper",
			Helper:   "marshalServicebodyinlinerecursiveuserPayloadTypeToPayloadTypeRequestBody",
			Expected: testdata.PayloadBodyInlineRecursiveUserTransformCodeCLI2,
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			f := ClientEncodeDecodeFile("", root.API.HTTP.Services[0], services)
			files := append(ClientCLIFiles("", services), f)
			code := namedTransformHelper(t, files, c.Section, c.Helper)
			require.Equal(t, c.Expected, code)
		})
	}
}

func namedTransformHelper(t *testing.T, files []*codegen.File, role, name string) string {
	t.Helper()
	var matches []string
	for _, file := range files {
		for _, section := range file.Section(role) {
			code := codegen.SectionCode(t, section)
			if strings.Contains(code, "func "+name+"(") {
				matches = append(matches, code)
			}
		}
	}
	require.Len(t, matches, 1, "expected exactly one %s helper %s", role, name)
	return matches[0]
}
