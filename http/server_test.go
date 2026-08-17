package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type mountedHandler struct {
	t *testing.T
}

func (h mountedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	require.Equal(h.t, "GET /items", r.Pattern)
	w.WriteHeader(http.StatusNoContent)
}

func TestMountHandlerDispatchesHTTPHandler(t *testing.T) {
	mux := NewMuxer()
	MountHandler(mux, http.MethodGet, "/items", mountedHandler{t: t})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/items", nil)
	mux.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
}
