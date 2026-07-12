package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// TestJSONRPCResponseDecoderSectionGolden renders the full
// jsonrpc-response-decoder section for representative design shapes and
// compares each against a golden file. The DSL cases run serially because DSL
// evaluation mutates package-global state.
func TestJSONRPCResponseDecoderSectionGolden(t *testing.T) {
	cases := []struct {
		name     string
		dsl      func()
		contains []string
	}{
		{
			name: "primitive-result",
			dsl: func() {
				dsl.API("jsonrpc-decoder-primitive", func() {
					dsl.JSONRPC(func() {})
				})
				dsl.Service("calc", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("ping", func() {
						dsl.Payload(func() {
							dsl.ID("id")
						})
						dsl.Result(dsl.String)
						dsl.JSONRPC(func() {})
					})
				})
			},
			contains: []string{"var jresp jsonrpc.RawResponse"},
		},
		{
			name: "object-result-validated",
			dsl: func() {
				dsl.API("jsonrpc-decoder-object", func() {
					dsl.JSONRPC(func() {})
				})
				dsl.Service("calc", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("describe", func() {
						dsl.Payload(func() {
							dsl.ID("id")
						})
						dsl.Result(func() {
							dsl.ID("id")
							dsl.Attribute("name", dsl.String, func() {
								dsl.MinLength(1)
							})
							dsl.Attribute("count", dsl.Int)
							dsl.Required("name")
						})
						dsl.JSONRPC(func() {})
					})
				})
			},
			contains: []string{"loomhttp.ErrValidationError"},
		},
		{
			name: "array-result",
			dsl: func() {
				dsl.API("jsonrpc-decoder-array", func() {
					dsl.JSONRPC(func() {})
				})
				dsl.Service("calc", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("list", func() {
						dsl.Payload(func() {
							dsl.ID("id")
						})
						dsl.Result(dsl.ArrayOf(dsl.String))
						dsl.JSONRPC(func() {})
					})
				})
			},
		},
		{
			name: "map-result",
			dsl: func() {
				dsl.API("jsonrpc-decoder-map", func() {
					dsl.JSONRPC(func() {})
				})
				dsl.Service("calc", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("index", func() {
						dsl.Payload(func() {
							dsl.ID("id")
						})
						dsl.Result(dsl.MapOf(dsl.String, dsl.Int))
						dsl.JSONRPC(func() {})
					})
				})
			},
		},
		{
			name: "single-error",
			dsl: func() {
				dsl.API("jsonrpc-decoder-single-error", func() {
					dsl.JSONRPC(func() {})
				})
				dsl.Service("calc", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("divide", func() {
						dsl.Payload(func() {
							dsl.ID("id")
						})
						dsl.Result(dsl.String)
						dsl.Error("bad_request")
						dsl.JSONRPC(func() {
							dsl.Response("bad_request", func() {
								dsl.Code(4100)
							})
						})
					})
				})
			},
			contains: []string{
				"switch jresp.Error.Code {",
				"case 4100:",
			},
		},
		{
			name: "named-errors",
			dsl: func() {
				dsl.API("jsonrpc-decoder-named-errors", func() {
					dsl.JSONRPC(func() {})
				})
				dsl.Service("calc", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("transfer", func() {
						dsl.Payload(func() {
							dsl.ID("id")
						})
						dsl.Result(dsl.String)
						dsl.Error("insufficient_funds")
						dsl.Error("account_closed")
						dsl.JSONRPC(func() {
							dsl.Response("insufficient_funds", func() {
								dsl.Code(4200)
							})
							dsl.Response("account_closed", func() {
								dsl.Code(4200)
							})
						})
					})
				})
			},
			contains: []string{
				"switch jresp.Error.Code {",
				"switch jerrData.Name {",
				`case "insufficient_funds":`,
				`case "account_closed":`,
			},
		},
		{
			name: "viewed-result-fixed-view",
			dsl: func() {
				dsl.API("jsonrpc-decoder-viewed-fixed", func() {
					dsl.JSONRPC(func() {})
				})
				resultType := dsl.ResultType("application/vnd.decoder.viewed.fixed", func() {
					dsl.TypeName("FixedViewResult")
					dsl.Attributes(func() {
						dsl.Attribute("a", dsl.String)
						dsl.Attribute("b", dsl.String)
					})
					dsl.View("default", func() {
						dsl.Attribute("a")
						dsl.Attribute("b")
					})
					dsl.View("tiny", func() {
						dsl.Attribute("a")
					})
				})
				dsl.Service("viewer", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("show", func() {
						dsl.Result(resultType, func() {
							dsl.View("tiny")
						})
						dsl.JSONRPC(func() {})
					})
				})
			},
			contains: []string{`view := "tiny"`},
		},
		{
			// Reuses the multi-view fixture from encode_decode_refactor_test.go:
			// no fixed View on the method means the decoder reads loom-view.
			name:     "viewed-result-view-header",
			dsl:      jsonrpcViewedResultDSL,
			contains: []string{`view := resp.Header.Get("loom-view")`},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code := responseDecoderSectionCode(t, c.dsl)
			for _, want := range c.contains {
				require.Contains(t, code, want)
			}
			golden := filepath.Join("testdata", "golden", "jsonrpc-response-decoder-"+c.name+".golden")
			testutil.AssertGo(t, golden, code)
		})
	}
}

