package testdata

// UnionResponseBodyHarness verifies generated success and error body constructors.
const UnionResponseBodyHarness = `package unionresponse_test

import (
	"context"
	"encoding/json/v2"
	c1 "example.com/unionresponse/gen/http/optional/client"
	h1 "example.com/unionresponse/gen/http/optional/server"
	c9 "example.com/unionresponse/gen/http/optional_mapped/client"
	h9 "example.com/unionresponse/gen/http/optional_mapped/server"
	c5 "example.com/unionresponse/gen/http/optional_named/client"
	h5 "example.com/unionresponse/gen/http/optional_named/server"
	c7 "example.com/unionresponse/gen/http/optional_named_nullable/client"
	h7 "example.com/unionresponse/gen/http/optional_named_nullable/server"
	c3 "example.com/unionresponse/gen/http/optional_nullable/client"
	h3 "example.com/unionresponse/gen/http/optional_nullable/server"
	c11 "example.com/unionresponse/gen/http/optional_viewed/client"
	h11 "example.com/unionresponse/gen/http/optional_viewed/server"
	c0 "example.com/unionresponse/gen/http/required/client"
	h0 "example.com/unionresponse/gen/http/required/server"
	c8 "example.com/unionresponse/gen/http/required_mapped/client"
	h8 "example.com/unionresponse/gen/http/required_mapped/server"
	c4 "example.com/unionresponse/gen/http/required_named/client"
	h4 "example.com/unionresponse/gen/http/required_named/server"
	c6 "example.com/unionresponse/gen/http/required_named_nullable/client"
	h6 "example.com/unionresponse/gen/http/required_named_nullable/server"
	c2 "example.com/unionresponse/gen/http/required_nullable/client"
	h2 "example.com/unionresponse/gen/http/required_nullable/server"
	c10 "example.com/unionresponse/gen/http/required_viewed/client"
	h10 "example.com/unionresponse/gen/http/required_viewed/server"
	s1 "example.com/unionresponse/gen/optional"
	s9 "example.com/unionresponse/gen/optional_mapped"
	s5 "example.com/unionresponse/gen/optional_named"
	s7 "example.com/unionresponse/gen/optional_named_nullable"
	s3 "example.com/unionresponse/gen/optional_nullable"
	s11 "example.com/unionresponse/gen/optional_viewed"
	s0 "example.com/unionresponse/gen/required"
	s8 "example.com/unionresponse/gen/required_mapped"
	s4 "example.com/unionresponse/gen/required_named"
	s6 "example.com/unionresponse/gen/required_named_nullable"
	s2 "example.com/unionresponse/gen/required_nullable"
	s10 "example.com/unionresponse/gen/required_viewed"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type responseCase struct {
	name    string
	result  func() any
	failure func() error
	mount   func(loom.Endpoint) http.Handler
	client  func(*httptest.Server) loom.Endpoint
	decode  func(*http.Response) (any, error)
	project func(any) (any, error)
}

func responseCases() []responseCase {
	return []responseCase{
		{name: "required",
			result: func() any {
				return new(s0.RequiredEnvelope)
			},
			failure: func() error {
				return new(s0.RequiredEnvelope)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h0.Mount(mux, h0.New(&s0.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c0.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c0.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "optional",
			result: func() any {
				return new(s1.OptionalEnvelope)
			},
			failure: func() error {
				return new(s1.OptionalEnvelope)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h1.Mount(mux, h1.New(&s1.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c1.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c1.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "required_nullable",
			result: func() any {
				return new(s2.RequiredNullableEnvelope)
			},
			failure: func() error {
				return new(s2.RequiredNullableEnvelope)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h2.Mount(mux, h2.New(&s2.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c2.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c2.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "optional_nullable",
			result: func() any {
				return new(s3.OptionalNullableEnvelope)
			},
			failure: func() error {
				return new(s3.OptionalNullableEnvelope)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h3.Mount(mux, h3.New(&s3.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c3.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c3.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "required_named",
			result: func() any {
				return new(s4.RequiredNamedEnvelope)
			},
			failure: func() error {
				return new(s4.RequiredNamedError)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h4.Mount(mux, h4.New(&s4.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c4.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c4.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "optional_named",
			result: func() any {
				return new(s5.OptionalNamedEnvelope)
			},
			failure: func() error {
				return new(s5.OptionalNamedError)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h5.Mount(mux, h5.New(&s5.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c5.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c5.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "required_named_nullable",
			result: func() any {
				return new(s6.RequiredNamedNullableEnvelope)
			},
			failure: func() error {
				return new(s6.RequiredNamedNullableError)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h6.Mount(mux, h6.New(&s6.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c6.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c6.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "optional_named_nullable",
			result: func() any {
				return new(s7.OptionalNamedNullableEnvelope)
			},
			failure: func() error {
				return new(s7.OptionalNamedNullableError)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h7.Mount(mux, h7.New(&s7.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c7.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c7.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "required_mapped",
			result: func() any {
				return new(s8.RequiredMappedEnvelope)
			},
			failure: func() error {
				return new(s8.RequiredMappedEnvelope)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h8.Mount(mux, h8.New(&s8.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c8.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c8.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "optional_mapped",
			result: func() any {
				return new(s9.OptionalMappedEnvelope)
			},
			failure: func() error {
				return new(s9.OptionalMappedEnvelope)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h9.Mount(mux, h9.New(&s9.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c9.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c9.DecodeShowResponse(loomhttp.ResponseDecoder, false),
		},
		{name: "required_viewed",
			result: func() any {
				return new(s10.RequiredViewed)
			},
			failure: func() error {
				return new(s10.RequiredViewedEnvelope)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h10.Mount(mux, h10.New(&s10.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c10.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c10.DecodeShowResponse(loomhttp.ResponseDecoder, false),
			project: func(value any) (any, error) {
				return s10.NewViewedRequiredViewed(value.(*s10.RequiredViewed), "default")
			},
		},
		{name: "optional_viewed",
			result: func() any {
				return new(s11.OptionalViewed)
			},
			failure: func() error {
				return new(s11.OptionalViewedEnvelope)
			},
			mount: func(endpoint loom.Endpoint) http.Handler {
				mux := loomhttp.NewMuxer()
				h11.Mount(mux, h11.New(&s11.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				return mux
			},
			client: func(host *httptest.Server) loom.Endpoint {
				return c11.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false).Show()
			},
			decode: c11.DecodeShowResponse(loomhttp.ResponseDecoder, false),
			project: func(value any) (any, error) {
				return s11.NewViewedOptionalViewed(value.(*s11.OptionalViewed), "default")
			},
		},
	}
}
func TestResponseUnionRoundTrips(t *testing.T) {
	for _, tc := range responseCases() {
		t.Run(tc.name, func(t *testing.T) {
			bodies := []string{"{\"type\":\"Leaf\",\"value\":{\"name\":\"alpha\"}}", "{\"type\":\"Other\",\"value\":{\"count\":7}}"}
			if strings.Contains(tc.name, "nullable") {
				bodies = append(bodies, "null")
			}
			for _, body := range bodies {
				for _, failure := range []bool{false, true} {
					result, serviceError := tc.result(), tc.failure()
					require.NoError(t, json.Unmarshal([]byte("{\"choice\":"+body+"}"), result))
					require.NoError(t, json.Unmarshal([]byte("{\"choice\":"+body+"}"), serviceError))
					var calls atomic.Int32
					endpoint := func(context.Context, any) (any, error) {
						calls.Add(1)
						if failure {
							return nil, serviceError
						}
						if tc.project != nil {
							return tc.project(result)
						}
						return result, nil
					}
					host := httptest.NewServer(tc.mount(endpoint))
					actual, err := tc.client(host)(t.Context(), nil)
					host.Close()
					require.EqualValues(t, 1, calls.Load())
					if failure {
						require.Equal(t, serviceError, err)
						require.Nil(t, actual)
					} else {
						require.NoError(t, err)
						require.Equal(t, result, actual)
					}
				}
			}
		})
	}
}
func TestResponseUnionInvalidBodies(t *testing.T) {
	for _, tc := range responseCases() {
		t.Run(tc.name, func(t *testing.T) {
			for _, status := range []int{http.StatusOK, http.StatusBadRequest} {
				for _, body := range []string{"", " \t\r\n", "{x}", "{\"type\":", "{\"type\":\"missing\",\"value\":{}}", "{\"type\":\"Leaf\",\"value\":{}}"} {
					response := &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
					value, err := tc.decode(response)
					require.Error(t, err, "status %d body %q", status, body)
					require.Nil(t, value)
				}
			}
		})
	}
}
`
