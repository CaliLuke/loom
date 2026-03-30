package xray

import (
	"net/http"

	loomhttp "github.com/CaliLuke/loom/http"
)

// xrayDoer is a loomhttp.Doer middleware that will create xray subsegments for
// traced requests.
type xrayDoer struct {
	wrapped loomhttp.Doer
}

// WrapDoer wraps a Loom HTTP Doer and creates xray subsegments for traced
// requests.
func WrapDoer(doer loomhttp.Doer) loomhttp.Doer {
	return &xrayDoer{doer}
}

// Do calls through to the wrapped Doer, creating subsegments as appropriate.
func (r *xrayDoer) Do(req *http.Request) (*http.Response, error) {
	return withTracedRequest(req, func(tracedReq *http.Request) (*http.Response, error) {
		return r.wrapped.Do(tracedReq)
	}, func() (*http.Response, error) {
		return r.wrapped.Do(req)
	})
}
