package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	servicetestdata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestRecursiveProjectionJSONRPC(t *testing.T) {
	for _, order := range []struct {
		name       string
		unionFirst bool
	}{{"array_first", false}, {"union_first", true}} {
		t.Run(order.name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, func() {
				servicetestdata.RecursiveProjectionTransportDSL("jsonrpc", order.unionFirst)
			})
			dir := t.TempDir()
			renderJSONRPCModule(t, dir, "example.com/projection", root)
			data := CreateJSONRPCServices(root)
			for _, svc := range root.Services {
				if views := servicecodegen.ViewsFile("example.com/projection/gen", svc, data.ServicesData); views != nil {
					renderCodegenFiles(t, dir, []*cg.File{views})
				}
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "projection_test.go"), []byte(recursiveProjectionJSONRPCHarness), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "service_test.go"), []byte(servicetestdata.RecursiveProjectionServiceHarness), 0o600))
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "vet", "./gen/...")
			runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}

const recursiveProjectionJSONRPCHarness = `package projection_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	server "example.com/projection/gen/jsonrpc/projection/server"
	svc "example.com/projection/gen/projection"
	loomhttp "github.com/CaliLuke/loom/http"
)

func (s implementation) Watch(ctx context.Context, stream svc.WatchServerStream) error {
	value, err := s.Show(ctx)
	if err != nil {
		return err
	}
	return stream.Send(ctx, value)
}

func TestSelectedWire(t *testing.T) {
	var prior, priorStream string
	for _, secret := range []string{"FIRST_SECRET", "SECOND_SECRET"} {
		mux := loomhttp.NewMuxer()
		server.Mount(mux, server.New(svc.NewEndpoints(implementation{secret}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
		request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"show\"}"))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		wire := response.Body.String()
		if response.Code != http.StatusOK || strings.Contains(wire, "\"error\"") {
			t.Fatalf("status=%d wire=%s", response.Code, wire)
		}
		if strings.Contains(wire, secret) {
			t.Errorf("excluded secret on wire %s", wire)
		}
		if !strings.Contains(wire, "allowed") {
			t.Errorf("explicit full view lost field: %s", wire)
		}
		if prior != "" && prior != wire {
			t.Errorf("excluded field changed bytes: %s vs %s", prior, wire)
		}
		prior = wire
		streamRequest := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"watch\"}"))
		streamRequest.Header.Set("Content-Type", "application/json")
		streamRequest.Header.Set("Accept", "text/event-stream")
		streamResponse := httptest.NewRecorder()
		mux.ServeHTTP(streamResponse, streamRequest)
		streamWire := streamResponse.Body.String()
		if streamResponse.Code != 200 || strings.Contains(streamWire, secret) || !strings.Contains(streamWire, "allowed") {
			t.Fatalf("invalid selected SSE wire: %d %s", streamResponse.Code, streamWire)
		}
		// The envelope currently uses a map with unstable key order. Compare the
		// exact encoded selected value, without decoding or re-encoding that value.
		var envelope struct {
			Params  json.RawMessage
			JSONRPC string
			Method  string
		}
		parts := strings.SplitN(streamWire, "data: ", 2)
		if len(parts) != 2 {
			t.Fatalf("missing SSE data: %q", streamWire)
		}
		data := strings.TrimSpace(parts[1])
		if err := json.Unmarshal([]byte(data), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.JSONRPC != "2.0" || envelope.Method != "projection/stream.event" || len(envelope.Params) == 0 {
			t.Fatalf("invalid SSE envelope: %+v", envelope)
		}
		selectedWire := string(envelope.Params)
		if priorStream != "" && priorStream != selectedWire {
			t.Errorf("excluded field changed SSE params bytes: %q vs %q", priorStream, selectedWire)
		}
		priorStream = selectedWire
	}
}
`
