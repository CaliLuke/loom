package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

type (
	// ExchangeWriter serializes authored informational responses with final
	// admission for one HTTP request. Obtain it from HandlerLifecycle or its
	// context; its zero value is not an initialized exchange.
	ExchangeWriter struct {
		writer http.ResponseWriter
		major  int
		minor  int
		mu     sync.Mutex
		final  bool
		sent   int
	}

	// FinalKind identifies the caller beginning final response work. It does
	// not imply physical commitment or successful publication.
	FinalKind uint8

	exchangeContextKey struct{}
)

const (
	// FinalResult marks result encoding, or handler-end admission cleanup.
	FinalResult FinalKind = iota
	// FinalError marks error routing or encoding.
	FinalError
	// FinalSSEOpen marks the first SSE handshake preparation.
	FinalSSEOpen
	// FinalUpgrade marks WebSocket upgrade preparation.
	FinalUpgrade
	// FinalFile marks file response preparation.
	FinalFile
	// FinalRaw marks raw response preparation, before any body peek.
	FinalRaw
	// FinalRedirect marks redirect header preparation.
	FinalRedirect

	// MaxInformationalResponses bounds authored interim responses per exchange.
	// The transport's automatic 100 Continue has an independent budget.
	MaxInformationalResponses = 8
)

var (
	// ErrInformationalClosed means final work has closed interim admission.
	ErrInformationalClosed = errors.New("loom http informational admission closed")
	// ErrInformationalUnsupported means the request protocol cannot carry 1xx.
	ErrInformationalUnsupported = errors.New("loom http informational response unsupported")
	// ErrInformationalLimit means the authored interim response budget is spent.
	ErrInformationalLimit = errors.New("loom http informational response limit exceeded")
)

// ExchangeFromContext returns the request's shared informational writer, or nil
// when ctx was not obtained from a HandlerLifecycle.
func ExchangeFromContext(ctx context.Context) *ExchangeWriter {
	exchange, _ := ctx.Value(exchangeContextKey{}).(*ExchangeWriter)
	return exchange
}

// Send publishes one authored informational response with exactly headers,
// then restores the final response header map. Only statuses 102 through 199
// are permitted; generated emitters separately validate their declared domains
// and header constraints. Callers must not mutate headers during Send.
func (e *ExchangeWriter) Send(code int, headers http.Header) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.final {
		return ErrInformationalClosed
	}
	if e.major < 1 || (e.major == 1 && e.minor == 0) {
		return ErrInformationalUnsupported
	}
	if e.sent >= MaxInformationalResponses {
		return ErrInformationalLimit
	}
	if code <= http.StatusSwitchingProtocols || code >= http.StatusOK {
		return fmt.Errorf("invalid authored informational status %d: must be 102 through 199", code)
	}
	header := e.writer.Header()
	snapshot := header.Clone()
	defer replaceHeaders(header, snapshot)
	replaceHeaders(header, headers.Clone())
	e.sent++
	e.writer.WriteHeader(code)
	return nil
}

// BeginFinal permanently closes informational admission before final headers
// are read or prepared. It waits for an admitted Send, is idempotent and performs
// no I/O or observation callback. kind documents the calling path only: repeated
// kinds do not change behavior. Physical commit and precommit fallback remain
// owned by the existing captured response writer and handler lifecycle.
func (e *ExchangeWriter) BeginFinal(_ FinalKind) {
	e.mu.Lock()
	e.final = true
	e.mu.Unlock()
}
