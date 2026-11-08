package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	exchangeTestWriter struct {
		header   http.Header
		codes    []int
		headers  []http.Header
		onHeader func()
	}
	gatedInterimWriter struct {
		http.ResponseWriter
		entered chan struct{}
		release chan struct{}
	}
	exchangeWireResult struct {
		first  error
		late   error
		waited bool
	}
)

func TestExchangeWriterSendBeforeFinal(t *testing.T) {
	w := &exchangeTestWriter{header: make(http.Header)}
	l := NewHandlerLifecycle(w, httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
	require.Nil(t, ExchangeFromContext(context.Background()))
	require.Same(t, l.Exchange(), ExchangeFromContext(l.Context()))
	require.NoError(t, l.Exchange().Send(http.StatusEarlyHints, http.Header{"Link": {"</asset>; rel=preload"}}))
	require.Equal(t, []int{http.StatusEarlyHints}, w.codes)
	require.False(t, l.responseCommitted())
	l.Exchange().BeginFinal(FinalResult)
	require.False(t, l.responseCommitted(), "admission must not fabricate physical commitment")
	l.Writer().WriteHeader(http.StatusCreated)
	require.True(t, l.responseCommitted())
	require.Equal(t, []int{http.StatusEarlyHints, http.StatusCreated}, w.codes)
}

func TestExchangeWriterRefusesAfterBeginFinal(t *testing.T) {
	for _, kind := range []FinalKind{FinalResult, FinalError, FinalSSEOpen, FinalUpgrade, FinalFile, FinalRaw, FinalRedirect} {
		t.Run(finalKindName(kind), func(t *testing.T) {
			w := &exchangeTestWriter{header: make(http.Header)}
			l := NewHandlerLifecycle(w, httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
			l.Exchange().BeginFinal(kind)
			l.Exchange().BeginFinal(FinalError)
			require.ErrorIs(t, l.Exchange().Send(103, http.Header{"Link": {"late"}}), ErrInformationalClosed)
			require.Empty(t, w.codes)
			require.Empty(t, w.header)
		})
	}
}

func TestExchangeWriterRestoresHeaderMap(t *testing.T) {
	w := &exchangeTestWriter{header: http.Header{"X-Final": {"one", "two"}}}
	l := NewHandlerLifecycle(w, httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
	before := w.header.Clone()
	headers := http.Header{"Link": {"a", "b"}}
	require.NoError(t, l.Exchange().Send(103, headers))
	require.Equal(t, []http.Header{headers}, w.headers)
	require.Equal(t, before, w.header)
	headers["Link"][0] = "changed"
	require.Equal(t, "a", w.headers[0].Get("Link"))
	require.NoError(t, l.Exchange().Send(102, nil))
	require.Empty(t, w.headers[1])
	require.Equal(t, before, w.header)
	require.NoError(t, l.Exchange().Send(103, w.header))
	require.Equal(t, before, w.headers[2], "the supplied header map may alias the writer map")
	require.Equal(t, before, w.header)
}

func TestExchangeWriterRestoresNoncanonicalHeaderMap(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		name := "lowercase"
		if mixed {
			name = "mixed-case"
		}
		t.Run(name, func(t *testing.T) {
			original := http.Header{"x-final": {"keep"}}
			interim := http.Header{"link": {"hint"}}
			if mixed {
				original["X-Final"] = []string{"also-keep"}
				interim["Link"] = []string{"also-hint"}
			}
			w := &exchangeTestWriter{header: original.Clone()}
			l := NewHandlerLifecycle(w, httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
			require.NoError(t, l.Exchange().Send(103, interim))
			require.Equal(t, interim, w.headers[0], "interim must not inherit final header entries")
			require.Equal(t, original, w.header, "restore must remove every interim key")
		})
	}
}

func TestExchangeWriterHTTP10Refused(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Proto, r.ProtoMajor, r.ProtoMinor = "HTTP/1.0", 1, 0
	w := &exchangeTestWriter{header: make(http.Header)}
	l := NewHandlerLifecycle(w, r, "svc", "method")
	require.ErrorIs(t, l.Exchange().Send(103, nil), ErrInformationalUnsupported)
	require.Empty(t, w.codes)
	l.Exchange().BeginFinal(FinalResult)
	require.ErrorIs(t, l.Exchange().Send(103, nil), ErrInformationalClosed)
}

func TestExchangeWriterLimit(t *testing.T) {
	w := &exchangeTestWriter{header: make(http.Header)}
	l := NewHandlerLifecycle(w, httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
	for range MaxInformationalResponses {
		require.NoError(t, l.Exchange().Send(103, nil))
	}
	require.ErrorIs(t, l.Exchange().Send(103, nil), ErrInformationalLimit)
	require.Len(t, w.codes, MaxInformationalResponses)
}

func TestExchangeWriterRefusesAuthored100And101(t *testing.T) {
	for _, code := range []int{99, 100, 101, 200, 600} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			w := &exchangeTestWriter{header: make(http.Header)}
			l := NewHandlerLifecycle(w, httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
			require.Error(t, l.Exchange().Send(code, nil))
			require.Empty(t, w.codes)
			for range MaxInformationalResponses {
				require.NoError(t, l.Exchange().Send(199, nil), "rejected status must not spend budget")
			}
		})
	}
}

func TestExchangeWriterConcurrentSendDuringFinalStart(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		for _, kind := range []FinalKind{FinalResult, FinalError, FinalSSEOpen, FinalUpgrade, FinalFile, FinalRaw, FinalRedirect} {
			name := "http1/" + finalKindName(kind)
			if h2 {
				name = "http2/" + finalKindName(kind)
			}
			t.Run(name, func(t *testing.T) {
				outcomes := make(chan exchangeWireResult, 1)
				server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gate := &gatedInterimWriter{ResponseWriter: w, entered: make(chan struct{}), release: make(chan struct{})}
					l := NewHandlerLifecycle(gate, r, "svc", "method")
					defer l.End()
					l.Writer().Header().Set("X-Original", "keep")
					sent := make(chan error, 1)
					go func() {
						sent <- l.Exchange().Send(103, http.Header{"Link": {"</asset>"}})
					}()
					select {
					case <-gate.entered:
					case err := <-sent:
						outcomes <- exchangeWireResult{first: err}
						return
					}
					attempted, finalDone := make(chan struct{}), make(chan struct{})
					go func() {
						close(attempted)
						l.Exchange().BeginFinal(kind)
						l.Writer().Header().Set("X-Final", "ready")
						l.Writer().WriteHeader(http.StatusCreated)
						close(finalDone)
					}()
					<-attempted
					waited := true
					select {
					case <-finalDone:
						waited = false
					default:
					}
					close(gate.release)
					first := <-sent
					<-finalDone
					outcomes <- exchangeWireResult{first: first, late: l.Exchange().Send(103, nil), waited: waited}
				}))
				server.EnableHTTP2 = h2
				if h2 {
					server.StartTLS()
				} else {
					server.Start()
				}
				defer server.Close()
				var hints []http.Header
				request, err := http.NewRequest(http.MethodGet, server.URL, nil)
				require.NoError(t, err)
				request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{
					Got1xxResponse: func(code int, h textproto.MIMEHeader) error {
						hints = append(hints, http.Header(h).Clone())
						return nil
					},
				}))
				response, err := server.Client().Do(request)
				require.NoError(t, err)
				_, err = io.Copy(io.Discard, response.Body)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				result := <-outcomes
				require.NoError(t, result.first)
				require.True(t, result.waited)
				require.ErrorIs(t, result.late, ErrInformationalClosed)
				require.Equal(t, http.StatusCreated, response.StatusCode)
				require.Equal(t, "keep", response.Header.Get("X-Original"))
				require.Equal(t, "ready", response.Header.Get("X-Final"))
				require.Empty(t, response.Header.Get("Link"))
				require.Equal(t, []http.Header{{"Link": {"</asset>"}}}, hints)
				if h2 {
					require.Equal(t, 2, response.ProtoMajor)
				} else {
					require.Equal(t, 1, response.ProtoMajor)
				}
			})
		}
	}
}

