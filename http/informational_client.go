package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"slices"
	"sync"
)

type (
	// InformationalObserver binds a declared informational selector to an optional
	// observer. Keep declarations with no registered observer in the slice too.
	InformationalObserver struct {
		// Selector is the declared informational response key.
		Selector StatusSelector
		// Observe synchronously decodes and observes a selected response. Headers
		// are read-only during this call. A nonnil error aborts this Do invocation.
		// A nil function leaves this selector declared but unobserved.
		Observe func(context.Context, int, http.Header) error
	}
	informationalClient struct {
		mu        sync.Mutex
		firstErr  error
		hundred   int
		other     int
		keys      []StatusSelector
		observers []InformationalObserver
		cancel    context.CancelFunc
	}
	exchangeBody struct {
		io.ReadCloser
		cancel func()
	}
)

// ObserveInformational prepares one Do invocation and returns its finish function.
// Call finish exactly once with Do's response and error, before decoding. All
// declared selectors must be supplied, including entries with nil Observe.
// With no registered observers it preserves the request and finish arguments.
//
// Observers run serially during admission. Once an observer or budget fails,
// later deliveries invoke no Loom observer; the caller's trace remains composed
// by net/http. finish retains the first Loom error, the error Do returned and
// any returned body's Close error. Successful bodies remain open until their
// normal owner closes them or reads EOF; cancellation is retained until then.
//
// Budgets reset at GotFirstResponseByte, which native HTTP/1 and HTTP/2 deliver
// once per response exchange before its informational responses. A custom Doer
// must preserve that trace ordering. Request-write notifications do not reset
// budgets: HTTP/2 may deliver responses before WroteHeaders runs.
func ObserveInformational(req *http.Request, observers []InformationalObserver) (*http.Request, func(*http.Response, error) (*http.Response, error)) {
	if !slices.ContainsFunc(observers, func(o InformationalObserver) bool {
		return o.Observe != nil
	}) {
		return req, func(resp *http.Response, err error) (*http.Response, error) {
			return resp, err
		}
	}
	ctx, cancel := context.WithCancel(req.Context())
	state := &informationalClient{
		keys:      make([]StatusSelector, len(observers)),
		observers: slices.Clone(observers),
		cancel:    cancel,
	}
	for i, observer := range state.observers {
		state.keys[i] = observer.Selector
	}
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			state.mu.Lock()
			state.hundred, state.other = 0, 0
			state.mu.Unlock()
		},
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			state.mu.Lock()
			err := state.firstErr
			if err == nil {
				err = state.admit(ctx, code, http.Header(header))
				if err != nil {
					state.firstErr = err
				}
			}
			state.mu.Unlock()
			if err != nil {
				// Go's composed hook drops the new hook's error when an old hook exists:
				// https://github.com/golang/go/issues/76669 and
				// https://github.com/golang/go/issues/76669#issuecomment-6675485882.
				// Store before cancellation, since Do may settle before the old hook.
				// Remove this workaround when the pinned Go propagates the new error and
				// TestHTTPTraceComposeDiscardsNewHookError confirms the upstream correction.
				cancel()
			}
			return err
		},
	}
	return req.WithContext(httptrace.WithClientTrace(ctx, trace)), state.finish
}

func newExchangeBody(body io.ReadCloser, cancel context.CancelFunc) *exchangeBody {
	return &exchangeBody{ReadCloser: body, cancel: sync.OnceFunc(cancel)}
}

// admit is called with the per-Do admission lock held. Budgeting precedes
// selection, so undeclared and unobserved responses consume their allowances.
func (s *informationalClient) admit(ctx context.Context, code int, header http.Header) error {
	if code == http.StatusContinue {
		if s.hundred >= 1 {
			return ErrInformationalLimit
		}
		s.hundred++
	} else {
		if s.other >= MaxInformationalResponses {
			return ErrInformationalLimit
		}
		s.other++
	}
	selected, ok := SelectInformational(s.keys, code)
	if !ok {
		return nil
	}
	for _, observer := range s.observers {
		if observer.Selector == selected {
			if observer.Observe == nil {
				return nil
			}
			return observer.Observe(ctx, code, header)
		}
	}
	return nil
}

func (s *informationalClient) finish(resp *http.Response, doErr error) (*http.Response, error) {
	s.mu.Lock()
	first := s.firstErr
	s.mu.Unlock()
	if first != nil {
		var closeErr error
		if resp != nil && resp.Body != nil {
			closeErr = resp.Body.Close()
		}
		s.cancel()
		return nil, errors.Join(first, doErr, closeErr)
	}
	if doErr != nil {
		s.cancel()
		return nil, doErr
	}
	resp.Body = newExchangeBody(resp.Body, s.cancel)
	return resp, nil
}

func (b *exchangeBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err == io.EOF {
		b.cancel()
	}
	return n, err
}

func (b *exchangeBody) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}
