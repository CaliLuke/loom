package http

import (
	"encoding/gob"
	"encoding/json/v2"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMuxNotFoundProblem(t *testing.T) {
	cases := []struct {
		name   string
		accept string
		wantCT string
	}{
		{name: "default", wantCT: "application/json"},
		{name: "json", accept: "application/json", wantCT: "application/json"},
		{name: "xml", accept: "application/xml", wantCT: "application/xml"},
		{name: "gob", accept: "application/gob", wantCT: "application/gob"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mux := NewMuxer()
			called := false
			mux.Handle(http.MethodGet, "/known", func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			r := httptest.NewRequest(http.MethodGet, "/missing", nil)
			if c.accept != "" {
				r.Header.Set("Accept", c.accept)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			require.False(t, called)
			require.Equal(t, http.StatusNotFound, w.Code)
			require.Equal(t, c.wantCT, w.Header().Get("Content-Type"))
			var problem ProblemResponse
			switch c.accept {
			case "application/xml":
				require.NoError(t, xml.NewDecoder(w.Body).Decode(&problem))
			case "application/gob":
				require.NoError(t, gob.NewDecoder(w.Body).Decode(&problem))
			default:
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &problem))
			}
			require.Equal(t, w.Code, problem.Status)
			require.Equal(t, "about:blank", problem.Type)
			require.Equal(t, "Not Found", problem.Title)
			require.Equal(t, "not_found", problem.Code)
			require.Equal(t, "404 page not found", problem.Detail)
			require.Regexp(t, `^urn:loom:error:.+$`, problem.Instance)
			require.Nil(t, problem.RetryHint)

			// The fallback must not interfere with a registered route.
			w = httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/known", nil))
			require.True(t, called)
			require.Equal(t, http.StatusNoContent, w.Code)
			require.Empty(t, w.Body.String())
		})
	}
}

func TestMuxNotFoundWriteFailure(t *testing.T) {
	mux := NewMuxer()
	mux.Handle(http.MethodGet, "/known", func(http.ResponseWriter, *http.Request) {
		t.Error("unexpected handler invocation")
	})
	for _, accept := range []string{"application/json", "text/plain", "text/html"} {
		t.Run(accept, func(t *testing.T) {
			w := &failingWriteWriter{header: make(http.Header), err: io.ErrClosedPipe}
			r := httptest.NewRequest(http.MethodGet, "/missing", nil)
			r.Header.Set("Accept", accept)
			require.PanicsWithValue(t, http.ErrAbortHandler, func() {
				mux.ServeHTTP(w, r)
			})
		})
	}
}

func TestMuxNotFoundText(t *testing.T) {
	mux := NewMuxer()
	mux.Handle(http.MethodGet, "/known", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	for _, accept := range []string{"text/plain", "text/html"} {
		t.Run(accept, func(t *testing.T) {
			r, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/missing", nil)
			require.NoError(t, err)
			r.Header.Set("Accept", accept)
			resp, err := server.Client().Do(r)
			require.NoError(t, err)
			body, readErr := io.ReadAll(resp.Body)
			closeErr := resp.Body.Close()
			require.NoError(t, readErr)
			require.NoError(t, closeErr)
			require.Equal(t, http.StatusNotFound, resp.StatusCode)
			require.Equal(t, accept, resp.Header.Get("Content-Type"))
			require.Equal(t, "404 page not found", string(body))
		})
	}
}