func TestExchangeWriterDerivedHeadWire(t *testing.T) {
	for _, h2 := range []bool{false, true} {
		name := "http1"
		if h2 {
			name = "http2"
		}
		t.Run(name, func(t *testing.T) {
			type outcome struct {
				sendErr             error
				writeErr            error
				lateErr             error
				committedAfterHint  bool
				underlyingAfterHint http.Header
			}
			outcomes := make(chan outcome, 1)
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Outer", "keep")
				DerivedHeadHandler(http.HandlerFunc(func(head http.ResponseWriter, r *http.Request) {
					l := NewHandlerLifecycle(head, r, "svc", "method")
					defer l.End()
					l.Writer().Header().Set("X-Final", "ready")
					result := outcome{sendErr: l.Exchange().Send(103, http.Header{"Link": {"</asset>"}})}
					result.committedAfterHint = l.responseCommitted()
					result.underlyingAfterHint = w.Header().Clone()
					l.EncodeResponse(func(_ context.Context, final http.ResponseWriter) error {
						final.Header().Set("Content-Type", "text/plain")
						final.WriteHeader(http.StatusCreated)
						final.Header().Set("X-Late", "must-not-leak")
						_, result.writeErr = io.WriteString(final, "payload")
						return result.writeErr
					}, nil, nil)
					result.lateErr = l.Exchange().Send(103, nil)
					outcomes <- result
				})).ServeHTTP(w, r)
			}))
			server.EnableHTTP2 = h2
			if h2 {
				server.StartTLS()
			} else {
				server.Start()
			}
			defer server.Close()
			var codes []int
			var hints []http.Header
			request, err := http.NewRequest(http.MethodHead, server.URL, nil)
			require.NoError(t, err)
			request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{
				Got1xxResponse: func(code int, h textproto.MIMEHeader) error {
					codes = append(codes, code)
					hints = append(hints, http.Header(h).Clone())
					return nil
				},
			}))
			response, err := server.Client().Do(request)
			require.NoError(t, err)
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			result := <-outcomes
			require.NoError(t, result.sendErr)
			require.NoError(t, result.writeErr)
			require.False(t, result.committedAfterHint)
			require.Equal(t, http.Header{"X-Outer": {"keep"}}, result.underlyingAfterHint)
			require.Equal(t, []int{103}, codes)
			require.Equal(t, []http.Header{{"Link": {"</asset>"}}}, hints)
			require.ErrorIs(t, result.lateErr, ErrInformationalClosed)
			require.Equal(t, http.StatusCreated, response.StatusCode)
			require.Equal(t, "7", response.Header.Get("Content-Length"))
			require.Equal(t, int64(7), response.ContentLength)
			require.Equal(t, "keep", response.Header.Get("X-Outer"))
			require.Equal(t, "ready", response.Header.Get("X-Final"))
			require.Empty(t, response.Header.Get("Link"))
			require.Empty(t, response.Header.Get("X-Late"))
			require.Empty(t, body)
			if h2 {
				require.Equal(t, 2, response.ProtoMajor)
			} else {
				require.Equal(t, 1, response.ProtoMajor)
			}
		})
	}
}

