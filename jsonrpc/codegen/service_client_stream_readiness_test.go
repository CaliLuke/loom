package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestServiceClientWebSocketStreamReadiness(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketStreamingPayloadDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwspayload", root)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	const assertions = `package jsonrpcwspayload_test

import (
	files "example.com/jsonrpcwspayload/gen/files"
	client "example.com/jsonrpcwspayload/gen/jsonrpc/files/client"
)

var _ files.UploadClientStream = (*client.UploadClientStream)(nil)
var _ files.CountClientStream = (*client.CountClientStream)(nil)
var _ files.NamesClientStream = (*client.NamesClientStream)(nil)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "service_contract_test.go"), []byte(assertions), 0o600))
	runGoJSONRPCTestCommand(t, dir, "test", "-run", "^$", "./...")
}

// This is the LocalBoth shape from the existing mixed-result specimen.
func TestServiceClientMixedSSEReadiness(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("mixed", func() {
			dsl.JSONRPC(func() {
			})
		})
		note := dsl.Type("Note", func() {
			dsl.Attribute("text", dsl.String)
		})
		item := dsl.Type("Item", func() {
			dsl.Attribute("id", dsl.Int)
		})
		dsl.Service("mixed", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("LocalBoth", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
				})
				dsl.Result(note)
				dsl.StreamingResult(item)
				dsl.JSONRPC(func() {
					dsl.ServerSentEvents()
				})
			})
		})
	})
	dir := t.TempDir()
	const modulePath = "example.com/mixedcontract"
	renderJSONRPCModule(t, dir, modulePath, root)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	const assertions = `package contract_test
import (
 svc "example.com/mixedcontract/gen/mixed"
 client "example.com/mixedcontract/gen/jsonrpc/mixed/client"
)
var _ svc.LocalBothClientStream = (*client.LocalBothClientStream)(nil)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "contract_test.go"), []byte(assertions), 0o600))
	runGoJSONRPCTestCommand(t, dir, "test", "-run", "^$", "./...")
}
