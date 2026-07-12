package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	httpm "github.com/CaliLuke/loom/http/middleware"
	"github.com/CaliLuke/loom/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDoer records the request it receives and returns canned results.
type stubDoer struct {
	req  *http.Request
	resp *http.Response
	err  error
}

func (d *stubDoer) Do(r *http.Request) (*http.Response, error) {
	d.req = r
	return d.resp, d.err
}

func TestWrapDoer(t *testing.T) {
	cases := []struct {
		name             string
		traceID          string
		spanID           string
		wantTraceHeader  string
		wantParentHeader string
	}{
		{
			name:             "traced context sets headers",
			traceID:          "trace-1",
			spanID:           "span-1",
			wantTraceHeader:  "trace-1",
			wantParentHeader: "span-1",
		},
		{
			name:             "untraced context sets no headers",
			traceID:          "",
			spanID:           "",
			wantTraceHeader:  "",
			wantParentHeader: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			if c.traceID != "" {
				ctx = middleware.WithSpan(ctx, c.traceID, c.spanID, "")
			}
			req, err := http.NewRequestWithContext(ctx, "GET", "http://example.com/", nil)
			require.NoError(t, err)

			wantResp := &http.Response{StatusCode: http.StatusTeapot, Body: http.NoBody}
			doer := &stubDoer{resp: wantResp}

			resp, err := httpm.WrapDoer(doer).Do(req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			defer func() {
				require.NoError(t, resp.Body.Close())
			}()
			assert.Same(t, wantResp, resp, "response not forwarded")
			require.NotNil(t, doer.req, "wrapped doer was not invoked")
			assert.Equal(t, c.wantTraceHeader, doer.req.Header.Get(httpm.TraceIDHeader))
			assert.Equal(t, c.wantParentHeader, doer.req.Header.Get(httpm.ParentSpanIDHeader))
		})
	}
}

func TestWrapDoerForwardsError(t *testing.T) {
	wantErr := errors.New("connection refused")
	doer := &stubDoer{err: wantErr}
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	require.NoError(t, err)

	resp, err := httpm.WrapDoer(doer).Do(req) // nolint: bodyclose // stub doer returns a nil response

	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, resp)
}
