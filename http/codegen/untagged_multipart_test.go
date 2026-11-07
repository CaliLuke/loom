package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestMultipartCapturedGeneration(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedMultipartDSL)
	data := CreateHTTPServices(root).Get("multipartchoices")
	for _, endpoint := range data.Endpoints {
		if !endpoint.Payload.Request.Multipart {
			continue
		}
		t.Run(endpoint.Method.Name, func(t *testing.T) {
			require.True(t, endpoint.Payload.Request.MultipartGenerated)
			require.Nil(t, endpoint.MultipartRequestDecoder)
			require.Nil(t, endpoint.MultipartRequestEncoder)
		})
	}
}

func TestUntaggedMultipartGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedMultipartDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/multipartchoices", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "multipart_test.go"), []byte(testdata.UntaggedMultipartHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
