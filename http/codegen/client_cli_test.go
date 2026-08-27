package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestClientCLIFiles(t *testing.T) {
	cases := []struct {
		Name         string
		DSL          func()
		FileIndex    int
		SectionIndex int
	}{
		{"no-payload-parse", testdata.MultiNoPayloadDSL, 0, 3},
		{"simple-parse", testdata.MultiSimpleDSL, 0, 3},
		{"multi-parse", testdata.MultiDSL, 0, 3},
		{"multi-required-payload", testdata.MultiRequiredPayloadDSL, 0, 3},
		{"skip-request-body-encode-decode", testdata.SkipRequestBodyEncodeDecodeDSL, 0, 3},
		{"streaming-parse", testdata.StreamingMultipleServicesDSL, 0, 3},
		{"simple-build", testdata.MultiSimpleDSL, 1, 1},
		{"multi-build", testdata.MultiDSL, 1, 1},
		{"bool-build", testdata.PayloadQueryBoolDSL, 1, 1},
		{"uint32-build", testdata.PayloadQueryUInt32DSL, 1, 1},
		{"uint64-build", testdata.PayloadQueryUIntDSL, 1, 1},
		{"string-build", testdata.PayloadQueryStringDSL, 1, 1},
		{"string-required-build", testdata.PayloadQueryStringValidateDSL, 1, 1},
		{"string-default-build", testdata.PayloadQueryStringDefaultDSL, 1, 1},
		{"body-query-path-object-build", testdata.PayloadBodyQueryPathObjectDSL, 1, 1},
		{"param-validation-build", testdata.ParamValidateDSL, 1, 1},
		{"payload-primitive-type", testdata.PayloadBodyPrimitiveBoolValidateDSL, 0, 3},
		{"payload-array-primitive-type", testdata.PayloadBodyPrimitiveArrayStringValidateDSL, 0, 3},
		{"payload-array-user-type", testdata.PayloadBodyInlineArrayUserDSL, 1, 1},
		{"payload-map-user-type", testdata.PayloadBodyInlineMapUserDSL, 1, 1},
		{"payload-object-type", testdata.PayloadBodyInlineObjectDSL, 1, 1},
		{"payload-object-default-type", testdata.PayloadBodyInlineObjectDefaultDSL, 1, 1},
		{"map-query", testdata.PayloadMapQueryPrimitiveArrayDSL, 0, 3},
		{"map-query-object", testdata.PayloadMapQueryObjectDSL, 1, 1},
		{"empty-body-build", testdata.PayloadBodyPrimitiveFieldEmptyDSL, 1, 1},
		{"with-params-and-headers-dsl", testdata.WithParamsAndHeadersBlockDSL, 1, 1},
		{"body-custom-name", testdata.PayloadBodyCustomNameDSL, 1, 1},
		{"path-custom-name", testdata.PayloadPathCustomNameDSL, 1, 1},
		{"query-custom-name", testdata.PayloadQueryCustomNameDSL, 1, 1},
		{"header-custom-name", testdata.PayloadHeaderCustomNameDSL, 1, 1},
		{"cookie-custom-name", testdata.PayloadCookieCustomNameDSL, 1, 1},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ClientCLIFiles("", services)
			sections := fs[c.FileIndex].AllSections()
			code := codegen.SectionCode(t, sections[c.SectionIndex])
			testutil.AssertGo(t, "testdata/golden/client_cli_"+c.Name+".go.golden", code)
		})
	}
}

func TestClientCLIExamplesAreStableAcrossUnrelatedServices(t *testing.T) {
	baseline := renderServiceClientCLI(t, clientCLIExampleStabilityDSL(false, false), "stable")
	changed := renderServiceClientCLI(t, clientCLIExampleStabilityDSL(true, false), "stable")
	reordered := renderServiceClientCLI(t, clientCLIExampleStabilityDSL(false, true), "stable")
	repeated := renderServiceClientCLI(t, clientCLIExampleStabilityDSL(false, false), "stable")
	baselineUnrelated := renderServiceClientCLI(t, clientCLIExampleStabilityDSL(false, false), "alpha")
	changedUnrelated := renderServiceClientCLI(t, clientCLIExampleStabilityDSL(true, false), "alpha")

	require.NotEqual(t, baselineUnrelated, changedUnrelated)
	require.Equal(t, baseline, changed)
	require.Equal(t, baseline, reordered)
	require.Equal(t, baseline, repeated)
}

func renderServiceClientCLI(t *testing.T, dsl func(), serviceName string) string {
	t.Helper()
	root := RunHTTPDSL(t, dsl)
	services := CreateHTTPServices(root)
	for _, file := range ClientCLIFiles("gen", services) {
		for _, section := range file.Section("cli-command-usage") {
			var output bytes.Buffer
			require.NoError(t, section.Write(&output))
			if strings.Contains(output.String(), codegen.Goify(serviceName, false)+"Usage") {
				return output.String()
			}
		}
	}
	t.Fatalf("client CLI command usage for service %q not found", serviceName)
	return ""
}

func clientCLIExampleStabilityDSL(changeUnrelated, reverse bool) func() {
	return func() {
		API("example-stability", func() {
			Server("api", func() {
				Services("stable", "alpha")
				Host("development", func() {
					URI("http://localhost:8080")
				})
			})
		})
		stable := func() {
			Service("stable", func() {
				Method("show", func() {
					Payload(func() {
						Attribute("stable_id", String, func() {
							Format(FormatUUID)
						})
						Required("stable_id")
					})
					Result(String)
					HTTP(func() {
						GET("/stable/{stable_id}")
						Param("stable_id")
					})
				})
			})
		}
		unrelated := func() {
			Service("alpha", func() {
				Method("list", func() {
					Payload(func() {
						Attribute("filter", String)
						if changeUnrelated {
							Attribute("cursor", String, func() {
								Format(FormatUUID)
							})
						}
					})
					Result(func() {
						Attribute("id", String, func() {
							Format(FormatUUID)
						})
						Required("id")
						if changeUnrelated {
							Attribute("detail", String)
							Required("detail")
						}
					})
					HTTP(func() {
						GET("/unrelated")
						Param("filter")
						if changeUnrelated {
							Param("cursor")
						}
					})
				})
			})
		}
		if reverse {
			stable()
			unrelated()
			return
		}
		unrelated()
		stable()
	}
}
