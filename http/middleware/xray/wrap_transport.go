package xray

import (
	"context"
	"net/http"

	"github.com/CaliLuke/loom/middleware"
	"github.com/CaliLuke/loom/middleware/xray"
)

// xrayTransport wraps an http RoundTripper to add a tracing subsegment of the
// request's context segment.
type xrayTransport struct {
	wrapped http.RoundTripper
}

// WrapTransport wraps a http RoundTripper with a RoundTripper which creates
// subsegments of the segment in each request's context. The subsegments
// created this way have their namespace set to "remote". The request's ctx
// must be set and contain the current request segment as set by the xray
// middleware.
//
// Example of how to wrap http.Client's transport:
//
//	httpClient := &http.Client{
//	   Transport: WrapTransport(http.DefaultTransport),
//	}
func WrapTransport(rt http.RoundTripper) http.RoundTripper {
	return &xrayTransport{rt}
}

// RoundTrip wraps the original RoundTripper.RoundTrip to create xray tracing
// segments.
func (t *xrayTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return withTracedRequest(req, func(tracedReq *http.Request) (*http.Response, error) {
		return t.wrapped.RoundTrip(tracedReq)
	}, func() (*http.Response, error) {
		return t.wrapped.RoundTrip(req)
	})
}

func withTracedRequest(req *http.Request, traced func(*http.Request) (*http.Response, error), untraced func() (*http.Response, error)) (*http.Response, error) {
	ctx := req.Context()
	seg := ctx.Value(xray.SegKey)
	if seg == nil {
		return untraced()
	}

	s := seg.(*xray.Segment)
	sub := s.NewSubsegment(req.URL.Host)
	hs := &HTTPSegment{Segment: sub}
	hs.RecordRequest(req, "remote")
	hs.SubmitInProgress()
	defer hs.Close()

	// update the context with the latest segment
	ctx = middleware.WithSpan(ctx, hs.TraceID, hs.ID, hs.ParentID)
	req = req.WithContext(context.WithValue(ctx, xray.SegKey, hs.Segment))
	resp, err := traced(req)
	if err != nil {
		hs.RecordError(err)
	} else {
		hs.RecordResponse(resp)
	}
	return resp, err
}
