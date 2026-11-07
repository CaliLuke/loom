package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestNullableDefaultErrorGenerated(t *testing.T) {
	for _, mapped := range []bool{false, true} {
		name := "unmapped"
		if mapped {
			name = "mapped"
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				record := dsl.Type("Record", func() {
					dsl.Nullable()
					dsl.Attribute("message", dsl.String)
					dsl.Required("message")
				})
				dsl.Service("defaults", func() {
					dsl.Method("ordinary", func() {
						dsl.Result(record)
						dsl.HTTP(func() {
							dsl.GET("/ordinary")
							dsl.Response(dsl.StatusOK, func() {
								dsl.Body(func() {
									dsl.Attribute("message")
								})
							})
						})
					})
					for _, method := range []string{"call", "pruned", "selected", "header", "cookie"} {
						dsl.Method(method, func() {
							dsl.Error("missing", func() {
								dsl.Nullable()
								dsl.Temporary()
							})
							dsl.HTTP(func() {
								dsl.GET("/" + method)
								if mapped {
									dsl.Response("missing", dsl.StatusNotFound, func() {
										if method == "pruned" {
											dsl.Body(func() {
												dsl.Attribute("detail")
											})
										}
										if method == "selected" {
											dsl.Body("detail")
										}
										if method == "header" {
											dsl.Header("retry_hint:X-Retry")
										}
										if method == "cookie" {
											dsl.Cookie("retry_hint:retry")
										}
									})
								}
							})
						})
					}
				})
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/defaults", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "gen/defaults/owner_internal_test.go"), []byte(nullableDefaultPrivateHarness), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "default_test.go"), []byte(nullableDefaultOwnerHarness), 0o600))
			if mapped {
				specs, err := openapiv3.Files(root)
				require.NoError(t, err)
				for _, spec := range specs {
					output, err := spec.Render(dir)
					require.NoError(t, err)
					data, err := os.ReadFile(output)
					require.NoError(t, err)
					document, err := libopenapi.NewDocument(data)
					require.NoError(t, err)
					_, err = document.BuildV3Model()
					require.NoError(t, err)
				}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte(nullableDefaultHTTPHarness), 0o600))
			}
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}

const nullableDefaultOwnerHarness = `package defaults_test

import (
	"errors"
	"testing"

	service "example.com/defaults/gen/defaults"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestDefaultOwner(t *testing.T) {
	cause := errors.New("failure")
	failure := service.MakeMissing(cause)
	var native *loom.ServiceError
	require.ErrorAs(t, failure, &native)
	require.True(t, native.Temporary)
	require.ErrorIs(t, failure, cause)
	native.Message = "mutated"
	remedy := &loom.ErrorRemedy{Code: "retry", SafeMessage: "safe", RetryHint: "later"}
	loom.WithErrorRemedy(native, remedy)
	require.Same(t, remedy, failure.LoomErrorRemedy())
	require.Equal(t, "safe", loom.ErrorSafeMessage(failure))
	payload, err := failure.Payload()
	require.NoError(t, err)
	value, ok := payload.Value()
	require.True(t, ok)
	require.Equal(t, "mutated", value.Message)
	native.Name = "different"
	require.Nil(t, loom.ResolveErrorResponse(failure).ServiceError)
	_, err = failure.Payload()
	require.Error(t, err)
	null, err := service.NewMissingError(loom.NullValue[loom.ServiceError]())
	require.NoError(t, err)
	native = nil
	require.False(t, errors.As(null, &native))
	_, err = service.NewMissingError(loom.NullableValue(loom.ServiceError{Name: "different"}))
	require.Error(t, err)
}
`

