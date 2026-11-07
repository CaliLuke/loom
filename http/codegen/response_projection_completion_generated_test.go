package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponseProjectionCompletionGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.NullableProjectionCompletionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/completion", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "completion_test.go"), []byte(responseProjectionCompletionHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const responseProjectionCompletionHarness = `package completion_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	service "example.com/completion/gen/completion"
	views "example.com/completion/gen/completion/views"
	client "example.com/completion/gen/http/completion/client"
	server "example.com/completion/gen/http/completion/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type writer struct {
	*httptest.ResponseRecorder
	commits int
}

func (w *writer) WriteHeader(code int) {
	w.commits++
	w.ResponseRecorder.WriteHeader(code)
}
func freshWriter() *writer {
	return &writer{ResponseRecorder: httptest.NewRecorder()}
}
func TestSelectedChildPresence(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		for _, state := range []string{"absent", "empty", "value", "nil-parent"} {
			t.Run(fmt.Sprintf("nullable=%v/%s", nullable, state), func(t *testing.T) {
				value := &service.OptionalParent{Header: new("must-not-leak")}
				if state == "empty" {
					value.Value = new("")
				}
				if state == "value" {
					value.Value = new("value")
				}
				var result any = value
				encode := server.EncodeOptionalResponse(loomhttp.ResponseEncoder)
				if nullable {
					result = loom.NullableValue(*value)
					encode = server.EncodeNullableOptionalResponse(loomhttp.ResponseEncoder)
				}
				if state == "nil-parent" {
					if nullable {
						result = loom.NullValue[service.OptionalParent]()
					} else {
						result = (*service.OptionalParent)(nil)
					}
				}
				w := freshWriter()
				var err error
				require.NotPanics(t, func() {
					err = encode(context.Background(), w, result)
				})
				if state == "absent" || state == "nil-parent" {
					require.Error(t, err)
					require.Zero(t, w.commits)
					require.Empty(t, w.Header())
					require.Zero(t, w.Body.Len())
					return
				}
				require.NoError(t, err)
				require.Equal(t, 1, w.commits)
				require.Equal(t, "must-not-leak", w.Header().Get("X-Info"))
				require.Equal(t, fmt.Sprintf("%q", *value.Value), strings.TrimSpace(w.Body.String()))
			})
		}
	}
}
func TestMetadataOnlyRoundTrip(t *testing.T) {
	for _, failure := range []bool{false, true} {
		for _, state := range []string{"value", "empty", "null", "absent"} {
			t.Run(fmt.Sprintf("error=%v/%s", failure, state), func(t *testing.T) {
				payload := loom.NullableValue(service.Metadata{Header: new("info"), PayloadError: new("payload"), ErrorOwner: new("owner"), Session: "session"})
				if state == "empty" {
					payload = loom.NullableValue(service.Metadata{})
				}
				if state == "null" {
					payload = loom.NullValue[service.Metadata]()
				}
				if state == "absent" {
					payload = loom.Nullable[service.Metadata]{}
				}
				w := freshWriter()
				var err error
				if failure {
					owned, e := service.NewMetadataError(payload)
					require.NoError(t, e)
					err = server.EncodeMetadataError(loomhttp.ResponseEncoder, nil)(context.Background(), w, owned)
				} else {
					err = server.EncodeMetadataResponse(loomhttp.ResponseEncoder)(context.Background(), w, payload)
				}
				if state == "null" || state == "absent" {
					require.Error(t, err)
					require.Zero(t, w.commits)
					require.Empty(t, w.Header())
					require.Zero(t, w.Body.Len())
					return
				}
				require.NoError(t, err)
				require.Zero(t, w.Body.Len())
				result, e := client.DecodeMetadataResponse(loomhttp.ResponseDecoder, false)(w.Result())
				var decoded loom.Nullable[service.Metadata]
				if failure {
					var owned *service.MetadataError
					require.ErrorAs(t, e, &owned)
					decoded, e = owned.Payload()
				} else {
					require.NoError(t, e)
					decoded = result.(loom.Nullable[service.Metadata])
				}
				require.NoError(t, e)
				concrete, ok := decoded.Value()
				require.True(t, ok)
				if state == "value" {
					require.Equal(t, "info", *concrete.Header)
					require.Equal(t, "payload", *concrete.PayloadError)
					require.Equal(t, "owner", *concrete.ErrorOwner)
				} else {
					require.Nil(t, concrete.Header)
				}
			})
		}
	}
}
func TestMetadataOnlyAbsentBindings(t *testing.T) {
	for _, code := range []int{200, 400} {
		headers := make(http.Header)
		if code == 400 {
			headers.Set("loom-error", "metadata")
		}
		response := &http.Response{StatusCode: code, Header: headers, Body: http.NoBody}
		value, err := client.DecodeMetadataResponse(loomhttp.ResponseDecoder, false)(response)
		var payload loom.Nullable[service.Metadata]
		if code == 400 {
			var owned *service.MetadataError
			require.ErrorAs(t, err, &owned)
			payload, err = owned.Payload()
		} else {
			require.NoError(t, err)
			payload = value.(loom.Nullable[service.Metadata])
		}
		require.NoError(t, err)
		concrete, ok := payload.Value()
		require.True(t, ok)
		require.Nil(t, concrete.Header)
		require.Equal(t, "fallback", concrete.Session)
		response = &http.Response{StatusCode: code, Header: headers, Body: http.NoBody}
		if code == 400 {
			headers.Set("loom-error", "required_metadata")
		}
		_, err = client.DecodeRequiredMetadataResponse(loomhttp.ResponseDecoder, false)(response)
		require.Error(t, err)
	}
}

