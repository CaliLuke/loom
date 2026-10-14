package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/codegentest"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRequestContentLengthGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.RequestContentLength)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/lengths", root)
	harness, err := os.ReadFile("testdata/request_content_length_test.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lengths_test.go"), harness, 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
	runGoCommand(t, dir, "vet", "./...")
}

func TestRequestContentLengthCompletionPlan(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
		want   bool
	}{
		{"mapped length", testdata.RequestContentLength, true},
		{"raw without length", testdata.RawRequestBodyOpenAPIDSL, true},
		{"ordinary header", testdata.PayloadHeaderPrimitiveStringValidateDSL, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			data := CreateHTTPServices(root)
			for _, ep := range data.Get(root.Services[0].Name).Endpoints {
				require.Equal(t, tc.want, ep.Payload.Request.FinalizeEncoding)
			}
			sections := codegentest.Sections(ClientFiles("gen", data), "encode_decode.go", "request-encoder")
			require.NotEmpty(t, sections)
			for _, section := range sections {
				rendered := codegen.SectionCode(t, section)
				count := 0
				if tc.want {
					count = 1
				}
				require.Equal(t, count, strings.Count(rendered, "loomhttp.FinalizeRequestEncoding("))
			}
		})
	}
}

func TestRequestContentLengthMethods(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("verbs", func() {
			for _, name := range []string{"get", "head", "delete", "options", "post", "put", "patch"} {
				Method(name, func() {
					Payload(func() {
						Attribute("length", Int64)
					})
					HTTP(func() {
						switch name {
						case "get":
							GET("/get")
						case "head":
							HEAD("/head")
						case "delete":
							DELETE("/delete")
						case "options":
							OPTIONS("/options")
						case "post":
							POST("/post")
						case "put":
							PUT("/put")
						case "patch":
							PATCH("/patch")
						}
						Header("length:Content-Length")
						SkipRequestBodyEncodeDecode()
					})
				})
			}
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/verbs", root)
	harness, err := os.ReadFile("testdata/request_content_length_methods.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "verbs_test.go"), harness, 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", "-v", "./...")
	runGoCommand(t, dir, "vet", "./...")
}
