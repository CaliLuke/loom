package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestJSONRPCErrorProjectionUsesCoreHelpers(t *testing.T) {
	t.Run("http handlers use safe message and structured error data", func(t *testing.T) {
		root := RunJSONRPCDSL(t, func() {
			dsl.Service("calc", func() {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})

				dsl.Method("divide", func() {
					dsl.Payload(func() {
						dsl.ID("id")
						dsl.Attribute("left", dsl.Int)
						dsl.Attribute("right", dsl.Int)
						dsl.Required("left", "right")
					})
					dsl.Result(func() {
						dsl.ID("id")
						dsl.Attribute("quotient", dsl.Int)
					})
					dsl.Error("bad_request")
					dsl.JSONRPC(func() {})
				})
			})
		})

		code := jsonrpcGeneratedCode(t, ServerFiles("", CreateJSONRPCServices(root)))
		assert.Contains(t, code, "loom.ErrorSafeMessage(err)")
		assert.Contains(t, code, "jsonrpc.NewErrorData(err)")
	})

	t.Run("sse stream helpers use safe message and structured error data", func(t *testing.T) {
		root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
		files := SSEServerFiles("", CreateJSONRPCServices(root))
		require.NotEmpty(t, files)

		found := false
		for _, file := range files {
			for _, section := range file.AllSections() {
				if section.SectionName() != "jsonrpc-sse-server-stream" {
					continue
				}
				found = true
				code := codegen.SectionCode(t, section)
				assert.Contains(t, code, "loom.ErrorSafeMessage(err)")
				assert.Contains(t, code, "jsonrpc.NewErrorData(err)")
			}
		}
		require.True(t, found, "expected jsonrpc-sse-server-stream section")
	})
}

func TestJSONRPCUnmappedServiceErrorsUseInternalError(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.Service("calc", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})

			dsl.Method("divide", func() {
				dsl.Payload(func() {
					dsl.ID("id")
					dsl.Attribute("left", dsl.Int)
					dsl.Attribute("right", dsl.Int)
					dsl.Required("left", "right")
				})
				dsl.Result(func() {
					dsl.ID("id")
					dsl.Attribute("quotient", dsl.Int)
				})
				dsl.Error("internal", func() {
					dsl.Fault()
				})
				dsl.JSONRPC(func() {})
			})
		})
	})

	serverCode := jsonrpcGeneratedCode(t, ServerFiles("", CreateJSONRPCServices(root)))
	streamCode := jsonrpcGeneratedCode(t, SSEServerFiles("", CreateJSONRPCServices(root)))
	code := serverCode + "\n" + streamCode

	assert.Contains(t, code, "errors.As(err, &serviceError)")
	assert.Contains(t, code, "code = jsonrpcErrorCodeForServiceError(serviceError)")
	assert.NotContains(t, code, `err.(*loom.ServiceError)`)
	assert.NotContains(t, code, `code = jsonrpc.InvalidParams`)
}

func TestJSONRPCServiceSSESendErrorUsesSafeMappedErrors(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	code := jsonrpcGeneratedCode(t, ServerFiles("", CreateJSONRPCServices(root)))

	assert.Contains(t, code, `loom.ErrorSafeMessage(err)`)
	assert.Contains(t, code, `case "invalid_params":`)
	assert.NotContains(t, code, `message := err.Error()`)
	assert.NotContains(t, code, `var data any`)
	testutil.AssertString(t, filepath.Join("testdata", "golden", "jsonrpc-sse-object-server-files.golden"), code)
}

func TestJSONRPCServiceSSESendErrorUsesDesignedResponseCodes(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.Service("feed", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/stream")
			})

			dsl.Method("watch", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
				})
				dsl.StreamingResult(func() {
					dsl.ID("id", dsl.String)
					dsl.Attribute("value", dsl.String)
				})
				dsl.Error("rate_limited")
				dsl.JSONRPC(func() {
					dsl.ServerSentEvents()
					dsl.Response("rate_limited", func() {
						dsl.Code(4901)
					})
				})
			})
		})
	})
	code := jsonrpcGeneratedCode(t, ServerFiles("", CreateJSONRPCServices(root)))

	assert.Contains(t, code, `case "rate_limited":`)
	assert.Contains(t, code, `s.sendError(ctx, id, 4901`)
}