type implementation struct {
	result loom.Nullable[service.Viewed]
}

func (i *implementation) Whole(context.Context) (loom.Nullable[service.Viewed], string, error) {
	return i.result, "default", nil
}
func (i *implementation) DynamicSelected(context.Context) (loom.Nullable[service.Viewed], string, error) {
	return i.result, "default", nil
}
func (i *implementation) Selected(context.Context) (loom.Nullable[service.Viewed], error) {
	return i.result, nil
}
func (i *implementation) Tagged(context.Context) (loom.Nullable[service.Viewed], string, error) {
	return i.result, "default", nil
}
func (*implementation) Metadata(context.Context) (loom.Nullable[service.Metadata], error) {
	return loom.Nullable[service.Metadata]{}, nil
}
func (*implementation) RequiredMetadata(context.Context) (loom.Nullable[service.Metadata], error) {
	return loom.Nullable[service.Metadata]{}, nil
}
func (*implementation) Optional(context.Context) (*service.OptionalParent, error) {
	return nil, nil
}
func (*implementation) NullableOptional(context.Context) (loom.Nullable[service.OptionalParent], error) {
	return loom.Nullable[service.OptionalParent]{}, nil
}

func TestViewedSourceAndTagFallback(t *testing.T) {
	for _, mode := range []string{"whole", "selected", "dynamic_selected", "tagged"} {
		for _, state := range []string{"value", "null", "absent"} {
			t.Run(mode+"/"+state, func(t *testing.T) {
				payload := loom.NullableValue(service.Viewed{Value: "value", Header: new("info"), Kind: new("accepted")})
				if state == "null" {
					payload = loom.NullValue[service.Viewed]()
				} else if state == "absent" {
					payload = loom.Nullable[service.Viewed]{}
				}
				mux := loomhttp.NewMuxer()
				server.Mount(mux, server.New(service.NewEndpoints(&implementation{result: payload}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				response := httptest.NewRecorder()
				require.NotPanics(t, func() {
					mux.ServeHTTP(response, httptest.NewRequest("GET", "/"+mode, nil))
				})
				if state == "absent" || state == "null" && (mode == "selected" || mode == "dynamic_selected") {
					require.Equal(t, 500, response.Code)
					require.Empty(t, response.Header().Get("X-Info"))
					require.Empty(t, response.Header().Get("loom-view"))
					return
				}
				status := 200
				if mode == "tagged" && state == "value" {
					status = 202
				}
				require.Equal(t, status, response.Code)
				if state == "null" {
					require.Equal(t, "null", strings.TrimSpace(response.Body.String()))
				} else if mode != "whole" {
					require.Equal(t, "\"value\"", strings.TrimSpace(response.Body.String()))
				}
				decode := client.DecodeWholeResponse(loomhttp.ResponseDecoder, false)
				if mode == "selected" {
					decode = client.DecodeSelectedResponse(loomhttp.ResponseDecoder, false)
				}
				if mode == "dynamic_selected" {
					decode = client.DecodeDynamicSelectedResponse(loomhttp.ResponseDecoder, false)
				}
				if mode == "tagged" {
					decode = client.DecodeTaggedResponse(loomhttp.ResponseDecoder, false)
				}
				got, err := decode(response.Result())
				require.NoError(t, err)
				round := got.(loom.Nullable[service.Viewed])
				require.Equal(t, state == "null", round.IsNull())
			})
		}
	}
}
func TestViewedEncoderRejectsBeforeMetadata(t *testing.T) {
	for _, payload := range []loom.Nullable[views.ViewedView]{loom.NullValue[views.ViewedView](), {}} {
		w := freshWriter()
		err := server.EncodeDynamicSelectedResponse(loomhttp.ResponseEncoder)(context.Background(), w, &views.ViewedNullable{Projected: payload, View: "other"})
		require.Error(t, err)
		require.Empty(t, w.Header())
		require.Zero(t, w.commits)
		require.Empty(t, w.Body.String())
	}
}
func TestTaggedNullContradiction(t *testing.T) {
	response := &http.Response{StatusCode: 202, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("null"))}
	_, err := client.DecodeTaggedResponse(loomhttp.ResponseDecoder, false)(response)
	require.Error(t, err)
}
`