const nullableDefaultHTTPHarness = `package defaults_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	service "example.com/defaults/gen/defaults"
	client "example.com/defaults/gen/http/defaults/client"
	server "example.com/defaults/gen/http/defaults/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestNullableProblemWire(t *testing.T) {
	for _, null := range []bool{false, true} {
		failure := service.MakeMissing(errors.New("details"))
		if null {
			var err error
			failure, err = service.NewMissingError(loom.NullValue[loom.ServiceError]())
			require.NoError(t, err)
		}
		wrapped := fmt.Errorf("diagnostic: %w", failure)
		rec := httptest.NewRecorder()
		require.NoError(t, server.EncodeCallError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, wrapped))
		require.Equal(t, 404, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		require.Equal(t, "missing", rec.Header().Get("loom-error"))
		_, got := client.DecodeCallResponse(loomhttp.ResponseDecoder, false)(rec.Result())
		var decoded *service.MissingError
		require.ErrorAs(t, got, &decoded)
		payload, err := decoded.Payload()
		require.NoError(t, err)
		require.Equal(t, null, payload.IsNull())
		if null {
			require.Equal(t, "null", strings.TrimSpace(rec.Body.String()))
			var native *loom.ServiceError
			require.False(t, errors.As(decoded, &native))
		} else {
			value, ok := payload.Value()
			require.True(t, ok)
			require.Equal(t, "missing", value.Name)
			require.Equal(t, "details", value.Message)
			require.True(t, value.Temporary)
			original, err := failure.Payload()
			require.NoError(t, err)
			originalValue, ok := original.Value()
			require.True(t, ok)
			require.Equal(t, originalValue.ID, value.ID)
		}
	}
	absent, err := service.NewMissingError(loom.Nullable[loom.ServiceError]{})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	require.Error(t, server.EncodeCallError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, absent))
	require.Empty(t, rec.Body.String())
	for _, code := range []string{"other", ""} {
		body := fmt.Sprintf("{\"type\":\"urn:test\",\"title\":\"failure\",\"status\":404,\"detail\":\"detail\",\"instance\":\"urn:one\",\"code\":%q,\"timeout\":false,\"temporary\":false,\"fault\":false}", code)
		response := &http.Response{StatusCode: 404, Header: http.Header{"Loom-Error": []string{"missing"}, "Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
		_, got := client.DecodeCallResponse(loomhttp.ResponseDecoder, false)(response)
		var invalid *loomhttp.ClientError
		require.ErrorAs(t, got, &invalid)
		require.Equal(t, "validation_error", invalid.Name)
	}
}

func TestPrunedNullableProblem(t *testing.T) {
	failure := service.MakeMissing(errors.New("details"))
	rec := httptest.NewRecorder()
	require.NoError(t, server.EncodePrunedError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, failure))
	require.JSONEq(t, "{\"detail\":\"details\"}", rec.Body.String())
	_, got := client.DecodePrunedResponse(loomhttp.ResponseDecoder, false)(rec.Result())
	var decoded *service.MissingError
	require.ErrorAs(t, got, &decoded)
	payload, err := decoded.Payload()
	require.NoError(t, err)
	value, ok := payload.Value()
	require.True(t, ok)
	require.Equal(t, "missing", value.Name)
	require.Equal(t, "details", value.Message)
	require.False(t, value.Temporary, "omitted traits are not inferred")
	rec = httptest.NewRecorder()
	require.NoError(t, server.EncodeSelectedError(loomhttp.ResponseEncoder, nil)(context.Background(), rec, failure))
	require.JSONEq(t, "\"details\"", rec.Body.String())
	_, got = client.DecodeSelectedResponse(loomhttp.ResponseDecoder, false)(rec.Result())
	require.ErrorAs(t, got, &decoded)
	payload, err = decoded.Payload()
	require.NoError(t, err)
	value, ok = payload.Value()
	require.True(t, ok)
	require.True(t, value.Temporary, "conveyed headers preserve traits")
}

func TestNonnullableObjectProjection(t *testing.T) {
	for _, value := range []loom.Nullable[service.Record]{loom.NullValue[service.Record](), {}} {
		recorder := httptest.NewRecorder()
		require.Error(t, server.EncodeOrdinaryResponse(loomhttp.ResponseEncoder)(context.Background(), recorder, value))
		require.Empty(t, recorder.Body.String())
	}
	recorder := httptest.NewRecorder()
	value := loom.NullableValue(service.Record{Message: "ordinary"})
	require.NoError(t, server.EncodeOrdinaryResponse(loomhttp.ResponseEncoder)(context.Background(), recorder, value))
	require.JSONEq(t, "{\"message\":\"ordinary\"}", recorder.Body.String())
	decoded, err := client.DecodeOrdinaryResponse(loomhttp.ResponseDecoder, false)(recorder.Result())
	require.NoError(t, err)
	payload := decoded.(loom.Nullable[service.Record])
	record, ok := payload.Value()
	require.True(t, ok)
	require.Equal(t, "ordinary", record.Message)
	null, err := service.NewMissingError(loom.NullValue[loom.ServiceError]())
	require.NoError(t, err)
	recorder = httptest.NewRecorder()
	require.Error(t, server.EncodePrunedError(loomhttp.ResponseEncoder, nil)(context.Background(), recorder, null))
	require.Empty(t, recorder.Body.String())
}

func TestNullableProblemMetadata(t *testing.T) {
	for _, cookie := range []bool{false, true} {
		for _, null := range []bool{false, true} {
			failure := service.MakeMissing(errors.New("details"))
			var native *loom.ServiceError
			require.ErrorAs(t, failure, &native)
			loom.WithErrorRemedy(native, &loom.ErrorRemedy{SafeMessage: "safe", RetryHint: "later"})
			if null {
				var err error
				failure, err = service.NewMissingError(loom.NullValue[loom.ServiceError]())
				require.NoError(t, err)
			}
			recorder := httptest.NewRecorder()
			encode := server.EncodeHeaderError(loomhttp.ResponseEncoder, nil)
			decode := client.DecodeHeaderResponse(loomhttp.ResponseDecoder, false)
			if cookie {
				encode = server.EncodeCookieError(loomhttp.ResponseEncoder, nil)
				decode = client.DecodeCookieResponse(loomhttp.ResponseDecoder, false)
			}
			require.NoError(t, encode(context.Background(), recorder, failure))
			_, got := decode(recorder.Result())
			var decoded *service.MissingError
			require.ErrorAs(t, got, &decoded)
			payload, err := decoded.Payload()
			require.NoError(t, err)
			require.Equal(t, null, payload.IsNull())
			if !null {
				require.Equal(t, "later", decoded.LoomErrorRemedy().RetryHint)
				require.Equal(t, "safe", decoded.Error())
			}
			response := &http.Response{StatusCode: 404, Header: http.Header{"Loom-Error": []string{"missing"}, "Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader("null"))}
			if cookie {
				response.Header.Set("Set-Cookie", "retry=later")
			} else {
				response.Header.Set("X-Retry", "later")
			}
			_, got = decode(response)
			var invalid *loomhttp.ClientError
			require.ErrorAs(t, got, &invalid)
			require.Contains(t, invalid.Message, "null response body cannot carry headers or cookies")
		}
	}
}
`

const nullableDefaultPrivateHarness = `package defaults

import (
	"errors"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestInvalidPrivateOccurrence(t *testing.T) {
	var native *loom.ServiceError
	invalid := &MissingError{payload: loom.NullableValue(native)}
	_, err := invalid.Payload()
	require.Error(t, err)
	require.Nil(t, invalid.LoomServiceError())
	require.Nil(t, loom.ResolveErrorResponse(invalid).ServiceError)
	require.False(t, errors.As(invalid, &native))
	var absentOwner *MissingError
	_, err = absentOwner.Payload()
	require.Error(t, err)
	require.Nil(t, absentOwner.LoomServiceError())
}
`
