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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type informationalBodyProbe struct {
	io.Reader
	closes       atomic.Int32
	closeErr     error
	closeEntered chan struct{}
	closeRelease chan struct{}
}

func TestObserveInformationalNoObservers(t *testing.T) {
	for _, observers := range [][]InformationalObserver{nil, {{Selector: StatusSelector{Code: 103}}}} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		caller := &httptrace.ClientTrace{Got1xxResponse: func(int, textproto.MIMEHeader) error {
			return nil
		}}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), caller))
		//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
		observed, finish := ObserveInformational(req, observers)
		require.Same(t, req, observed)
		require.Same(t, caller, httptrace.ContextClientTrace(observed.Context()))
		sentinel := errors.New("do")
		body := &informationalBodyProbe{Reader: strings.NewReader("body")}
		response := &http.Response{Body: body}
		got, err := finish(response, sentinel)
		require.Same(t, response, got)
		require.Same(t, sentinel, err)
		require.Same(t, body, response.Body)
		require.Zero(t, body.closes.Load())
		require.NoError(t, got.Body.Close())
	}
}

func TestObserveInformationalSelectionAndSnapshot(t *testing.T) {
	var codes []int
	observers := []InformationalObserver{
		{Selector: StatusSelector{Code: 103}},
		{Selector: StatusSelector{Class: 1}, Observe: func(_ context.Context, code int, header http.Header) error {
			codes = append(codes, code)
			require.Equal(t, "value", header.Get("X-Test"))
			return nil
		}},
	}
	//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
	_, finish, trace := informationalTrace(t, observers, nil)
	observers[0].Selector = StatusSelector{Code: 102}
	observers[1].Observe = nil
	for _, code := range []int{100, 102, 103, 199} {
		require.NoError(t, trace.Got1xxResponse(code, textproto.MIMEHeader{"X-Test": {"value"}}))
	}
	require.Equal(t, []int{100, 102, 199}, codes, "unobserved exact key must shadow class")
	original := &http.Response{Body: http.NoBody}
	response, err := finish(original, nil)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
}

func TestObserveInformationalBudgets(t *testing.T) {
	tests := []struct {
		name  string
		codes []int
		limit bool
		calls int
	}{
		{"independent", append([]int{100}, informationalCodes(103, 8)...), false, 9},
		{"non100_overflow", informationalCodes(103, 9), true, 8},
		{"second100", []int{100, 100}, true, 1},
		{"undeclared_counted", informationalCodes(102, 9), true, 0},
		{"unobserved_counted", informationalCodes(104, 9), true, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			observe := func(context.Context, int, http.Header) error {
				calls++
				return nil
			}
			//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
			req, finish, trace := informationalTrace(t, []InformationalObserver{
				{Selector: StatusSelector{Code: 100}, Observe: observe},
				{Selector: StatusSelector{Code: 103}, Observe: observe},
				{Selector: StatusSelector{Code: 104}},
			}, nil)
			var got error
			for _, code := range tc.codes {
				got = trace.Got1xxResponse(code, nil)
			}
			require.Equal(t, tc.calls, calls)
			if tc.limit {
				require.ErrorIs(t, got, ErrInformationalLimit)
				require.ErrorIs(t, req.Context().Err(), context.Canceled)
				response, err := finish(nil, context.Canceled)
				if response != nil {
					require.NoError(t, response.Body.Close())
				}
				require.Nil(t, response)
				require.ErrorIs(t, err, ErrInformationalLimit)
			} else {
				require.NoError(t, got)
				response, err := finish(&http.Response{Body: http.NoBody}, nil)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
			}
		})
	}
}

