package testdata

// UntaggedArraysHarness exercises compiled service codecs and both HTTP directions.
const UntaggedArraysHarness = `package arrayunion_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	client "example.com/arrayunion/gen/http/listing/client"
	server "example.com/arrayunion/gen/http/listing/server"
	svc "example.com/arrayunion/gen/listing"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestServiceCodecs(t *testing.T) {
	for _, wire := range []string{"[]", "[{\"name\":\"ok\",\"label\":null}]", "{\"total\":0}"} {
		var value svc.Result
		require.NoError(t, json.Unmarshal([]byte(wire), &value))
		original := value.Kind()
		encoded, err := json.Marshal(value, json.Deterministic(true))
		require.NoError(t, err)
		var decoded svc.Result
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		require.Equal(t, original, decoded.Kind())
		require.Equal(t, value, decoded)
	}
	empty := svc.NewResultItems(nil)
	encoded, err := json.Marshal(empty, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, "[]", string(encoded))
	_, err = json.Marshal(svc.NewResultPage(nil), json.Deterministic(true))
	require.Error(t, err)
	for _, wire := range []string{"{}", "null", "[null]", "[{\"name\":\"x\"}]", "[{\"Name\":\"ok\"}]", "[{\"name\":\"ok\",\"extra\":true}]"} {
		value := empty
		require.Error(t, json.Unmarshal([]byte(wire), &value))
		require.Equal(t, empty, value, "failed decode must not mutate the destination")
	}
	for _, wire := range []string{"[]", "[-1]", "[1]"} {
		var numbers svc.Numbers
		require.ErrorContains(t, json.Unmarshal([]byte(wire), &numbers), "matched 2 branches in schema")
	}
	_, err = json.Marshal(svc.NewNumbersSigned([]int32{-1}), json.Deterministic(true))
	require.ErrorContains(t, err, "matched 2 branches in schema")
	_, err = json.Marshal(svc.NewOverlapLeft(&svc.Left{}), json.Deterministic(true))
	require.ErrorContains(t, err, "matched 2 branches in schema")
	var overlapping svc.Overlap
	require.ErrorContains(t, json.Unmarshal([]byte("{}"), &overlapping), "matched 2 branches in schema")

}

func TestHTTPCodecs(t *testing.T) {
	var calls atomic.Int32
	type observation struct{ value any }
	var observed atomic.Value
	echo := func(_ context.Context, value any) (any, error) {
		calls.Add(1)
		observed.Store(observation{value})
		return value, nil
	}
	optional := func(_ context.Context, value any) (any, error) {
		calls.Add(1)
		observed.Store(observation{value})
		return nil, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{Echo: echo, Optional: optional}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	address, err := url.Parse(host.URL)
	require.NoError(t, err)
	c := client.NewClient(address.Scheme, address.Host, host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, wire := range []string{"[]", "[{\"name\":\"ok\"}]", "{\"total\":0}"} {
		payload, err := client.BuildEchoPayload(wire)
		require.NoError(t, err)
		result, err := c.Echo()(t.Context(), payload)
		require.NoError(t, err)
		require.Equal(t, payload, result)
	}
	for _, path := range []string{"/echo", "/optional"} {
		for _, tc := range []struct {
			body          string
			absent, valid bool
		}{
			{"", true, false}, {" \t\r\n", true, false}, {"{x}", false, false},
			{"[", false, false}, {"null", false, false}, {"[null]", false, false},
			{"{\"total\":0,\"extra\":1}", false, false},
			{"[{\"name\":\"x\"}]", false, false}, {"[]", false, true},
		} {
			before := calls.Load()
			response, err := host.Client().Post(host.URL+path, "application/json", strings.NewReader(tc.body))
			require.NoError(t, err)
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			if tc.valid || tc.absent && path == "/optional" {
				status := http.StatusOK
				if path == "/optional" {
					status = http.StatusNoContent
				}
				require.Equal(t, status, response.StatusCode, string(body))
				require.Equal(t, before+1, calls.Load())
				encoded, err := json.Marshal(observed.Load().(observation).value, json.Deterministic(true))
				require.NoError(t, err)
				if path == "/echo" {
					require.Equal(t, "[]", string(encoded))
				}
				if path == "/optional" && tc.absent {
					require.Equal(t, "{}", string(encoded))
				}
				continue
			}
			require.Equal(t, http.StatusBadRequest, response.StatusCode, string(body))
			require.Equal(t, before, calls.Load())
			var problem loomhttp.ProblemResponse
			require.NoError(t, json.Unmarshal(body, &problem))
			code, detail := "decode_payload", "invalid request body"
			if tc.absent {
				code, detail = "missing_payload", "validation error"
			}
			require.Equal(t, code, problem.Code)
			require.Equal(t, detail, problem.Detail)
		}
	}
}
`
