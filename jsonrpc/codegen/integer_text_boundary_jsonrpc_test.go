package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestIntegerTextBoundaryJSONRPC proves that JSON-RPC metadata reuses the HTTP
// integer text owner: physical storage round-trips, the client rejects an
// unrepresentable value before Do and the server rejects narrow overflow.
func TestIntegerTextBoundaryJSONRPC(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		Service("ints", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("inspect", func() {
				Payload(func() {
					Attribute("wide", UInt, func() {
						Meta("struct:field:type", "int64")
					})
					Attribute("dur", Int, func() {
						Meta("struct:field:type", "time.Duration", "time")
					})
					Attribute("small", Int, func() {
						Meta("struct:field:type", "int8")
					})
					Required("wide", "dur", "small")
				})
				JSONRPC(func() {
					Header("wide:X-Wide")
					Header("dur:X-Dur")
					Header("small:X-Small")
				})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/rpcints", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ints_test.go"), []byte(integerTextBoundaryJSONRPCHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

const integerTextBoundaryJSONRPCHarness = `package rpcints_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	svc "example.com/rpcints/gen/ints"
	client "example.com/rpcints/gen/jsonrpc/ints/client"
	server "example.com/rpcints/gen/jsonrpc/ints/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type countingDoer struct {
	calls int
	base  *http.Client
}

func (d *countingDoer) Do(req *http.Request) (*http.Response, error) {
	d.calls++
	return d.base.Do(req)
}

func TestIntegerMetadata(t *testing.T) {
	var got *svc.InspectPayload
	endpoint := func(_ context.Context, raw any) (any, error) {
		got = raw.(*svc.InspectPayload)
		return nil, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{Inspect: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	doer := &countingDoer{base: host.Client()}
	c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	payload := &svc.InspectPayload{Wide: 1 << 40, Dur: 5 * time.Second, Small: -128}
	_, err := c.Inspect()(t.Context(), payload)
	require.NoError(t, err)
	require.Equal(t, payload, got)
	_, err = c.Inspect()(t.Context(), &svc.InspectPayload{Wide: -1, Dur: 1, Small: 1})
	require.ErrorContains(t, err, "cannot be represented as uint")
	require.Equal(t, 1, doer.calls)

	got = nil
	request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\"inspect\",\"params\":{},\"id\":1}"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Wide", "1")
	request.Header.Set("X-Dur", "5000000000")
	request.Header.Set("X-Small", "200")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Nil(t, got)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Equal(t, float64(-32602), envelope["error"].(map[string]any)["code"])
}
`
