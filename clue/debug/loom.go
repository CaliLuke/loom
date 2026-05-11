package debug

import (
	"net/http"

	loomhttp "github.com/CaliLuke/loom/http"
)

// muxAdapter is a debug.Muxer adapter for loomhttp.Muxer.
type muxAdapter struct {
	muxer loomhttp.Muxer
}

// HTTP methods supported by the adapter.
var httpMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

// Adapt returns a debug.Muxer adapter for the given loomhttp.Muxer.
func Adapt(m loomhttp.Muxer) Muxer {
	return muxAdapter{muxer: m}
}

func (m muxAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.muxer.ServeHTTP(w, r)
}

func (m muxAdapter) Handle(path string, handler http.Handler) {
	for _, method := range httpMethods {
		m.muxer.Handle(method, path, handler.ServeHTTP)
	}
}

func (m muxAdapter) HandleFunc(path string, handler func(http.ResponseWriter, *http.Request)) {
	for _, method := range httpMethods {
		m.muxer.Handle(method, path, handler)
	}
}
