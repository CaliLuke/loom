package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPTraceComposeDiscardsNewHookError(t *testing.T) {
	first := errors.New("new")
	var sequence []string
	old := &httptrace.ClientTrace{Got1xxResponse: func(int, textproto.MIMEHeader) error {
		sequence = append(sequence, "old")
		return nil
	}}
	ctx := httptrace.WithClientTrace(context.Background(), old)
	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{Got1xxResponse: func(int, textproto.MIMEHeader) error {
		sequence = append(sequence, "new")
		return first
	}})
	require.NoError(t, httptrace.ContextClientTrace(ctx).Got1xxResponse(103, nil))
	require.Equal(t, []string{"new", "old"}, sequence)
}

func TestObserveInformationalWireHooks(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		t.Run(informationalProtocol(h2), func(t *testing.T) {
			for _, mode := range []string{"no_caller", "caller_nil", "caller_error", "both_fail"} {
				t.Run(mode, func(t *testing.T) {
					loomErr, callerErr := errors.New("observer"), errors.New("caller")
					server := informationalServer(t, h2, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Link", "</asset>")
						w.WriteHeader(103)
						w.WriteHeader(200)
					}))
					req, err := http.NewRequest(http.MethodGet, server.URL, nil)
					require.NoError(t, err)
					var callerCalls, observed atomic.Int32
					callerDone := make(chan struct{})
					if mode != "no_caller" {
						req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{Got1xxResponse: func(int, textproto.MIMEHeader) error {
							callerCalls.Add(1)
							close(callerDone)
							if mode == "caller_error" || mode == "both_fail" {
								return callerErr
							}
							return nil
						}}))
					}
					//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
					req, finish := ObserveInformational(req, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(_ context.Context, code int, headers http.Header) error {
						observed.Add(1)
						if code != 103 || headers.Get("Link") != "</asset>" {
							return errors.New("wrong informational metadata")
						}
						if mode != "caller_error" {
							return loomErr
						}
						return nil
					}}})
					//nolint:bodyclose // finish closes on abort or transfers to the caller below: https://github.com/timakin/bodyclose/issues/30. Remove when the analyzer follows this ownership transfer.
					response, doErr := server.Client().Do(req)
					response, err = finish(response, doErr)
					if response != nil {
						require.NoError(t, response.Body.Close())
					}
					require.Nil(t, response)
					require.EqualValues(t, 1, observed.Load())
					if mode == "caller_error" {
						require.Same(t, doErr, err)
						// HTTP/2 wraps the caller error as StreamError.Cause without Unwrap.
						// The owner preserves the actual Do error on both protocols.
						if !h2 {
							require.ErrorIs(t, err, callerErr)
						}
					} else {
						require.ErrorIs(t, err, loomErr)
						if doErr != nil {
							require.ErrorIs(t, err, doErr)
						}
					}
					if mode != "no_caller" {
						<-callerDone
						require.EqualValues(t, 1, callerCalls.Load())
					}
				})
			}
		})
	}
}

func TestObserveInformationalWireStreamingBodySurvives(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		t.Run(informationalProtocol(h2), func(t *testing.T) {
			release := make(chan struct{})
			wrote := make(chan error, 1)
			server := informationalServer(t, h2, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(103)
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(200)
				if err := http.NewResponseController(w).Flush(); err != nil {
					wrote <- err
					return
				}
				<-release
				_, err := io.WriteString(w, "data: late\n\n")
				wrote <- err
			}))
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			var observed atomic.Int32
			//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
			req, finish := ObserveInformational(req, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(context.Context, int, http.Header) error {
				observed.Add(1)
				return nil
			}}})
			//nolint:bodyclose // finish closes on abort or transfers to the caller below: https://github.com/timakin/bodyclose/issues/30. Remove when the analyzer follows this ownership transfer.
			resp, doErr := server.Client().Do(req)
			resp, err = finish(resp, doErr)
			close(release)
			require.NoError(t, err)
			require.NoError(t, req.Context().Err(), "finish must retain cancellation for the body")
			if h2 {
				require.Equal(t, 2, resp.ProtoMajor)
			} else {
				require.Equal(t, 1, resp.ProtoMajor)
			}
			require.EqualValues(t, 1, observed.Load())
			data, readErr := io.ReadAll(resp.Body)
			require.NoError(t, readErr)
			require.Equal(t, "data: late\n\n", string(data))
			require.ErrorIs(t, req.Context().Err(), context.Canceled)
			require.NoError(t, resp.Body.Close())
			require.NoError(t, <-wrote)
		})
	}
}

func TestObserveInformationalWireRedirectCountsPerExchange(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		t.Run(informationalProtocol(h2), func(t *testing.T) {
			server := informationalServer(t, h2, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for range 8 {
					w.WriteHeader(103)
				}
				if r.URL.Path == "/" {
					w.Header().Set("Location", "/final")
					w.WriteHeader(307)
				} else {
					w.WriteHeader(204)
				}
			}))
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			var observed atomic.Int32
			//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
			req, finish := ObserveInformational(req, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(context.Context, int, http.Header) error {
				observed.Add(1)
				return nil
			}}})
			//nolint:bodyclose // finish closes on abort or transfers to the caller below: https://github.com/timakin/bodyclose/issues/30. Remove when the analyzer follows this ownership transfer.
			resp, doErr := server.Client().Do(req)
			resp, err = finish(resp, doErr)
			require.NoError(t, err)
			if h2 {
				require.Equal(t, 2, resp.ProtoMajor)
			} else {
				require.Equal(t, 1, resp.ProtoMajor)
			}
			require.Equal(t, 204, resp.StatusCode)
			require.EqualValues(t, 16, observed.Load())
			require.NoError(t, resp.Body.Close())
			require.ErrorIs(t, req.Context().Err(), context.Canceled)
		})
	}
}

func TestObserveInformationalWireExpectContinueBudget(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		t.Run(informationalProtocol(h2), func(t *testing.T) {
			read := make(chan error, 1)
			server := informationalServer(t, h2, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.Copy(io.Discard, r.Body)
				read <- err
				for range 8 {
					w.WriteHeader(103)
				}
				w.WriteHeader(204)
			}))
			req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("request"))
			require.NoError(t, err)
			req.Header.Set("Expect", "100-continue")
			var hundred, other atomic.Int32
			//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
			req, finish := ObserveInformational(req, []InformationalObserver{{Selector: StatusSelector{Class: 1}, Observe: func(_ context.Context, code int, _ http.Header) error {
				if code == 100 {
					hundred.Add(1)
				} else {
					other.Add(1)
				}
				return nil
			}}})
			//nolint:bodyclose // finish closes on abort or transfers to the caller below: https://github.com/timakin/bodyclose/issues/30. Remove when the analyzer follows this ownership transfer.
			resp, doErr := server.Client().Do(req)
			resp, err = finish(resp, doErr)
			require.NoError(t, err)
			if h2 {
				require.Equal(t, 2, resp.ProtoMajor)
			} else {
				require.Equal(t, 1, resp.ProtoMajor)
			}
			require.NoError(t, resp.Body.Close())
			require.EqualValues(t, 1, hundred.Load())
			require.EqualValues(t, 8, other.Load())
			require.NoError(t, <-read)
		})
	}
}

func informationalProtocol(h2 bool) string {
	if h2 {
		return "http2"
	}
	return "http1"
}

func informationalServer(t *testing.T, h2 bool, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	server.EnableHTTP2 = h2
	server.StartTLS()
	server.Client().Timeout = 5 * time.Second
	t.Cleanup(server.Close)
	return server
}
