package codegen

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

//go:embed testdata/notification_completion_harness.go.txt
var notificationCompletionHarness string

func TestJSONRPCNotificationCompletionGenerated(t *testing.T) {
	dir := t.TempDir()
	if output := os.Getenv("LOOM_NOTIFICATION_COMPLETION_OUTPUT"); output != "" {
		dir = output
		require.NoError(t, os.MkdirAll(dir, 0o700))
	}
	renderJSONRPCModule(t, dir, "example.com/notificationcompletion", RunJSONRPCDSL(t, notificationCompletionDSL))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "completion_test.go"), []byte(notificationCompletionHarness), 0o600))
	if os.Getenv("LOOM_NOTIFICATION_COMPLETION_GENERATE_ONLY") != "" {
		return
	}
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "-v", "./...")
}

func TestJSONRPCNotificationCompletionOrdinary(t *testing.T) {
	dir := t.TempDir()
	if output := os.Getenv("LOOM_NOTIFICATION_COMPLETION_ORDINARY_OUTPUT"); output != "" {
		dir = output
		require.NoError(t, os.MkdirAll(dir, 0o700))
	}
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwsprimitive", RunJSONRPCDSL(t, jsonrpcWebSocketPrimitiveResultDSL))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "primitive_test.go"), []byte(jsonRPCWebSocketPrimitiveResultHarness), 0o600))
	if os.Getenv("LOOM_NOTIFICATION_COMPLETION_GENERATE_ONLY") != "" {
		return
	}
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func notificationCompletionDSL() {
	dsl.API("completion", func() {
		dsl.JSONRPC(func() {
		})
	})
	input := dsl.Type("Input", func() {
		dsl.ID("id", dsl.String)
		dsl.Attribute("value", dsl.String)
		dsl.Required("value")
	})
	dsl.Service("probe", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		for _, kind := range []string{"unary", "server", "bidi", "required"} {
			dsl.Method(kind, func() {
				if kind == "bidi" || kind == "unary" {
					dsl.StreamingPayload(input)
				} else {
					dsl.Payload(input, func() {
						if kind == "required" {
							dsl.Required("id")
						}
					})
				}
				if kind == "unary" {
					dsl.Result(dsl.String)
				} else {
					dsl.StreamingResult(dsl.String)
				}
				dsl.JSONRPC(func() {
				})
			})
		}
	})
}
