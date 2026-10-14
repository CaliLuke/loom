package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

func TestJSONRPCScalarBindingPresence(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		Service("presence", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("inspect", func() {
				Payload(func() {
					Attribute("required", String)
					Attribute("optional", String)
					Attribute("defaulted", String, func() {
						Default("fallback")
					})
					Attribute("number", Int)
					Required("required")
				})
				JSONRPC(func() {
					Header("required")
					Cookie("optional")
					Cookie("defaulted")
					Header("number")
				})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/rpcpresence", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_test.go"), []byte(jsonrpcScalarPresenceHarness), 0600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const jsonrpcScalarPresenceHarness = `package presence_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	server "example.com/rpcpresence/gen/jsonrpc/presence/server"
	svc "example.com/rpcpresence/gen/presence"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestScalarMappings(t *testing.T) {
	for _, name := range []string{"absent", "empty", "value", "missing-required", "empty-number"} {
		t.Run(name, func(t *testing.T) {
			var got *svc.InspectPayload
			endpoint := func(_ context.Context, p any) (any, error) {
				got = p.(*svc.InspectPayload)
				return nil, nil
			}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(&svc.Endpoints{Inspect: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
			request := httptest.NewRequest("POST", "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\"inspect\",\"params\":{},\"id\":1}"))
			request.Header.Set("Content-Type", "application/json")
			if name != "missing-required" {
				request.Header.Set("required", "")
			}
			if name == "empty" {
				request.AddCookie(&http.Cookie{Name: "optional", Value: ""})
				request.AddCookie(&http.Cookie{Name: "defaulted", Value: ""})
			}
			if name == "value" {
				request.AddCookie(&http.Cookie{Name: "optional", Value: "value"})
			}
			if name == "empty-number" {
				request.Header.Set("number", "")
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
			if name == "missing-required" || name == "empty-number" {
				require.Nil(t, got)
				require.Contains(t, envelope, "error")
				rpcError := envelope["error"].(map[string]any)
				require.Equal(t, float64(-32602), rpcError["code"])
				return
			}
			require.NotContains(t, envelope, "error", response.Body.String())
			require.NotNil(t, got)
			require.Empty(t, got.Required)
			switch name {
			case "absent":
				require.Nil(t, got.Optional)
				require.Equal(t, "fallback", got.Defaulted)
			case "empty":
				require.NotNil(t, got.Optional)
				require.Empty(t, *got.Optional)
				require.Empty(t, got.Defaulted)
			case "value":
				require.NotNil(t, got.Optional)
				require.Equal(t, "value", *got.Optional)
			}
		})
	}
}
`
