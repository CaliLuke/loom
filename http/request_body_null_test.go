package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

// TestNonNullableJSONRequestBody checks root null rejection independently of
// empty, malformed, truncated, and valid JSON decoding.
func TestNonNullableJSONRequestBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want error
	}{
		{"empty", "", io.EOF},
		{"whitespace", " \t\r\n", io.EOF},
		{"null", "null", errNullRequestBody},
		{"padded null", " \nnull\t", errNullRequestBody},
		{"valid", `{"value":"ok"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			var got struct {
				Value string `json:"value"`
			}
			err := RequestDecoder(WithNonNullableBody(r)).Decode(&got)
			require.ErrorIs(t, err, tc.want)
			if tc.want == nil {
				require.Equal(t, "ok", got.Value)
			} else {
				require.Empty(t, got.Value)
			}
		})
	}
	for _, body := range []string{`{x}`, `{"value":`, "null true", "\u00a0null"} {
		t.Run(body, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(body))
			var value any
			err := RequestDecoder(WithNonNullableBody(r)).Decode(&value)
			require.Error(t, err)
			require.NotErrorIs(t, err, io.EOF)
			require.NotErrorIs(t, err, errNullRequestBody)
		})
	}
}

// TestNullableJSONRequestBody checks that unmarked requests preserve explicit
// null in nullable and unconstrained values.
func TestNullableJSONRequestBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("null"))
	var got loom.Nullable[string]
	require.NoError(t, RequestDecoder(r).Decode(&got))
	require.True(t, got.IsNull())
	r = httptest.NewRequest("POST", "/", strings.NewReader("null"))
	var raw loom.JSONValue
	require.NoError(t, RequestDecoder(r).Decode(&raw))
	require.Equal(t, "null", string(raw))
}

// TestNonNullableBodyPreservesOtherCodecs checks that the JSON null policy does
// not reinterpret text or XML values.
func TestNonNullableBodyPreservesOtherCodecs(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"text/plain", "null"},
		{"application/xml", "<string>null</string>"},
	} {
		t.Run(tc.contentType, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.contentType)
			var got string
			require.NoError(t, RequestDecoder(WithNonNullableBody(r)).Decode(&got))
			require.Equal(t, "null", got)
		})
	}
}

// TestResponseDecoderNonNullablePolicy checks that the ordinary response
// decoder applies the shared body null policy carried by the request context
// independently of empty, malformed, truncated, and valid JSON decoding.
func TestResponseDecoderNonNullablePolicy(t *testing.T) {
	for _, contentType := range []string{"application/json", ""} {
		for _, tc := range []struct {
			name string
			body string
			want error
		}{
			{"empty", "", io.EOF},
			{"whitespace", " \t\r\n", io.EOF},
			{"null", "null", errNullResponseBody},
			{"padded null", " \nnull\t", errNullResponseBody},
			{"valid", `"ok"`, nil},
		} {
			t.Run(contentType+"/"+tc.name, func(t *testing.T) {
				request := httptest.NewRequest("GET", "/", nil)
				request = request.WithContext(context.WithValue(request.Context(), nonNullableBodyKey{}, true))
				response := jsonResponse(request, contentType, tc.body)
				defer func() {
					require.NoError(t, response.Body.Close())
				}()
				var got string
				err := ResponseDecoder(response).Decode(&got)
				require.ErrorIs(t, err, tc.want)
				if tc.want == nil {
					require.Equal(t, "ok", got)
				} else {
					require.Empty(t, got)
				}
			})
		}
	}
	for _, body := range []string{`{x}`, `"ok`, "null true", "\u00a0null"} {
		t.Run(body, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			request = request.WithContext(context.WithValue(request.Context(), nonNullableBodyKey{}, true))
			response := jsonResponse(request, "application/json", body)
			defer func() {
				require.NoError(t, response.Body.Close())
			}()
			var value any
			err := ResponseDecoder(response).Decode(&value)
			require.Error(t, err)
			require.NotErrorIs(t, err, io.EOF)
			require.NotErrorIs(t, err, errNullResponseBody)
		})
	}
}

