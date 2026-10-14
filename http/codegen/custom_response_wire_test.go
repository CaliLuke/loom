package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

func TestCustomResponseRepresentationWire(t *testing.T) {
	root := RunHTTPDSL(t, customResponseWireDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_CUSTOM_RESPONSE_OUTPUT"); output != "" {
		dir = output
		require.NoError(t, os.MkdirAll(dir, 0750))
	}
	renderHTTPModule(t, dir, "example.com/customresponse", root)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scalar/token.go"), []byte(customTextTokenSource), 0600))
	source, err := os.ReadFile("testdata/custom_response_wire.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), source, 0600))
	if os.Getenv("LOOM_CUSTOM_RESPONSE_GENERATE_ONLY") != "" {
		return
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func customResponseWireDSL() {
	label := Type("Label", String)
	fields := func(constrained bool) {
		for _, name := range []string{"optional", "required", "defaulted"} {
			Attribute(name, String, func() {
				Meta("struct:field:type", "scalar.Token", "example.com/customresponse/scalar")
				if constrained {
					MinLength(7)
				}
				if constrained && name == "defaulted" {
					Default("token:def")
				}
			})
		}
		Attribute("native", String)
		if constrained {
			Attribute("named", label)
		} else {
			Attribute("named", String)
		}
		Required("required")
	}
	value := Type("Value", func() {
		fields(true)
	})
	failure := Type("Failure", func() {
		fields(true)
	})
	projected := ResultType("application/vnd.record", func() {
		fields(false)
		View("default", func() {
			for _, name := range []string{"optional", "required", "defaulted", "native", "named"} {
				Attribute(name)
			}
		})
	})
	Service("customresponse", func() {
		for _, location := range []string{"header", "cookie", "projected"} {
			Method(location, func() {
				if location == "projected" {
					Result(projected)
				} else {
					Result(value)
				}
				Error("invalid", failure)
				HTTP(func() {
					GET("/" + location)
					mapping := func() {
						for _, name := range []string{"optional", "required", "defaulted", "native", "named"} {
							if location == "cookie" {
								Cookie(name)
							} else {
								Header(name)
							}
						}
					}
					Response(StatusOK, mapping)
					Response("invalid", StatusBadRequest, mapping)
				})
			})
		}
	})
}
