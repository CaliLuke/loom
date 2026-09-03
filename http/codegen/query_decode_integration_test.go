package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedHTTPServerQueryDecodeContract(t *testing.T) {
	const modulePath = "example.com/querydecodeit"

	root := RunHTTPDSL(t, queryDecodeDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	if err := os.WriteFile(filepath.Join(dir, "query_decode_test.go"), []byte(queryDecodeHarness), 0o600); err != nil {
		t.Fatalf("write query decode harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func queryDecodeDSL() {
	Service("query", func() {
		Method("Decode", func() {
			Payload(func() {
				Attribute("qp", String)
				Attribute("query_err", String)
				Attribute("value", String)
				Attribute("values", ArrayOf(String))
			})
			Result(String)
			HTTP(func() {
				GET("/decode")
				Param("qp")
				Param("query_err")
				Param("value")
				Param("values")
			})
		})
	})
}

const queryDecodeHarness = `package querydecodeit_test

import (
	"context"
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	query "example.com/querydecodeit/gen/query"
	server "example.com/querydecodeit/gen/http/query/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestQueryDecodeContract(t *testing.T) {
	tests := []struct {
		name         string
		rawQuery     string
		wantStatus   int
		wantCode     string
		wantDetail   string
		wantQP       *string
		wantQueryErr *string
		wantValue    *string
		wantValues   []string
		wantCalls    int32
	}{
		{
			name:       "raw semicolon is an explicit decode error",
			rawQuery:   "value=one;two",
			wantStatus: http.StatusBadRequest,
			wantCode:   loom.DecodePayload,
			wantDetail: "invalid query string",
		},
		{
			name:         "percent-encoded semicolon is preserved",
			rawQuery:     "qp=local&query_err=none&value=one%3Btwo",
			wantStatus:   http.StatusOK,
			wantQP:       stringPointer("local"),
			wantQueryErr: stringPointer("none"),
			wantValue:    stringPointer("one;two"),
			wantCalls:    1,
		},
		{
			name:       "repeated query keys are preserved",
			rawQuery:   "values=one&values=two%3Bthree",
			wantStatus: http.StatusOK,
			wantValues: []string{"one", "two;three"},
			wantCalls:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			var gotQP *string
			var gotQueryErr *string
			var gotValue *string
			var gotValues []string
			endpoints := &query.Endpoints{
				Decode: func(_ context.Context, rawPayload any) (any, error) {
					payload := rawPayload.(*query.DecodePayload)
					calls.Add(1)
					gotQP = payload.Qp
					gotQueryErr = payload.QueryErr
					gotValue = payload.Value
					gotValues = payload.Values
					return "ok", nil
				},
			}
			mux := loomhttp.NewMuxer()
			generated := server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
			server.Mount(mux, generated)
			httpServer := httptest.NewServer(mux)
			t.Cleanup(httpServer.Close)

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, httpServer.URL+"/decode?"+test.rawQuery, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			response, err := httpServer.Client().Do(request)
			if err != nil {
				t.Fatalf("send request: %v", err)
			}
			body, readErr := io.ReadAll(response.Body)
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Errorf("close response body: %v", closeErr)
			}
			if readErr != nil {
				t.Fatalf("read response body: %v", readErr)
			}

			if response.StatusCode != test.wantStatus {
				t.Errorf("status = %d, want %d: %s", response.StatusCode, test.wantStatus, body)
			}
			if test.wantCode != "" {
				if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, loomhttp.ProblemJSONContentType) {
					t.Errorf("content type = %q, want %q", contentType, loomhttp.ProblemJSONContentType)
				}
				var problem loomhttp.ProblemResponse
				if err := json.Unmarshal(body, &problem); err != nil {
					t.Fatalf("decode problem response: %v", err)
				}
				if problem.Code != test.wantCode {
					t.Errorf("problem code = %q, want %q", problem.Code, test.wantCode)
				}
				if problem.Detail != test.wantDetail {
					t.Errorf("problem detail = %q, want %q", problem.Detail, test.wantDetail)
				}
			}
			if got := calls.Load(); got != test.wantCalls {
				t.Errorf("service invocation count = %d, want %d", got, test.wantCalls)
			}
			if !reflect.DeepEqual(gotQP, test.wantQP) {
				t.Errorf("decoded qp = %#v, want %#v", gotQP, test.wantQP)
			}
			if !reflect.DeepEqual(gotQueryErr, test.wantQueryErr) {
				t.Errorf("decoded query_err = %#v, want %#v", gotQueryErr, test.wantQueryErr)
			}
			if !reflect.DeepEqual(gotValue, test.wantValue) {
				t.Errorf("decoded value = %#v, want %#v", gotValue, test.wantValue)
			}
			if !reflect.DeepEqual(gotValues, test.wantValues) {
				t.Errorf("decoded values = %#v, want %#v", gotValues, test.wantValues)
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}
`