func TestObserveInformationalLateWroteHeadersKeepsBudget(t *testing.T) {
	tests := []struct {
		name  string
		code  int
		limit int
	}{
		{"non100", http.StatusEarlyHints, 8},
		{"hundred", http.StatusContinue, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var firstBytes, headerWrites, calls int
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
				GotFirstResponseByte: func() {
					firstBytes++
				},
				WroteHeaders: func() {
					headerWrites++
				},
			}))
			//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
			req, finish := ObserveInformational(req, []InformationalObserver{{Selector: StatusSelector{Class: 1}, Observe: func(context.Context, int, http.Header) error {
				calls++
				return nil
			}}})
			trace := httptrace.ContextClientTrace(req.Context())
			trace.GotFirstResponseByte()
			for range tc.limit {
				require.NoError(t, trace.Got1xxResponse(tc.code, nil))
			}
			// HTTP/2 can deliver response headers before the writing goroutine's
			// WroteHeaders notification. It is still the same response exchange.
			trace.WroteHeaders()
			got := trace.Got1xxResponse(tc.code, nil)
			if !errors.Is(got, ErrInformationalLimit) {
				t.Errorf("late write notification allowed excess status %d: %v", tc.code, got)
			}
			if calls != tc.limit {
				t.Errorf("observer calls = %d, want %d", calls, tc.limit)
			}
			require.Equal(t, 1, firstBytes, "caller first-byte hook remains composed")
			require.Equal(t, 1, headerWrites, "caller write hook remains composed")
			response, err := finish(nil, context.Canceled)
			if response != nil {
				require.NoError(t, response.Body.Close())
			}
			require.Nil(t, response)
			require.ErrorIs(t, err, ErrInformationalLimit)
		})
	}
}

func TestObserveInformationalDeclared100Abort(t *testing.T) {
	first := errors.New("declared100 observer")
	calls := 0
	//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
	req, finish, trace := informationalTrace(t, []InformationalObserver{{Selector: StatusSelector{Code: 100}, Observe: func(context.Context, int, http.Header) error {
		calls++
		return first
	}}}, nil)
	require.Same(t, first, trace.Got1xxResponse(100, nil))
	require.Same(t, first, trace.Got1xxResponse(103, nil))
	require.Equal(t, 1, calls)
	require.ErrorIs(t, req.Context().Err(), context.Canceled)
	response, err := finish(nil, context.Canceled)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Nil(t, response)
	require.ErrorIs(t, err, first)
}

func TestObserveInformationalResetAndFirstError(t *testing.T) {
	first := errors.New("observer first")
	second := errors.New("must not invoke")
	calls := 0
	//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
	_, finish, trace := informationalTrace(t, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(context.Context, int, http.Header) error {
		calls++
		if calls <= 8 {
			return nil
		}
		if calls == 9 {
			return first
		}
		return second
	}}}, nil)
	for range 8 {
		require.NoError(t, trace.Got1xxResponse(103, nil))
	}
	trace.GotFirstResponseByte()
	require.Same(t, first, trace.Got1xxResponse(103, nil))
	trace.GotFirstResponseByte()
	require.Same(t, first, trace.Got1xxResponse(103, nil))
	require.Equal(t, 9, calls)
	response, err := finish(nil, context.Canceled)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Nil(t, response)
	require.ErrorIs(t, err, first)
	require.NotErrorIs(t, err, second)
}

func TestObserveInformationalFinishErrors(t *testing.T) {
	for _, returned := range []bool{false, true} {
		for _, doFailed := range []bool{false, true} {
			name := "without_response"
			if returned {
				name = "with_response"
			}
			if doFailed {
				name += "_do_error"
			} else {
				name += "_nil_do_error"
			}
			t.Run(name, func(t *testing.T) {
				first, transport, closing := errors.New("observer"), errors.New("transport"), errors.New("close")
				//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
				_, finish, trace := informationalTrace(t, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(context.Context, int, http.Header) error {
					return first
				}}}, nil)
				require.Same(t, first, trace.Got1xxResponse(103, nil))
				var response *http.Response
				body := &informationalBodyProbe{Reader: strings.NewReader("body"), closeErr: closing}
				if returned {
					response = &http.Response{Body: body}
				}
				var doErr error
				if doFailed {
					doErr = transport
				}
				got, err := finish(response, doErr)
				if returned {
					require.EqualValues(t, 1, body.closes.Load())
				}
				if got != nil {
					require.NoError(t, got.Body.Close())
				}
				require.Nil(t, got)
				require.ErrorIs(t, err, first)
				joined, ok := err.(interface{ Unwrap() []error })
				require.True(t, ok)
				expected := []error{first}
				if doFailed {
					expected = append(expected, transport)
				}
				if returned {
					expected = append(expected, closing)
				}
				require.Equal(t, expected, joined.Unwrap())
			})
		}
	}
}

