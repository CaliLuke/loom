package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicetestdata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestRecursiveProjectionHTTP(t *testing.T) {
	for _, order := range []struct {
		name       string
		unionFirst bool
	}{{"array_first", false}, {"union_first", true}} {
		t.Run(order.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				servicetestdata.RecursiveProjectionTransportDSL("http", order.unionFirst)
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/projection", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "projection_test.go"), []byte(recursiveProjectionHTTPHarness), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "service_test.go"), []byte(servicetestdata.RecursiveProjectionServiceHarness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./gen/...")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}

const recursiveProjectionHTTPHarness = `package projection_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/projection/gen/http/projection/client"
	server "example.com/projection/gen/http/projection/server"
	svc "example.com/projection/gen/projection"
	loomhttp "github.com/CaliLuke/loom/http"
)

func (s implementation) Watch(ctx context.Context, stream svc.WatchServerStream) error {
	value, err := s.Show(ctx)
	if err != nil {
		return err
	}
	return stream.SendWithContext(ctx, value)
}

func TestSelectedWireAndReverseConversion(t *testing.T) {
	var prior, priorStream string
	for _, secret := range []string{"FIRST_SECRET", "SECOND_SECRET"} {
		mux := loomhttp.NewMuxer()
		server.Mount(mux, server.New(svc.NewEndpoints(implementation{secret}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
		request := httptest.NewRequest(http.MethodGet, "/show", nil)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		wire := response.Body.String()
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", response.Code, wire)
		}
		if strings.Contains(wire, secret) {
			t.Errorf("excluded secret in wire %s", wire)
		}
		if !strings.Contains(wire, "allowed") {
			t.Errorf("explicit full view lost field: %s", wire)
		}
		if prior != "" && prior != wire {
			t.Errorf("excluded field changed bytes: %s vs %s", prior, wire)
		}
		prior = wire
		streamRequest := httptest.NewRequest(http.MethodGet, "/watch", nil)
		streamResponse := httptest.NewRecorder()
		mux.ServeHTTP(streamResponse, streamRequest)
		streamWire := streamResponse.Body.String()
		if streamResponse.Code != 200 || strings.Contains(streamWire, secret) || !strings.Contains(streamWire, "allowed") {
			t.Fatalf("invalid selected SSE wire: %d %s", streamResponse.Code, streamWire)
		}
		if priorStream != "" && priorStream != streamWire {
			t.Errorf("excluded field changed SSE bytes")
		}
		priorStream = streamWire
		res, err := client.DecodeShowResponse(loomhttp.ResponseDecoder, false)(&http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(wire))})
		if err != nil {
			t.Fatal(err)
		}
		result := res.(*svc.Wrap)
		for _, owner := range []*svc.Owner{result.Owner, result.Owners[0], result.Items[0], result.ByKey["k"], result.Grouped["k"][0], result.Holder.Owner} {
			if owner.ID != "owner" || owner.Secret != "" {
				t.Errorf("wrong selected owner: %+v", owner)
			}
		}
		branch, ok := result.Pick.Choice.AsOwner()
		if !ok || branch.Secret != "" {
			t.Errorf("union reverse projection leaked field: %+v", branch)
		}
		if result.FullOwner.Secret != "allowed" {
			t.Errorf("full reverse projection lost secret")
		}
	}
}
`
