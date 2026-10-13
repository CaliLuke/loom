package middleware_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
	httpm "github.com/CaliLuke/loom/http/middleware"
	"github.com/CaliLuke/loom/observability/diagnostics"
)

type metadataMutatingWriter struct {
	*httptest.ResponseRecorder
	before bool
}

type metadataReadAheadWriter struct {
	*httptest.ResponseRecorder
}

func TestDebugRedactsCredentialsWithoutChangingWire(t *testing.T) {
	for _, location := range []string{"query", "authorization", "cookie", "JSON", "form", "response JSON", "response cookie"} {
		t.Run(location, func(t *testing.T) {
			const secret = "diagnostic-secret-value"
			target := "/resource?ordinary=visible"
			body, responseBody := `{"ordinary":"visible"}`, `{"ordinary":"visible"}`
			contentType := "application/json"
			switch location {
			case "query":
				target += "&accessToken=" + secret
			case "JSON":
				body = `{"ordinary":"visible","password":"` + secret + `"}`
			case "form":
				body = "ordinary=visible&apiKey=" + secret
				contentType = "application/x-www-form-urlencoded"
			case "response JSON":
				responseBody = `{"ordinary":"visible","password":"` + secret + `"}`
			}
			req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
			req.Header.Set("Content-Type", contentType)
			if location == "authorization" {
				req.Header.Set("Authorization", "Bearer "+secret)
			}
			if location == "cookie" {
				req.Header.Set("Cookie", "opaque="+secret)
			}
			originalHeaders := req.Header.Clone()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, target, r.URL.String())
				require.Equal(t, originalHeaders, r.Header)
				got, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.Equal(t, body, string(got))
				w.Header().Set("Content-Type", "application/json")
				if location == "response cookie" {
					w.Header().Set("Set-Cookie", "opaque="+secret)
				}
				_, err = io.WriteString(w, responseBody)
				require.NoError(t, err)
			})
			var output bytes.Buffer
			response := httptest.NewRecorder()
			httpm.Debug(loomhttp.NewMuxer(), &output)(handler).ServeHTTP(response, req)
			require.Equal(t, responseBody, response.Body.String())
			if location == "response cookie" {
				require.Equal(t, "opaque="+secret, response.Header().Get("Set-Cookie"))
			}
			require.Contains(t, output.String(), "visible")
			require.NotContains(t, output.String(), secret)
		})
	}
}

func TestDebugExplicitCredentialPolicy(t *testing.T) {
	const secret = "diagnostic-secret-value"
	fields := diagnostics.Fields{Path: true, JSON: []string{"permit"}}
	mux := loomhttp.NewMuxer()
	var output bytes.Buffer
	debug := httpm.Debug(mux, &output, fields)
	fields.Path = false
	fields.JSON[0] = "ordinary"
	mux.Handle(http.MethodPost, "/resource/{ticket}", debug(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, secret, mux.Vars(r)["ticket"])
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), secret)
		// Later mutation must not select a different parser for the captured request.
		r.Header.Set("Content-Type", "text/plain")
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(body)
		require.NoError(t, err)
	})).ServeHTTP)
	body := `{"nested":[{"permit":"` + secret + `","ordinary":"visible"}]}`
	req := httptest.NewRequest(http.MethodPost, "/resource/"+secret, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	require.Equal(t, body, response.Body.String())
	require.NotContains(t, output.String(), secret)
	require.Contains(t, output.String(), "visible")
}

// TestDebugResponseParserUsesCommittedMetadata keeps later header mutations out of redaction.
func TestDebugResponseParserUsesCommittedMetadata(t *testing.T) {
	const body = `{"password":"diagnostic-secret-value","ordinary":"visible"}`
	for _, commit := range []string{"write", "header", "flush"} {
		t.Run(commit, func(t *testing.T) {
			var output bytes.Buffer
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch commit {
				case "header":
					w.WriteHeader(http.StatusOK)
				case "flush":
					w.(http.Flusher).Flush()
				}
				_, err := io.WriteString(w, body)
				require.NoError(t, err)
				w.Header().Set("Content-Type", "text/plain")
			})
			response := httptest.NewRecorder()
			httpm.Debug(loomhttp.NewMuxer(), &output)(handler).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
			require.Equal(t, "application/json", response.Result().Header.Get("Content-Type"))
			require.Equal(t, body, response.Body.String())
			require.Contains(t, output.String(), "visible")
			require.NotContains(t, output.String(), "diagnostic-secret-value")
		})
	}
}

// TestDebugOpaqueReadFromOmitsAmbiguousMetadata does not mistake source reads for a wire commit.
func TestDebugOpaqueReadFromOmitsAmbiguousMetadata(t *testing.T) {
	const body = `{"password":"diagnostic-secret-value","ordinary":"visible"}`
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprint(committed), func(t *testing.T) {
			var output bytes.Buffer
			original := &metadataReadAheadWriter{ResponseRecorder: httptest.NewRecorder()}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if committed {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
				}
				n, err := w.(io.ReaderFrom).ReadFrom(strings.NewReader(body))
				require.NoError(t, err)
				require.EqualValues(t, len(body), n)
				w.Header().Set("Content-Type", "text/plain")
			})
			httpm.Debug(loomhttp.NewMuxer(), &output)(handler).ServeHTTP(original, httptest.NewRequest(http.MethodGet, "/?ordinary=visible", nil))
			require.Equal(t, body, original.Body.String())
			require.Equal(t, "application/json", original.Result().Header.Get("Content-Type"))
			require.NotContains(t, output.String(), "diagnostic-secret-value")
			require.Contains(t, output.String(), "visible")
			if committed {
				require.Contains(t, output.String(), "[REDACTED]")
			} else {
				require.Contains(t, output.String(), "[body omitted: response metadata unavailable]")
			}
		})
	}
}

// ReadFrom buffers before its header commit, so first source Read cannot identify wire metadata.
func (w *metadataReadAheadWriter) ReadFrom(r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	w.Header().Set("Content-Type", "application/json")
	n, err := w.Write(data)
	return int64(n), err
}

// TestDebugForwardedMetadataMutation omits a preview when forwarding changes its parser.
func TestDebugForwardedMetadataMutation(t *testing.T) {
	const body = `{"password":"diagnostic-secret-value","ordinary":"visible"}`
	for _, before := range []bool{false, true} {
		t.Run(fmt.Sprint(before), func(t *testing.T) {
			var output bytes.Buffer
			original := &metadataMutatingWriter{ResponseRecorder: httptest.NewRecorder(), before: before}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if before {
					w.Header().Set("Content-Type", "text/plain")
				}
				_, err := w.Write([]byte(body))
				require.NoError(t, err)
			})
			httpm.Debug(loomhttp.NewMuxer(), &output)(handler).ServeHTTP(original, httptest.NewRequest(http.MethodGet, "/", nil))
			require.Equal(t, body, original.Body.String())
			require.Equal(t, "application/json", original.Result().Header.Get("Content-Type"))
			require.NotContains(t, output.String(), "diagnostic-secret-value")
			require.Contains(t, output.String(), "[body omitted: response metadata unavailable]")
		})
	}
}

func (w *metadataMutatingWriter) Write(b []byte) (int, error) {
	if w.before {
		w.Header().Set("Content-Type", "application/json")
	}
	n, err := w.ResponseRecorder.Write(b)
	if !w.before {
		w.Header().Set("Content-Type", "text/plain")
	}
	return n, err
}
