package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestMediaGeneratedIntegration(t *testing.T) {
	const media = "application/vnd.example+json; charset=utf-8"
	for _, test := range []struct {
		name    string
		module  string
		design  func(string)
		harness string
	}{
		{"aliases", "example.com/aliasbody", aliasBodyModuleDSL, aliasRequestBodyHarness},
		{"nullable", "example.com/nullbody", nullableRequestBodyIntegrationDSL, nullableRequestBodyHarness},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				test.design(media)
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, test.module, root)
			harness := strings.ReplaceAll(test.harness, `"application/json"`, `"`+media+`"`)
			harness = strings.ReplaceAll(harness, "loomhttp.RequestEncoder", "requestMediaEncoder")
			harness = strings.ReplaceAll(harness, "loomhttp.RequestDecoder", "requestMediaDecoder")
			harness += `
func requestMediaDecoder(request *http.Request) loomhttp.Decoder {
	media := request.Header.Get("Content-Type")
	if media != "" && media != "application/vnd.example+json; charset=utf-8" {
		panic("custom decoder did not receive original request media")
	}
	return loomhttp.RequestDecoder(request)
}

func requestMediaEncoder(request *http.Request) loomhttp.Encoder {
	if request.Header.Get("Content-Type") != "application/vnd.example+json; charset=utf-8" {
		panic("authored media was not published before custom encoder factory")
	}
	return loomhttp.RequestEncoder(request)
}
`
			require.NoError(t, os.WriteFile(filepath.Join(dir, "request_media_test.go"), []byte(harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "-count=1", ".")
		})
	}
}
