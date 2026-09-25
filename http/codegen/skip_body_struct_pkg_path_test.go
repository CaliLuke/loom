package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestSkipBodyStructPkgPathClientReferences checks that the client code of
// methods that skip the HTTP body encoding qualifies the request and response
// data structures with the service package, not with the struct:pkg:path
// package of the payload or result type.
func TestSkipBodyStructPkgPathClientReferences(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SkipBodyStructPkgPathDSL)
	services := CreateHTTPServices(root)
	code := make(map[string]string)
	for _, f := range ClientFiles("example.com/probe/gen", services) {
		content, err := renderFileToString(f)
		require.NoError(t, err, f.Path)
		code[filepath.Base(f.Path)] = content
	}
	cases := []struct {
		File    string
		Want    []string
		NotWant []string
	}{
		{
			File: "encode_decode.go",
			Want: []string{
				"rd, ok := v.(*files.UploadRequestData)",
				"data, ok := v.(*files.UploadRequestData)",
				`"*files.UploadRequestData", v)`,
				"func BuildUploadStreamPayload(payload any, fpath string) (*files.UploadRequestData, error)",
				"return &files.UploadRequestData{",
			},
			NotWant: []string{"common.UploadRequestData"},
		},
		{
			File:    "client.go",
			Want:    []string{"return &files.DownloadResponseData{Result: res.(*common.Item), Body: resp.Body}, nil"},
			NotWant: []string{"common.DownloadResponseData"},
		},
	}
	for _, c := range cases {
		content, ok := code[c.File]
		if !assert.True(t, ok, c.File) {
			continue
		}
		for _, want := range c.Want {
			assert.Contains(t, content, want, c.File)
		}
		for _, notWant := range c.NotWant {
			assert.NotContains(t, content, notWant, c.File)
		}
	}
}
