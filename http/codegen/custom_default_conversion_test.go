package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestCustomDefaultConversionHTTP(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("customdefault", func() {
			for _, name := range []string{"good", "bad"} {
				value := "token:default"
				if name == "bad" {
					value = "invalid"
				}
				fields := func() {
					Attribute("token", String, func() {
						Meta("struct:field:type", "scalar.Token", "example.com/customdefault/scalar")
						Default(value)
					})
				}
				Method(name+"stream", func() {
					StreamingPayload(fields)
					Result(String)
					HTTP(func() {
						GET("/" + name + "/stream")
					})
				})
				Method(name, func() {
					Payload(fields)
					Result(fields)
					HTTP(func() {
						POST("/" + name)
						Response(StatusOK)
					})
				})
			}
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/customdefault", root)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scalar/token.go"), []byte(customTextTokenSource), 0o600))
	source, err := os.ReadFile("testdata/custom_default_conversion.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "conversion_test.go"), source, 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}