func TestObserveInformationalCallerOnlyErrorUnchanged(t *testing.T) {
	callerErr := errors.New("caller")
	//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
	req, finish, trace := informationalTrace(t, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(context.Context, int, http.Header) error {
		return nil
	}}}, &httptrace.ClientTrace{Got1xxResponse: func(int, textproto.MIMEHeader) error {
		return callerErr
	}})
	require.Same(t, callerErr, trace.Got1xxResponse(103, nil))
	response, err := finish(nil, callerErr)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Nil(t, response)
	require.Same(t, callerErr, err)
	require.ErrorIs(t, req.Context().Err(), context.Canceled)
}

func TestObserveInformationalLateCallerAndBufferedHook(t *testing.T) {
	first := errors.New("first")
	entered, release := make(chan struct{}), make(chan struct{})
	var callerCalls, observerCalls atomic.Int32
	caller := &httptrace.ClientTrace{Got1xxResponse: func(int, textproto.MIMEHeader) error {
		if callerCalls.Add(1) == 1 {
			close(entered)
			<-release
		}
		return nil
	}}
	//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
	_, finish, trace := informationalTrace(t, []InformationalObserver{{Selector: StatusSelector{Class: 1}, Observe: func(context.Context, int, http.Header) error {
		observerCalls.Add(1)
		return first
	}}}, caller)
	done := make(chan error, 1)
	go func() {
		done <- trace.Got1xxResponse(103, nil)
	}()
	<-entered
	// The old caller is still blocked: Do is allowed to settle meanwhile.
	response, err := finish(nil, context.Canceled)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Nil(t, response)
	require.ErrorIs(t, err, first)
	captured := err.Error()
	close(release)
	require.NoError(t, <-done, "Go discards new hook's returned error when old hook returns nil")
	require.NoError(t, trace.Got1xxResponse(102, nil))
	require.EqualValues(t, 1, observerCalls.Load())
	require.EqualValues(t, 2, callerCalls.Load())
	require.Equal(t, captured, err.Error())
}

func TestObserveInformationalSuccessfulResponseRacesAbort(t *testing.T) {
	first, closing := errors.New("observer"), errors.New("close")
	entered, release := make(chan struct{}), make(chan struct{})
	//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
	req, finish, trace := informationalTrace(t, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(context.Context, int, http.Header) error {
		return first
	}}}, &httptrace.ClientTrace{Got1xxResponse: func(int, textproto.MIMEHeader) error {
		close(entered)
		<-release
		return nil
	}})
	done := make(chan error, 1)
	go func() {
		done <- trace.Got1xxResponse(103, nil)
	}()
	<-entered
	require.ErrorIs(t, req.Context().Err(), context.Canceled)
	body := &informationalBodyProbe{Reader: strings.NewReader("already returned"), closeErr: closing}
	response, err := finish(&http.Response{Body: body}, nil)
	require.EqualValues(t, 1, body.closes.Load())
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Nil(t, response)
	require.ErrorIs(t, err, first)
	require.ErrorIs(t, err, closing)
	require.EqualValues(t, 1, body.closes.Load())
	close(release)
	require.NoError(t, <-done)
}

func TestObserveInformationalBodyOwnership(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, restore := range []bool{false, true} {
			t.Run(informationalBodyName(stream, restore), func(t *testing.T) {
				//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
				req, finish, _ := informationalTrace(t, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(context.Context, int, http.Header) error {
					return nil
				}}}, nil)
				inner := &informationalBodyProbe{Reader: strings.NewReader("payload")}
				response, err := finish(&http.Response{Body: inner}, nil)
				require.NoError(t, err)
				require.NoError(t, req.Context().Err(), "successful Do must not cancel")
				result, err := DecodeResponse(response, restore, stream, func(*http.Response) (any, error) {
					return "metadata", nil
				})
				require.NoError(t, err)
				require.Equal(t, "metadata", result)
				if stream {
					require.NoError(t, req.Context().Err())
					require.Zero(t, inner.closes.Load())
					data, readErr := io.ReadAll(response.Body)
					require.NoError(t, readErr)
					require.Equal(t, "payload", string(data))
					require.ErrorIs(t, req.Context().Err(), context.Canceled)
					require.NoError(t, response.Body.Close())
				} else {
					require.ErrorIs(t, req.Context().Err(), context.Canceled)
					if restore {
						data, readErr := io.ReadAll(response.Body)
						require.NoError(t, readErr)
						require.Equal(t, "payload", string(data))
					}
				}
				require.EqualValues(t, 1, inner.closes.Load())
			})
		}
	}
}

