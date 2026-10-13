package codegen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMappedErrorBodyMaterializesItsSource(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcErrorDisclosureDSL(true))
	services := CreateJSONRPCServices(root)
	for _, name := range []string{"unary", "events", "socket"} {
		t.Run(name, func(t *testing.T) {
			endpoint := services.Get(name).Endpoint("show")
			require.Len(t, endpoint.Errors, 1)
			response := endpoint.Errors[0].Errors[0].Response
			require.True(t, response.EncodePlan.NeedsProblemSource)
			body := response.EncodePlan.FirstBody
			require.Equal(t, "problem", body.Init.ServerArgs[0].Ref)
			require.Equal(t, "*loomhttp.ProblemResponse", body.Init.ServerArgs[0].TypeRef)

			files := append(ServerFiles("", services), SSEServerFiles("", services)...)
			matched := 0
			for _, file := range files {
				if !strings.Contains(file.Path, "/"+name+"/") {
					continue
				}
				for _, section := range file.AllSections() {
					var out bytes.Buffer
					require.NoError(t, section.Write(&out))
					code := out.String()
					calls := strings.Count(code, body.Init.Name+"(problem)")
					if calls == 0 {
						continue
					}
					matched += calls
					normalizations := strings.Count(code, `problem := loomhttp.NewProblemResponse(ctx, err, 4403, "", "")`)
					require.Equal(t, calls, normalizations, file.Path)
				}
			}
			require.Positive(t, matched)
		})
	}
}

func TestJSONRPCSSEEndpointUsesMappedErrorBody(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonRPCTypedStreamingErrorBodyDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/disclosure", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mapped_test.go"), []byte(jsonrpcMappedEndpointHarness), 0600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const jsonrpcMappedEndpointHarness = `package disclosure_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	server "example.com/disclosure/gen/jsonrpc/tools/server"
	tools "example.com/disclosure/gen/tools"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestMappedEndpoint(t *testing.T) {
	calls := 0
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&tools.Endpoints{Watch: func(context.Context, any) (any, error) {
		calls++
		return nil, &tools.ToolError{Name: "forbidden", Message: "public detail"}
	}}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, host.URL+"/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\"watch\",\"id\":\"request\"}"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	res, err := host.Client().Do(req)
	require.NoError(t, err)
	wire, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, 1, calls)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Contains(t, res.Header.Get("Content-Type"), "text/event-stream")
	require.Contains(t, string(wire), "event: message")
	for _, line := range strings.Split(string(wire), "\n") {
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &envelope))
		require.Equal(t, "request", envelope["id"])
		failure,ok:=envelope["error"].(map[string]any)
		require.True(t,ok)
		require.Equal(t, float64(4403), failure["code"])
		require.Equal(t, "public detail", failure["message"])
		require.Equal(t, map[string]any{"name": "forbidden", "message": "public detail"}, failure["data"])
		return
	}
	t.Fatal("missing SSE data frame")
}
`
