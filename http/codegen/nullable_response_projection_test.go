package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestNullableResponseProjectionGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.NullableResponseProjectionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/nullableprojection", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "projection_test.go"), []byte(nullableProjectionHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const nullableProjectionHarness = `package nullableprojection_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/nullableprojection/gen/http/projection/client"
	server "example.com/nullableprojection/gen/http/projection/server"
	service "example.com/nullableprojection/gen/projection"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type responseWriter struct {
	*httptest.ResponseRecorder
	commits int
}

func (w *responseWriter) WriteHeader(status int) {
	w.commits++
	w.ResponseRecorder.WriteHeader(status)
}

type projectionCase struct {
	name        string
	encode      func(context.Context, http.ResponseWriter, any) error
	encodeError func(context.Context, http.ResponseWriter, error) error
	decode      func(*http.Response) (any, error)
	failure     func(loom.Nullable[service.Parent]) (error, error)
	payload     func(error) (loom.Nullable[service.Parent], error)
}

func projectionCases() []projectionCase {
	return []projectionCase{
		{"selected", server.EncodeSelectedResponse(loomhttp.ResponseEncoder), server.EncodeSelectedError(loomhttp.ResponseEncoder, nil), client.DecodeSelectedResponse(loomhttp.ResponseDecoder, false), func(v loom.Nullable[service.Parent]) (error, error) { return service.NewSelectedError(v) }, func(err error) (loom.Nullable[service.Parent], error) { return err.(*service.SelectedError).Payload() }},
		{"whole", server.EncodeWholeResponse(loomhttp.ResponseEncoder), server.EncodeWholeError(loomhttp.ResponseEncoder, nil), client.DecodeWholeResponse(loomhttp.ResponseDecoder, false), func(v loom.Nullable[service.Parent]) (error, error) { return service.NewWholeError(v) }, func(err error) (loom.Nullable[service.Parent], error) { return err.(*service.WholeError).Payload() }},
		{"required", server.EncodeRequiredResponse(loomhttp.ResponseEncoder), server.EncodeRequiredError(loomhttp.ResponseEncoder, nil), client.DecodeRequiredResponse(loomhttp.ResponseDecoder, false), func(v loom.Nullable[service.Parent]) (error, error) { return service.NewRequiredError(v) }, func(err error) (loom.Nullable[service.Parent], error) { return err.(*service.RequiredError).Payload() }},
	}
}
func TestFormatterProjectionOwnership(t *testing.T) {
	for _, state := range []string{"null parent", "absent child", "value", "absent parent"} {
		t.Run(state, func(t *testing.T) {
			payload := loom.NullableValue(service.FormattedChild{Detail: loom.NullableValue(7)})
			switch state {
			case "null parent":
				payload.SetNull()
			case "absent child":
				payload.SetValue(service.FormattedChild{})
			case "absent parent":
				payload = loom.Nullable[service.FormattedChild]{}
			}
			failure, err := service.NewFormattedChildError(payload)
			require.NoError(t, err)
			calls := 0
			formatter := func(_ context.Context, original error) loomhttp.StatusCoder {
				calls++
				require.Same(t, failure, original)
				return &loomhttp.ProblemResponse{Status: 400, Detail: "replacement"}
			}
			writer := &responseWriter{ResponseRecorder: httptest.NewRecorder()}
			err = server.EncodeFormattedChildError(loomhttp.ResponseEncoder, formatter)(t.Context(), writer, failure)
			if state == "absent parent" {
				require.Error(t, err)
				require.Zero(t, calls)
				require.Zero(t, writer.commits)
				require.Empty(t, writer.Header())
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			require.Equal(t, 1, writer.commits)
			require.Equal(t, 400, writer.Code)
			require.Contains(t, writer.Body.String(), "replacement")
		})
	}
	t.Run("metadata still needs concrete parent", func(t *testing.T) {
		failure, err := service.NewSelectedError(loom.NullValue[service.Parent]())
		require.NoError(t, err)
		calls := 0
		formatter := func(context.Context, error) loomhttp.StatusCoder {
			calls++
			return &loomhttp.ProblemResponse{Status: 400}
		}
		writer := &responseWriter{ResponseRecorder: httptest.NewRecorder()}
		require.Error(t, server.EncodeSelectedError(loomhttp.ResponseEncoder, formatter)(t.Context(), writer, failure))
		require.Zero(t, calls)
		require.Zero(t, writer.commits)
		require.Empty(t, writer.Header())
		require.Empty(t, writer.Body.String())
		requiredFailure, err := service.NewRequiredError(loom.NullValue[service.Parent]())
		require.NoError(t, err)
		writer = &responseWriter{ResponseRecorder: httptest.NewRecorder()}
		require.Error(t, server.EncodeRequiredError(loomhttp.ResponseEncoder, formatter)(t.Context(), writer, requiredFailure))
		require.Zero(t, calls)
		require.Zero(t, writer.commits)
		require.Empty(t, writer.Header())
		require.Empty(t, writer.Body.String())
	})
}

func TestProjectionRoundTrip(t *testing.T) {
	for _, c := range projectionCases() {
		for _, failure := range []bool{false, true} {
			for _, state := range []string{"value", "null", "absent"} {
				t.Run(fmt.Sprintf("%s/error=%v/%s", c.name, failure, state), func(t *testing.T) {
					header, count := "info", int8(42)
					payload := loom.NullableValue(service.Parent{Message: "message", Header: &header, Count: &count, Session: "session"})
					switch state {
					case "null":
						payload = loom.NullValue[service.Parent]()
					case "absent":
						payload = loom.Nullable[service.Parent]{}
					}
					w := &responseWriter{ResponseRecorder: httptest.NewRecorder()}
					var err error
					if failure {
						value, e := c.failure(payload)
						require.NoError(t, e)
						err = c.encodeError(context.Background(), w, value)
					} else {
						err = c.encode(context.Background(), w, payload)
					}
					if state == "absent" || state == "null" && c.name != "whole" {
						require.Error(t, err)
						require.Zero(t, w.commits)
						require.Empty(t, w.Header())
						require.Zero(t, w.Body.Len())
						return
					}
					require.NoError(t, err)
					require.Equal(t, 1, w.commits)
					status := 200
					if failure {
						status = 400
					}
					require.Equal(t, status, w.Code)
					if state == "null" {
						require.Equal(t, "null", strings.TrimSpace(w.Body.String()))
						require.Empty(t, w.Header().Values("Set-Cookie"))
						require.Empty(t, w.Header().Values("X-Info"))
					} else {
						require.Equal(t, "info", w.Header().Get("X-Info"))
						require.Equal(t, "42", w.Header().Get("X-Count"))
						require.Contains(t, w.Header().Get("Set-Cookie"), "Session=session")
					}
					decoded, e := c.decode(w.Result())
					var round loom.Nullable[service.Parent]
					if failure {
						require.Error(t, e)
						round, err = c.payload(e)
						require.NoError(t, err)
					} else {
						require.NoError(t, e)
						round = decoded.(loom.Nullable[service.Parent])
					}
					require.True(t, round.Present())
					require.Equal(t, state == "null", round.IsNull())
					if state == "value" {
						value, ok := round.Value()
						require.True(t, ok)
						require.Equal(t, "message", value.Message)
						require.Equal(t, &header, value.Header)
						require.Equal(t, &count, value.Count)
						require.Equal(t, "session", value.Session)
					}
				})
			}
		}
	}
}
func TestWholeNullMetadataPresence(t *testing.T) {
	for _, status := range []int{200, 400} {
		for _, test := range []struct {
			name    string
			headers http.Header
			valid   bool
		}{
			{"absent", nil, true},
			{"empty header", http.Header{"X-Info": []string{""}}, false},
			{"header", http.Header{"X-Info": []string{"value"}}, false},
			{"zero count", http.Header{"X-Count": []string{"0"}}, false},
			{"cookie", http.Header{"Set-Cookie": []string{"Session="}}, false},
		} {
			t.Run(fmt.Sprintf("%d/%s", status, test.name), func(t *testing.T) {
				headers := test.headers.Clone()
				if headers == nil {
					headers = make(http.Header)
				}
				if status == 400 {
					headers.Set("loom-error", "whole")
				}
				res := &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader("null"))}
				value, err := client.DecodeWholeResponse(loomhttp.ResponseDecoder, false)(res)
				if !test.valid {
					require.Error(t, err)
					require.Nil(t, value)
					if status == 400 {
						_, designed := err.(*service.WholeError)
						require.False(t, designed)
					}
					return
				}
				var payload loom.Nullable[service.Parent]
				if status == 400 {
					var e *service.WholeError
					require.ErrorAs(t, err, &e)
					payload, err = e.Payload()
				} else {
					payload = value.(loom.Nullable[service.Parent])
				}
				require.NoError(t, err)
				require.True(t, payload.IsNull())
			})
		}
	}
}
func TestNullableSelectedChild(t *testing.T) {
	for _, null := range []bool{false, true} {
		for _, failure := range []bool{false, true} {
			child := loom.NullableValue(7)
			if null {
				child = loom.NullValue[int]()
			}
			payload := loom.NullableValue(service.Child{Detail: child})
			w := httptest.NewRecorder()
			if failure {
				value, err := service.NewChildError(payload)
				require.NoError(t, err)
				require.NoError(t, server.EncodeChildError(loomhttp.ResponseEncoder, nil)(context.Background(), w, value))
			} else {
				require.NoError(t, server.EncodeChildResponse(loomhttp.ResponseEncoder)(context.Background(), w, payload))
			}
			decoded, err := client.DecodeChildResponse(loomhttp.ResponseDecoder, false)(w.Result())
			var round loom.Nullable[service.Child]
			if failure {
				var e *service.ChildError
				require.ErrorAs(t, err, &e)
				round, err = e.Payload()
			} else {
				round = decoded.(loom.Nullable[service.Child])
			}
			require.NoError(t, err)
			actual, ok := round.Value()
			require.True(t, ok)
			require.Equal(t, null, actual.Detail.IsNull())
			if !null {
				v, ok := actual.Detail.Value()
				require.True(t, ok)
				require.Equal(t, 7, v)
			}
		}
	}
	for _, payload := range []loom.Nullable[service.Child]{loom.NullValue[service.Child](), {}} {
		w := &responseWriter{ResponseRecorder: httptest.NewRecorder()}
		require.Error(t, server.EncodeChildResponse(loomhttp.ResponseEncoder)(context.Background(), w, payload))
		require.Zero(t, w.commits)
		require.Empty(t, w.Header())
	}
}
func TestProjectionIncomingRejections(t *testing.T) {
	for _, test := range []struct {
		name, body string
		headers    http.Header
		decode     func(*http.Response) (any, error)
	}{
		{"selected null", "null", nil, client.DecodeSelectedResponse(loomhttp.ResponseDecoder, false)},
		{"empty", "", nil, client.DecodeWholeResponse(loomhttp.ResponseDecoder, false)},
		{"malformed", "{", nil, client.DecodeWholeResponse(loomhttp.ResponseDecoder, false)},
		{"missing required header", "null", nil, client.DecodeRequiredResponse(loomhttp.ResponseDecoder, false)},
		{"native range", "\"message\"", http.Header{"X-Count": []string{"128"}}, client.DecodeSelectedResponse(loomhttp.ResponseDecoder, false)},
	} {
		t.Run(test.name, func(t *testing.T) {
			res := &http.Response{StatusCode: 200, Header: test.headers, Body: io.NopCloser(strings.NewReader(test.body))}
			value, err := test.decode(res)
			require.Error(t, err)
			require.Nil(t, value)
		})
	}
}

func TestSelectedObjectProjection(t *testing.T) {
	for _, failure := range []bool{false, true} {
		header := "info"
		payload := loom.NullableValue(service.ObjectParent{Detail: &service.Parent{Message: "nested", Session: "session"}, Header: &header})
		w := httptest.NewRecorder()
		if failure {
			value, err := service.NewObjectError(payload)
			require.NoError(t, err)
			require.NoError(t, server.EncodeObjectError(loomhttp.ResponseEncoder, nil)(context.Background(), w, value))
		} else {
			require.NoError(t, server.EncodeObjectResponse(loomhttp.ResponseEncoder)(context.Background(), w, payload))
		}
		decoded, err := client.DecodeObjectResponse(loomhttp.ResponseDecoder, false)(w.Result())
		var round loom.Nullable[service.ObjectParent]
		if failure {
			var e *service.ObjectError
			require.ErrorAs(t, err, &e)
			round, err = e.Payload()
		} else {
			round = decoded.(loom.Nullable[service.ObjectParent])
		}
		require.NoError(t, err)
		actual, ok := round.Value()
		require.True(t, ok)
		require.Equal(t, "nested", actual.Detail.Message)
		require.Equal(t, &header, actual.Header)
	}
}
`