func TestObserveInformationalNoBodyCancels(t *testing.T) {
	//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
	req, finish, _ := informationalTrace(t, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(context.Context, int, http.Header) error {
		return nil
	}}}, nil)
	//nolint:bodyclose // DecodeResponse closes the transferred body: https://github.com/timakin/bodyclose/issues/30. Remove when the analyzer follows this ownership transfer.
	response, err := finish(&http.Response{Body: http.NoBody}, nil)
	require.NoError(t, err)
	require.NoError(t, req.Context().Err())
	_, err = DecodeResponse(response, false, false, func(*http.Response) (any, error) {
		return nil, nil
	})
	require.NoError(t, err)
	require.ErrorIs(t, req.Context().Err(), context.Canceled)
}

func TestObserveInformationalCloseEOFRaceCancelsOnce(t *testing.T) {
	// Exercise the concrete wrapper's once primitive with EOF during inner Close.
	closing := errors.New("close")
	entered, release := make(chan struct{}), make(chan struct{})
	inner := &informationalBodyProbe{Reader: strings.NewReader(""), closeErr: closing, closeEntered: entered, closeRelease: release}
	var cancels atomic.Int32
	body := newExchangeBody(inner, func() {
		cancels.Add(1)
	})
	done := make(chan error, 1)
	go func() {
		done <- body.Close()
	}()
	<-entered
	require.Zero(t, cancels.Load(), "Close must forward before cancellation")
	n, err := body.Read(make([]byte, 1))
	require.Zero(t, n)
	require.ErrorIs(t, err, io.EOF)
	require.EqualValues(t, 1, cancels.Load())
	close(release)
	require.Same(t, closing, <-done)
	require.EqualValues(t, 1, cancels.Load())
	require.EqualValues(t, 1, inner.closes.Load())
}

func TestObserveInformationalConcurrentInvocations(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
			observed, finish := ObserveInformational(req, []InformationalObserver{{Selector: StatusSelector{Code: 103}, Observe: func(context.Context, int, http.Header) error {
				return nil
			}}})
			trace := httptrace.ContextClientTrace(observed.Context())
			if trace == nil {
				t.Error("missing owned trace")
				return
			}
			trace.GotFirstResponseByte()
			for range 8 {
				if err := trace.Got1xxResponse(103, nil); err != nil {
					t.Error(err)
				}
			}
			response, err := finish(&http.Response{Body: http.NoBody}, nil)
			if err != nil {
				t.Error(err)
				return
			}
			if err := response.Body.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	require.Nil(t, httptrace.ContextClientTrace(req.Context()))
	require.NoError(t, req.Context().Err())
}

func informationalTrace(t *testing.T, observers []InformationalObserver, caller *httptrace.ClientTrace) (*http.Request, func(*http.Response, error) (*http.Response, error), *httptrace.ClientTrace) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if caller != nil {
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), caller))
	}
	//nolint:bodyclose // Factory returns a function, not a response: https://github.com/timakin/bodyclose/issues/71#issuecomment-6709847919. Remove when the pinned analyzer distinguishes nested function signatures.
	observed, finish := ObserveInformational(req, observers)
	trace := httptrace.ContextClientTrace(observed.Context())
	require.NotNil(t, trace, "active observer must install a per-Do trace")
	require.NotNil(t, trace.GotFirstResponseByte, "per-exchange reset must be installed")
	require.NotNil(t, trace.Got1xxResponse, "interim observation must be installed")
	trace.GotFirstResponseByte()
	return observed, finish, trace
}

func informationalCodes(code, count int) []int {
	codes := make([]int, count)
	for i := range codes {
		codes[i] = code
	}
	return codes
}

func informationalBodyName(stream, restore bool) string {
	name := "ordinary"
	if stream {
		name = "stream"
	}
	if restore {
		name += "_restore"
	}
	return name
}

func (b *informationalBodyProbe) Close() error {
	b.closes.Add(1)
	if b.closeEntered != nil {
		close(b.closeEntered)
		<-b.closeRelease
	}
	return b.closeErr
}
