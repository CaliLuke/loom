package testdata

// BooleanMapUnionHarness checks CLI parsing and bidirectional generated HTTP
// codecs, including nested nullable maps in tagged union branches.
const BooleanMapUnionHarness = `package unionkeys_test

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

	client "example.com/unionkeys/gen/http/probe/client"
	server "example.com/unionkeys/gen/http/probe/server"
	svc "example.com/unionkeys/gen/probe"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestBooleanMapUnionRoundTrips(t *testing.T) {
	var calls atomic.Int32
	echo := func(_ context.Context, value any) (any, error) {
		calls.Add(1)
		return value, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{Tagged: echo}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	address, err := url.Parse(host.URL)
	require.NoError(t, err)
	c := client.NewClient(address.Scheme, address.Host, host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, tc := range []struct {
		name, body string
		build      func(string) (any, error)
		endpoint   loom.Endpoint
	}{
		{"tagged map", "{\"type\":\"flags\",\"value\":{\"false\":\"off\",\"true\":\"on\"}}", func(body string) (any, error) {
			return client.BuildTaggedPayload(body)
		}, c.Tagged()},
		{"tagged nested", "{\"type\":\"nested\",\"value\":{\"marker\":\"present\",\"flags\":{\"false\":true,\"true\":false}}}", func(body string) (any, error) {
			return client.BuildTaggedPayload(body)
		}, c.Tagged()},

	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := tc.build(tc.body)
			require.NoError(t, err)
			before := calls.Load()
			result, err := tc.endpoint(t.Context(), payload)
			require.NoError(t, err)
			require.Equal(t, before+1, calls.Load())
			require.Equal(t, payload, result)
			data, err := json.Marshal(payload, json.Deterministic(true))
			require.NoError(t, err)
			require.JSONEq(t, "{\"choice\":"+tc.body+"}", string(data))
		})
	}
}

func TestBooleanMapUnionBodyErrors(t *testing.T) {
	var calls atomic.Int32
	var observed atomic.Value
	consume := func(_ context.Context, value any) (any, error) {
		data, err := json.Marshal(value, json.Deterministic(true))
		if err != nil {
			return nil, err
		}
		observed.Store(string(data))
		calls.Add(1)
		return nil, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{Required: consume, Optional: consume}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	const valid = "{\"type\":\"flags\",\"value\":{\"false\":\"off\",\"true\":\"on\"}}"
	for _, path := range []string{"/required", "/optional"} {
		for _, tc := range []struct {
			name, body    string
			absent, valid bool
		}{
			{"empty", "", true, false},
			{"whitespace", " \t\r\n", true, false},
			{"malformed", "{x}", false, false},
			{"truncated", "{\"type\":", false, false},
			{"noncanonical", "{\"type\":\"flags\",\"value\":{\"TRUE\":\"on\"}}", false, false},
			{"duplicate", "{\"type\":\"flags\",\"value\":{\"true\":\"on\",\"true\":\"off\"}}", false, false},
			{"quoted boolean", "{\"type\":\"nested\",\"value\":{\"marker\":\"present\",\"flags\":{\"true\":\"false\"}}}", false, false},
			{"valid", valid, false, true},
		} {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				before := calls.Load()
				response, err := host.Client().Post(host.URL+path, "application/json", strings.NewReader(tc.body))
				require.NoError(t, err)
				data, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				if tc.valid || tc.absent && path == "/optional" {
					require.Equal(t, http.StatusNoContent, response.StatusCode, string(data))
					require.Equal(t, before+1, calls.Load())
					want := "{}"
					if tc.valid {
						want = "{\"choice\":" + valid + "}"
					}
					require.JSONEq(t, want, observed.Load().(string))
					return
				}
				require.Equal(t, http.StatusBadRequest, response.StatusCode, string(data))
				require.Equal(t, before, calls.Load(), "service invoked for invalid body")
				var problem loomhttp.ProblemResponse
				require.NoError(t, json.Unmarshal(data, &problem))
				code, detail := "decode_payload", "invalid request body"
				if tc.absent {
					code, detail = "missing_payload", "validation error"
				}
				require.Equal(t, code, problem.Code)
				require.Equal(t, detail, problem.Detail)
			})
		}
	}
}
`
