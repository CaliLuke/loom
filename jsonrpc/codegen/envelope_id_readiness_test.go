package codegen

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

//go:embed testdata/envelope_id_readiness_harness.go.txt
var envelopeIDReadinessHarness string

//go:embed testdata/envelope_id_stream_harness.go.txt
var envelopeIDStreamHarness string

func TestJSONRPCEnvelopeIDMounted(t *testing.T) {
	dir := t.TempDir()
	if output := os.Getenv("LOOM_ENVELOPE_ID_OUTPUT"); output != "" {
		dir = output
		require.NoError(t, os.MkdirAll(dir, 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(fmt.Sprintf("module example.com/r4id\n\ngo 1.27.3\n\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", resolveJSONRPCTestLoomModulePath(t, dir))), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "design"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design", "design.go"), []byte(`package design
import "github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
func init() {
 testdata.JSONRPCEnvelopeID()
}
`), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/r4id/design")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "h_test.go"), []byte(envelopeIDReadinessHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stream_test.go"), []byte(envelopeIDStreamHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	out, err := runJSONRPCTestCommandAllowFailure(dir, "go", "test", "-count=1", "-v", "./...")
	t.Log(out)
	require.NoError(t, err)
}

func TestJSONRPCEnvelopeIDBodyOwnership(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCEnvelopeID)
	for _, endpoint := range root.API.JSONRPC.Services[0].HTTPEndpoints {
		if endpoint.Name() != "fmt" && endpoint.Name() != "pat" && endpoint.Name() != "plain" {
			continue
		}
		assert.Nil(t, endpoint.Body.Find("id"), "ID must have exactly one wire location: envelope, not params")
		require.NotNil(t, endpoint.Body.Find("v"), "ordinary params member must remain")
	}
}

// TestJSONRPCEnvelopeIDAdmission rejects competing envelope ownership at the design boundary.
func TestJSONRPCEnvelopeIDAdmission(t *testing.T) {
	for _, tc := range []struct {
		name    string
		field   func()
		mapping func()
	}{
		{"ordinary", func() {
			dsl.Attribute("id", dsl.String)
		}, nil},
		{"plain", func() {
			dsl.ID("id", dsl.String)
		}, nil},
		{"integer", func() {
			dsl.ID("id", dsl.Int)
		}, nil},
		{"nullable", func() {
			dsl.ID("id", dsl.String, func() {
				dsl.Nullable()
			})
		}, nil},
		{"default", func() {
			dsl.ID("id", dsl.String, func() {
				dsl.Default("fallback")
			})
		}, nil},
		{"renamed", func() {
			dsl.ID("id", dsl.String, func() {
				dsl.Meta("struct:field:name", "Correlation")
			})
		}, nil},
		{"duplicate", func() {
			dsl.ID("first", dsl.String)
			dsl.ID("second", dsl.String)
		}, nil},
		{"header", func() {
			dsl.ID("id", dsl.String)
		}, func() {
			dsl.Header("id:X-Request-ID")
		}},
		{"other-body", func() {
			dsl.ID("id", dsl.String)
			dsl.Attribute("value", dsl.String)
		}, func() {
			dsl.Body(func() {
				dsl.Attribute("value")
			})
		}},
		{"explicit-id-copy", func() {
			dsl.ID("id", dsl.String)
		}, func() {
			dsl.Body(func() {
				dsl.Attribute("id")
			})
		}},
		{"nested", func() {
			dsl.Attribute("nested", func() {
				dsl.ID("id", dsl.String)
			})
		}, nil},
		{"query", func() {
			dsl.ID("id", dsl.String)
		}, func() {
			dsl.Param("id")
		}},
		{"cookie", func() {
			dsl.ID("id", dsl.String)
		}, func() {
			dsl.Cookie("id")
		}},
		{"selected-id-body", func() {
			dsl.ID("id", dsl.String)
		}, func() {
			dsl.Body("id")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.SetupTestDSL(t)
			require.True(t, eval.Execute(func() {
				dsl.API("admission", func() {
					dsl.JSONRPC(func() {})
				})
				dsl.Service("probe", func() {
					dsl.JSONRPC(func() {
						dsl.POST("/rpc")
					})
					dsl.Method("check", func() {
						dsl.Payload(tc.field)
						dsl.JSONRPC(func() {
							if tc.mapping != nil {
								tc.mapping()
							}
						})
					})
				})
			}, nil))
			err := eval.RunDSL()
			switch tc.name {
			case "integer", "duplicate", "header", "query", "cookie", "nested", "selected-id-body", "explicit-id-copy":
				require.ErrorContains(t, err, "JSON-RPC")
				return
			default:
				require.NoError(t, err)
			}
			endpoint := root.API.JSONRPC.Services[0].HTTPEndpoints[0]
			if tc.name == "ordinary" {
				require.Nil(t, endpoint.JSONRPCID)
				require.NotNil(t, expr.AsObject(endpoint.Body.Type).Attribute("id"))
				return
			}
			require.NotNil(t, endpoint.JSONRPCID)
			assert.Equal(t, "id", endpoint.JSONRPCID.Name)
			if object := expr.AsObject(endpoint.Body.Type); object != nil {
				assert.Nil(t, object.Attribute("id"))
			}
		})
	}
}

func TestJSONRPCEnvelopeIDTransportBodyOwnership(t *testing.T) {
	for _, transport := range []string{"unary", "sse", "ws-server", "ws-bidirectional"} {
		t.Run(transport, func(t *testing.T) {
			root := RunJSONRPCDSL(t, func() {
				dsl.API("envelope", func() {
					dsl.JSONRPC(func() {})
				})
				dsl.Service("transport", func() {
					dsl.JSONRPC(func() {
						if transport == "ws-server" || transport == "ws-bidirectional" {
							dsl.GET("/rpc")
						} else {
							dsl.POST("/rpc")
						}
					})
					dsl.Method("call", func() {
						payload := func() {
							dsl.ID("id", dsl.String)
							dsl.Attribute("value", dsl.String)
							dsl.Required("id", "value")
						}
						if transport == "ws-bidirectional" {
							dsl.StreamingPayload(payload)
						} else {
							dsl.Payload(payload)
						}
						if transport == "unary" {
							dsl.Result(dsl.String)
						} else {
							dsl.StreamingResult(dsl.String)
						}
						dsl.JSONRPC(func() {
							if transport == "sse" {
								dsl.ServerSentEvents()
							}
						})
					})
				})
			})
			endpoint := root.API.JSONRPC.Services[0].HTTPEndpoints[0]
			assert.Equal(t, "id", endpoint.PayloadIDAttribute)
			assert.Nil(t, endpoint.Body.Find("id"), "message params must exclude envelope ID")
			assert.False(t, endpoint.Body.IsRequired("id"), "params cannot require an excluded member")
			require.NotNil(t, endpoint.Body.Find("value"))
			assert.NotNil(t, endpoint.MethodExpr.Payload.Find("id"), "service payload must retain the ID field")
		})
	}
}

func TestJSONRPCEnvelopeIDMappedSource(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		root := expr.SetupTestDSL(t)
		require.True(t, eval.Execute(func() {
			dsl.API("mapped", func() {
				dsl.JSONRPC(func() {})
			})
			dsl.Service("mapped", func() {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
				dsl.Method("call", func() {
					dsl.Payload(func() {
						dsl.ID("id:wire", dsl.String)
						dsl.Attribute("value", dsl.String)
					})
					dsl.JSONRPC(func() {
						if conflict {
							dsl.Header("id:X-ID")
						}
					})
				})
			})
		}, nil))
		err := eval.RunDSL()
		if conflict {
			require.ErrorContains(t, err, "belongs to the envelope")
			continue
		}
		require.NoError(t, err)
		endpoint := root.API.JSONRPC.Services[0].HTTPEndpoints[0]
		assert.Equal(t, "id:wire", endpoint.JSONRPCID.Name)
		assert.Nil(t, expr.AsObject(endpoint.Body.Type).Attribute("id:wire"))
	}
}