// TestJSONRPCErrorDecodeSwitchBranches exercises the error decode switch
// writer directly to cover the branch structure the DSL cannot reach in a
// single design: skipped groups, single-error groups, and multi-error groups
// that dispatch on the JSON-RPC error data name.
func TestJSONRPCErrorDecodeSwitchBranches(t *testing.T) {
	makeInit := func(name string) *httpcodegen.InitData {
		return &httpcodegen.InitData{
			Name: name,
			ClientArgs: []*httpcodegen.InitArgData{
				{Ref: "&body"},
			},
		}
	}
	endpoint := &httpcodegen.EndpointData{
		ServiceName: "calc",
		Method:      &service.MethodData{Name: "divide"},
		Errors: []*httpcodegen.ErrorGroupData{
			{
				StatusCode: "jsonrpc.InvalidParams",
				Errors:     []*httpcodegen.ErrorData{},
			},
			{
				StatusCode: "jsonrpc.InternalError",
				Errors: []*httpcodegen.ErrorData{
					{
						Name: "timeout",
						Response: &httpcodegen.ResponseData{
							ClientBody: &httpcodegen.TypeData{
								VarName:     "DivideTimeoutResponseBody",
								ValidateRef: "err = ValidateDivideTimeoutResponseBody(&body)",
							},
							ResultInit: makeInit("NewDivideTimeout"),
						},
					},
				},
			},
			{
				StatusCode: "jsonrpc.ServerError",
				Errors: []*httpcodegen.ErrorData{
					{
						Name: "insufficient_funds",
						Response: &httpcodegen.ResponseData{
							ClientBody: &httpcodegen.TypeData{
								VarName: "DivideInsufficientFundsResponseBody",
							},
							ResultInit: makeInit("NewDivideInsufficientFunds"),
						},
					},
					{
						Name:     "unmapped",
						Response: nil,
					},
					{
						Name: "account_closed",
						Response: &httpcodegen.ResponseData{
							ClientBody: &httpcodegen.TypeData{
								VarName: "DivideAccountClosedResponseBody",
							},
						},
					},
				},
			},
		},
	}

	code := renderGroupWriter(t, func(g *jen.Group) {
		g.Switch(jen.Id("jresp").Dot("Error").Dot("Code")).BlockFunc(func(sg *jen.Group) {
			writeJSONRPCErrorDecodeSwitch(sg, endpoint)
			sg.Default().Block(
				jen.Return(jen.Nil(), jen.Id("jresp").Dot("Error")),
			)
		})
	})

	require.Contains(t, code, `var jerrData jsonrpc.ErrorData`)
	require.Contains(t, code, `switch jerrData.Name {`)
	require.NotContains(t, code, `case "unmapped":`)
	golden := filepath.Join("testdata", "golden", "jsonrpc-error-decode-switch.golden")
	testutil.AssertGo(t, golden, code)
}

// responseDecoderSectionCode runs the given DSL and renders every
// jsonrpc-response-decoder section of the generated client encode_decode.go
// file.
func responseDecoderSectionCode(t *testing.T, dslFn func()) string {
	t.Helper()

	root := RunJSONRPCDSL(t, dslFn)
	files := ClientFiles("", CreateJSONRPCServices(root))
	file := requireEncodeDecodeFile(t, files, "client")

	var out strings.Builder
	for _, section := range file.AllSections() {
		if section.SectionName() != "jsonrpc-response-decoder" {
			continue
		}
		out.WriteString(codegen.SectionCode(t, section))
	}
	require.NotEmpty(t, out.String(), "expected at least one jsonrpc-response-decoder section")
	return out.String()
}

// renderGroupWriter renders the statements produced by the given writer inside
// a wrapper function so the output can be formatted as Go source.
func renderGroupWriter(t *testing.T, write func(g *jen.Group)) string {
	t.Helper()

	section := codegen.NewJenniferSection("decoder-sections-test", func(stmt *jen.Statement) {
		stmt.Func().Id("testcase").Params().BlockFunc(write)
	})
	return codegen.SectionCode(t, section)
}