// TestNonNullableJSONResponseBody checks that the response adapter installs the
// policy on a copy and leaves the caller's response and request unchanged.
func TestNonNullableJSONResponseBody(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	response := jsonResponse(request, "application/json", "null")
	adapted := WithNonNullableResponseBody(response)
	defer func() {
		require.NoError(t, adapted.Body.Close())
	}()
	require.NotSame(t, response, adapted)
	require.NotSame(t, request, adapted.Request)
	require.Same(t, request, response.Request)
	require.Nil(t, request.Context().Value(nonNullableBodyKey{}))
	require.Equal(t, response.Header, adapted.Header)
	var got string
	require.ErrorIs(t, ResponseDecoder(adapted).Decode(&got), errNullResponseBody)

	orphan := jsonResponse(nil, "application/json", " null ")
	orphanAdapted := WithNonNullableResponseBody(orphan)
	defer func() {
		require.NoError(t, orphanAdapted.Body.Close())
	}()
	require.Nil(t, orphan.Request)
	require.ErrorIs(t, ResponseDecoder(orphanAdapted).Decode(&got), errNullResponseBody)

	valid := jsonResponse(nil, "application/json", `"ok"`)
	validAdapted := WithNonNullableResponseBody(valid)
	defer func() {
		require.NoError(t, validAdapted.Body.Close())
	}()
	require.NoError(t, ResponseDecoder(validAdapted).Decode(&got))
	require.Equal(t, "ok", got)
}

// TestNullableJSONResponseBody checks that unmarked responses preserve explicit
// null in nullable and unconstrained values.
func TestNullableJSONResponseBody(t *testing.T) {
	response := jsonResponse(nil, "application/json", "null")
	defer func() {
		require.NoError(t, response.Body.Close())
	}()
	var got loom.Nullable[string]
	require.NoError(t, ResponseDecoder(response).Decode(&got))
	require.True(t, got.IsNull())
	rawResponse := jsonResponse(nil, "application/json", "null")
	defer func() {
		require.NoError(t, rawResponse.Body.Close())
	}()
	var raw loom.JSONValue
	require.NoError(t, ResponseDecoder(rawResponse).Decode(&raw))
	require.Equal(t, "null", string(raw))
}

// TestNonNullableResponseBodyPreservesOtherCodecs checks that the JSON null
// policy does not reinterpret text or XML response values.
func TestNonNullableResponseBodyPreservesOtherCodecs(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"text/plain", "null"},
		{"application/xml", "<string>null</string>"},
	} {
		t.Run(tc.contentType, func(t *testing.T) {
			response := jsonResponse(nil, tc.contentType, tc.body)
			var got string
			adapted := WithNonNullableResponseBody(response)
			defer func() {
				require.NoError(t, adapted.Body.Close())
			}()
			require.NoError(t, ResponseDecoder(adapted).Decode(&got))
			require.Equal(t, "null", got)
		})
	}
}

// TestJSONNullDecodeDirection preserves request diagnostics and gives ordinary
// and media response decoders the same response-specific cause.
func TestJSONNullDecodeDirection(t *testing.T) {
	for _, media := range []bool{false, true} {
		for _, response := range []bool{false, true} {
			t.Run(fmt.Sprintf("media=%v/response=%v", media, response), func(t *testing.T) {
				r := httptest.NewRequest("POST", "/", strings.NewReader("null"))
				r.Header.Set("Content-Type", "application/json")
				var value string
				var err error
				if response {
					resp := WithNonNullableResponseBody(&http.Response{Header: r.Header, Body: r.Body, Request: r})
					defer func() {
						require.NoError(t, resp.Body.Close())
					}()
					if media {
						err = DecodeMediaResponse(resp, &value, false, nil, ResponseDecoder)
					} else {
						err = ResponseDecoder(resp).Decode(&value)
					}
					require.EqualError(t, err, "JSON null is not allowed for this response body")
					require.ErrorIs(t, err, errNullResponseBody)
				} else {
					defer func() {
						require.NoError(t, r.Body.Close())
					}()
					if media {
						err = DecodeMediaRequest(r, &value, false, nil, RequestDecoder)
					} else {
						err = RequestDecoder(WithNonNullableBody(r)).Decode(&value)
					}
					require.EqualError(t, err, "JSON null is not allowed for this request body")
					require.ErrorIs(t, err, errNullRequestBody)
				}
			})
		}
	}
}

func jsonResponse(request *http.Request, contentType, body string) *http.Response {
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: request}
}
