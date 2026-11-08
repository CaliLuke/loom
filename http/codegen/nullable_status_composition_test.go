package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestNullableStatusCompositionGenerated(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		value := Type("NullableStatus", func() {
			Attribute("status", Int, func() {
				Enum(201, 202)
			})
			Attribute("message", String)
			Attribute("trace", String)
			Attribute("session", String)
			Required("status", "message", "trace", "session")
			Nullable()
		})
		failure := Type("StatusFailure", func() {
			Attribute("status", Int, func() {
				Enum(400, 401)
			})
			Attribute("message", String)
			Required("status")
			Nullable()
		})
		Service("nullable_status", func() {
			Method("fail", func() {
				Error("failed", failure)
				HTTP(func() {
					GET("/fail")
					Response("failed", Status4XX, func() {
						StatusField("status")
						Body("message")
					})
				})
			})
			Method("read", func() {
				Result(value)
				HTTP(func() {
					GET("/status")
					for _, selector := range []any{StatusCreated, Status2XX} {
						Response(selector, func() {
							StatusField("status")
							if selector == StatusCreated {
								Header("trace:X-Exact")
							} else {
								Header("trace:X-Class")
							}
							Cookie("session")
							Body("message")
						})
					}
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/nullablestatus", root)
	source, err := os.ReadFile(filepath.Join(dir, "gen", "http", "nullable_status", "server", "encode_decode.go"))
	require.NoError(t, err)
	_, function, found := strings.Cut(string(source), "func EncodeReadResponse")
	require.True(t, found)
	function, _, _ = strings.Cut(function, "\nfunc ")
	require.Equal(t, 1, strings.Count(function, "res.Value()"), "selection and branch projection share one acquired native value")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "status_test.go"), []byte(nullableStatusCompositionHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const nullableStatusCompositionHarness = `package nullablestatus_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/nullablestatus/gen/http/nullable_status/client"
	server "example.com/nullablestatus/gen/http/nullable_status/server"
	service "example.com/nullablestatus/gen/nullable_status"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type countedWriter struct {
	*httptest.ResponseRecorder
	commits int
}

func (w *countedWriter) WriteHeader(status int) {
	w.commits++
	w.ResponseRecorder.WriteHeader(status)
}
func TestNullableStatusWire(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		state  string
		header string
		valid  bool
	}{
		{"exact", 201, "value", "X-Exact", true},
		{"class", 202, "value", "X-Class", true},
		{"enum", 203, "value", "", false},
		{"domain", 400, "value", "", false},
		{"null", 201, "null", "", false},
		{"absent", 201, "absent", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			native := service.NullableStatus{Status: tc.status, Message: "message", Trace: "trace", Session: "session"}
			value := loom.NullableValue(native)
			switch tc.state {
			case "null":
				value = loom.NullValue[service.NullableStatus]()
			case "absent":
				value = loom.Nullable[service.NullableStatus]{}
			}
			w := &countedWriter{ResponseRecorder: httptest.NewRecorder()}
			err := server.EncodeReadResponse(loomhttp.ResponseEncoder)(context.Background(), w, value)
			if !tc.valid {
				var violation *loomhttp.StatusViolationError
				require.ErrorAs(t, err, &violation)
				require.Zero(t, w.commits)
				require.Empty(t, w.Header())
				require.Empty(t, w.Body.String())
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, w.commits)
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, "trace", w.Header().Get(tc.header))
			response := w.Result()
			require.Len(t, response.Cookies(), 1)
			require.Equal(t, "session", response.Cookies()[0].Value)
			decoded, err := client.DecodeReadResponse(loomhttp.ResponseDecoder, false)(response)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			restored, ok := decoded.(loom.Nullable[service.NullableStatus]).Value()
			require.True(t, ok)
			require.Equal(t, native, restored)
		})
	}
}
func TestNullableStatusIncomingConstraint(t *testing.T) {
	response := &http.Response{
		StatusCode: 203,
		Header: http.Header{
			"Content-Type": {"application/json"},
			"X-Class":      {"trace"},
			"Set-Cookie":   {"session=session"},
		},
		Body: io.NopCloser(strings.NewReader("\"message\"")),
	}
	_, err := client.DecodeReadResponse(loomhttp.ResponseDecoder, false)(response)
	require.Error(t, err)
	require.Contains(t, err.Error(), "status")
	require.NoError(t, response.Body.Close())
}

func TestNullableStatusFormattedError(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		state     string
		formatted bool
		valid     bool
	}{
		{"value", 400, "value", true, true},
		{"other-owned-value", 401, "value", true, true},
		{"invalid-native", 402, "value", true, false},
		{"null", 400, "null", true, false},
		{"absent", 400, "absent", true, false},
		{"selected-body-absent", 400, "value", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := loom.NullableValue(service.StatusFailure{Status: tc.status})
			switch tc.state {
			case "null":
				payload = loom.NullValue[service.StatusFailure]()
			case "absent":
				payload = loom.Nullable[service.StatusFailure]{}
			}
			failure, err := service.NewFailedError(payload)
			require.NoError(t, err)
			calls := 0
			var formatter func(context.Context, error) loomhttp.StatusCoder
			if tc.formatted {
				formatter = func(context.Context, error) loomhttp.StatusCoder {
					calls++
					return &loomhttp.ProblemResponse{Status: 418, Detail: "replacement"}
				}
			}
			w := &countedWriter{ResponseRecorder: httptest.NewRecorder()}
			err = server.EncodeFailError(loomhttp.ResponseEncoder, formatter)(t.Context(), w, failure)
			if !tc.valid {
				if tc.name == "selected-body-absent" {
					require.ErrorContains(t, err, "selected response value is absent")
				} else {
					var violation *loomhttp.StatusViolationError
					require.ErrorAs(t, err, &violation)
				}
				require.Zero(t, calls)
				require.Zero(t, w.commits)
				require.Empty(t, w.Header())
				require.Empty(t, w.Body.String())
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			require.Equal(t, 1, w.commits)
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, "failed", w.Header().Get("loom-error"))
			require.Contains(t, w.Body.String(), "replacement")
		})
	}
}

func TestNullableErrorStatusIncomingConstraint(t *testing.T) {
	for _, status := range []int{400, 401, 402} {
		response := &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader("\"decoded\"")),
		}
		result, err := client.DecodeFailResponse(loomhttp.ResponseDecoder, false)(response)
		require.NoError(t, response.Body.Close())
		require.Nil(t, result)
		if status == 402 {
			require.ErrorContains(t, err, "status")
			continue
		}
		var failure *service.FailedError
		require.ErrorAs(t, err, &failure)
		payload, payloadError := failure.Payload()
		require.NoError(t, payloadError)
		value, present := payload.Value()
		require.True(t, present)
		require.Equal(t, status, value.Status)
		require.NotNil(t, value.Message)
		require.Equal(t, "decoded", *value.Message)
	}
}
`
