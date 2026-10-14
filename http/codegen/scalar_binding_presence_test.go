package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

func TestScalarBindingPresenceGenerated(t *testing.T) {
	root := RunHTTPDSL(t, scalarBindingPresenceDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_SCALAR_OUTPUT"); output != "" {
		dir = output
		require.NoError(t, os.MkdirAll(dir, 0750))
	}
	renderHTTPModule(t, dir, "example.com/scalarpresence", root)
	source, err := os.ReadFile("testdata/scalar_binding_presence.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_test.go"), source, 0600))
	if os.Getenv("LOOM_SCALAR_GENERATE_ONLY") != "" {
		return
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func scalarBindingPresenceDSL() {
	label := Type("Label", String)
	value := Type("Values", func() {
		Attribute("required", String)
		Attribute("required_default", String, func() {
			Default("required-fallback")
		})
		Attribute("optional", label)
		Attribute("optional_present", String)
		Attribute("defaulted", String, func() {
			Default("fallback")
		})
		Attribute("bytes", Bytes)
		Attribute("zero", Int, func() {
			Default(0)
		})
		Attribute("false", Boolean, func() {
			Default(false)
		})
		Attribute("number", Int)
		Attribute("flag", Boolean, func() {
			Default(true)
		})
		Attribute("limited", String, func() {
			MinLength(1)
		})
		Required("required", "required_default")
	})
	custom := Type("CustomValues", func() {
		Attribute("id", String, func() {
			Format(FormatUUID)
			Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
		})
		Attribute("fallback", String, func() {
			Format(FormatUUID)
			Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
			Default("550e8400-e29b-41d4-a716-446655440000")
		})
	})

	Service("presence", func() {
		for _, location := range []string{"header", "cookie", "query"} {
			Method(location+"_custom", func() {
				Payload(custom)
				HTTP(func() {
					GET("/" + location + "_custom")
					for _, field := range []string{"id", "fallback"} {
						switch location {
						case "header":
							Header(field)
						case "cookie":
							Cookie(field)
						case "query":
							Param(field)
						}
					}
				})
			})

			Method(location, func() {
				Payload(value)
				Result(value)
				HTTP(func() {
					GET("/" + location)
					for _, field := range []string{"required", "required_default", "optional", "optional_present", "defaulted", "bytes", "zero", "false", "number", "flag", "limited"} {
						switch location {
						case "header":
							Header(field)
						case "cookie":
							Cookie(field)
						case "query":
							Param(field)
						}
					}
					if location != "query" {
						Response(StatusOK, func() {
							for _, field := range []string{"required", "required_default", "optional", "optional_present", "defaulted", "bytes", "zero", "false", "number", "flag", "limited"} {
								if location == "header" {
									Header(field)
								} else {
									Cookie(field)
								}
							}
						})
					}
				})
			})
		}
	})
}

// TestScalarBindingDecodeOwner checks the shared presence/decoder plan before
// compiling mounted examples, including required fields that declare defaults.
func TestScalarBindingDecodeOwner(t *testing.T) {
	root := RunHTTPDSL(t, scalarBindingPresenceDSL)
	services := CreateHTTPServices(root)
	functions := renderTransportFuncs(t, append(ServerFiles("", services), ClientFiles("", services)...))
	for _, name := range []string{"DecodeHeaderRequest", "DecodeCookieRequest", "DecodeQueryRequest", "DecodeHeaderResponse", "DecodeCookieResponse"} {
		code := functions[name]
		require.NotEmpty(t, code, name)
		require.Contains(t, code, "if optionalPresent {", name)
		require.Contains(t, code, "if requiredDefaultPresent {", name)
		require.NotContains(t, code, "requiredDefaultRaw ==", name)
		require.NotContains(t, code, "optionalRaw !=", name)
		require.Contains(t, code, "if false_Present {", name)
	}
}

// TestScalarBindingPreservesMapParamsValidation prevents scalar decoding from
// changing the independent whole-query map validation path.
func TestScalarBindingPreservesMapParamsValidation(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("maps", func() {
			Method("inspect", func() {
				Payload(func() {
					Attribute("values", MapOf(String, String), func() {
						MinLength(2)
					})
				})
				HTTP(func() {
					GET("/inspect")
					MapParams("values")
				})
			})
		})
	})
	functions := renderTransportFuncs(t, ServerFiles("", CreateHTTPServices(root)))
	code := functions["DecodeInspectRequest"]
	require.Contains(t, code, "valuesRaw := qp")
	require.Contains(t, code, "len(values) < 2")
	require.Contains(t, code, `loom.InvalidLengthError("values", len(values), 2, true)`)
}
