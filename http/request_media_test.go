package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestJSONMediaBodyContract(t *testing.T) {
	for _, media := range []string{"", "application/json", "Application/Vnd.example+JSON; charset=utf-8"} {
		for _, test := range []struct {
			name    string
			body    string
			want    string
			eof     bool
			invalid bool
		}{
			{name: "empty", eof: true},
			{name: "whitespace", body: " \r\n\t", eof: true},
			{name: "malformed", body: `{"value":}`, invalid: true},
			{name: "truncated", body: `{"value":"unfinished`, invalid: true},
			{name: "valid", body: `{"value":"ok"}`, want: "ok"},
		} {
			t.Run(media+"/"+test.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
				request.Header.Set("Content-Type", media)
				var got struct {
					Value string `json:"value"`
				}
				err := RequestDecoder(request).Decode(&got)
				switch {
				case test.eof:
					require.ErrorIs(t, err, io.EOF)
				case test.invalid:
					require.Error(t, err)
					require.NotErrorIs(t, err, io.EOF)
				default:
					require.NoError(t, err)
				}
				require.Equal(t, test.want, got.Value)
			})
		}
	}
}

func TestRequestJSONMediaLimit(t *testing.T) {
	for _, media := range []string{"application/json", "application/merge-patch+json"} {
		for _, extra := range []string{"", " "} {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"value":"ok"}`+extra))
			request.Header.Set("Content-Type", media)
			policy, err := NewRequestBodyPolicy(14)
			require.NoError(t, err)
			policy.Handler(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				var got map[string]string
				err := RequestDecoder(request).Decode(&got)
				if extra == "" {
					require.NoError(t, err)
					require.Equal(t, "ok", got["value"])
				} else {
					requireRequestBodyTooLargeResponse(t, err)
				}
			})).ServeHTTP(httptest.NewRecorder(), request)
		}
	}
}
