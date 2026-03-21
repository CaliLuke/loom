package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/dsl"
	"goa.design/goa/v3/jsonrpc/codegen/testdata"
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
		assert.Contains(t, code, "goa.ErrorSafeMessage(err)")
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
				assert.Contains(t, code, "goa.ErrorSafeMessage(err)")
				assert.Contains(t, code, "jsonrpc.NewErrorData(err)")
			}
		}
		require.True(t, found, "expected jsonrpc-sse-server-stream section")
	})
}