func TestExchangeWriterDerivedHeadForwardingBoundary(t *testing.T) {
	underlying := &exchangeTestWriter{header: http.Header{"X-Outer": {"keep"}}}
	DerivedHeadHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		clear(w.Header())
		w.Header()["x-interim"] = []string{"temporary"}
		for _, code := range []int{100, 101, 103, 199} {
			w.WriteHeader(code)
			require.Equal(t, http.Header{"X-Outer": {"keep"}}, underlying.header)
		}
		clear(w.Header())
		w.Header().Set("X-Final", "ready")
		w.WriteHeader(http.StatusCreated)
		require.NotContains(t, underlying.codes, http.StatusCreated, "final status remains buffered")
		w.WriteHeader(103)
		_, err := io.WriteString(w, "payload")
		require.NoError(t, err)
	})).ServeHTTP(underlying, httptest.NewRequest(http.MethodHead, "/", nil))
	require.Equal(t, []int{100, 103, 199, http.StatusCreated}, underlying.codes)
	for _, headers := range underlying.headers[:3] {
		require.Equal(t, http.Header{"x-interim": {"temporary"}}, headers)
	}
	require.Equal(t, http.Header{"X-Final": {"ready"}, "Content-Length": {"7"}}, underlying.headers[3])
}

func finalKindName(kind FinalKind) string {
	return [...]string{"result", "error", "sse-open", "upgrade", "file", "raw", "redirect"}[kind]
}

func (w *exchangeTestWriter) Header() http.Header {
	if w.onHeader != nil {
		w.onHeader()
	}
	return w.header
}

func (w *exchangeTestWriter) WriteHeader(code int) {
	w.codes = append(w.codes, code)
	w.headers = append(w.headers, w.header.Clone())
}

func (w *exchangeTestWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (w *gatedInterimWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	if code == http.StatusEarlyHints {
		close(w.entered)
		<-w.release
	}
}
