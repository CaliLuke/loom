package transport

import (
	"context"
	"net/http"
	"time"
)

// RequestObserver tracks a single transport request lifecycle and emits
// Start, terminal, and panic events through an Observer attached to the
// request context. Generated HTTP, JSON-RPC, and MCP transports use the
// observer to consolidate per-request boilerplate into a small surface:
//
//	obs, w := transport.BeginHTTPRequest(ctx, w, "Service", "Method", r)
//	defer obs.End()
//	// ...
//	obs.Fail(transport.ReasonRequestDecodeFailed) // before an error return
//
// End must be deferred so panics propagate through it as
// EventKindRequestFailure with [ReasonPanic] and are re-panicked; if End is
// reached without a recorded failure, EventKindRequestFinish is emitted.
type RequestObserver struct {
	ctx       context.Context
	transport TransportKind
	service   string
	method    string
	route     string
	httpVerb  string
	start     time.Time
	capture   *CaptureResponseWriter

	reason          Reason
	failed          bool
	finished        bool
	emitWithMessage string
	jsonrpcMethod   string
	jsonrpcID       string
	batchCount      int
	notification    bool
	sessionID       string
}

// BeginHTTPRequest wraps w with a [CaptureResponseWriter], records the
// request start time, emits an EventKindRequestStart event, and returns the
// observer together with the wrapped writer. Generated HTTP handlers
// reassign their response writer parameter to the returned writer so all
// subsequent writes are captured.
func BeginHTTPRequest(ctx context.Context, w http.ResponseWriter, service, method string, r *http.Request) (*RequestObserver, http.ResponseWriter) {
	capture := NewCaptureResponseWriter(w)
	obs := &RequestObserver{
		ctx:       ctx,
		transport: TransportHTTP,
		service:   service,
		method:    method,
		route:     httpRoute(r),
		httpVerb:  httpVerb(r),
		start:     time.Now(),
		capture:   capture,
	}
	obs.emit(EventKindRequestStart, "")
	return obs, capture
}

// BeginJSONRPCRequest wraps w with a [CaptureResponseWriter] and starts a
// request lifecycle classified as [TransportJSONRPC]. service identifies
// the Loom service; the JSON-RPC envelope's method, id, batch count, and
// notification flag are filled in later through
// [RequestObserver.SetJSONRPC] once the envelope has been decoded. Pre-
// decode rejection events therefore leave the JSON-RPC fields empty as the
// plan requires.
func BeginJSONRPCRequest(ctx context.Context, w http.ResponseWriter, service string, r *http.Request) (*RequestObserver, http.ResponseWriter) {
	capture := NewCaptureResponseWriter(w)
	obs := &RequestObserver{
		ctx:       ctx,
		transport: TransportJSONRPC,
		service:   service,
		route:     httpRoute(r),
		httpVerb:  httpVerb(r),
		start:     time.Now(),
		capture:   capture,
	}
	obs.emit(EventKindRequestStart, "")
	return obs, capture
}

// BeginRequest starts a transport request lifecycle without an HTTP
// response writer. JSON-RPC and MCP generators that need to emit start
// events outside the HTTP capture pipeline use this entry point. End and
// Fail behave identically to the HTTP variant; StatusCode and BytesWritten
// remain zero unless updated through ApplyHTTPStatus.
func BeginRequest(ctx context.Context, kind TransportKind, service, method string) *RequestObserver {
	obs := &RequestObserver{
		ctx:       ctx,
		transport: kind,
		service:   service,
		method:    method,
		start:     time.Now(),
	}
	obs.emit(EventKindRequestStart, "")
	return obs
}

// Fail records reason as the terminal classification for the request. The
// first call wins so the originating failure is not overwritten by
// downstream cleanup classifications. End will emit
// EventKindRequestFailure with the recorded reason.
func (o *RequestObserver) Fail(reason Reason) {
	if o == nil || o.failed {
		return
	}
	o.failed = true
	o.reason = reason
}

// FailWithMessage records reason and a redaction-safe message for the
// terminal event. The message is reported through Event.SafeMessage and
// must not contain raw user input, credentials, or payload data.
func (o *RequestObserver) FailWithMessage(reason Reason, safeMessage string) {
	if o == nil || o.failed {
		return
	}
	o.failed = true
	o.reason = reason
	o.emitWithMessage = safeMessage
}

// SetJSONRPC populates JSON-RPC-specific fields on the terminal event. It
// is safe to call once the JSON-RPC envelope has been decoded. When the
// observer's operation name was not provided at Begin time, method is also
// used as the observer's Method so the terminal event reports the JSON-RPC
// method as the operation name.
func (o *RequestObserver) SetJSONRPC(method, id string, batchCount int, notification bool) {
	if o == nil {
		return
	}
	o.jsonrpcMethod = method
	o.jsonrpcID = id
	o.batchCount = batchCount
	o.notification = notification
	if o.method == "" {
		o.method = method
	}
}

// SetSession records the transport session identifier carried by the
// request, when known.
func (o *RequestObserver) SetSession(sessionID string) {
	if o == nil {
		return
	}
	o.sessionID = sessionID
}

// End emits the terminal request event. End must be deferred so a
// recovered panic emits EventKindRequestFailure with [ReasonPanic] before
// re-panicking. End is a no-op on a nil observer or after the first call,
// allowing it to be safely composed with explicit failure emission.
func (o *RequestObserver) End() {
	if o == nil {
		return
	}
	if rec := recover(); rec != nil {
		o.failed = true
		o.reason = ReasonPanic
		o.emit(EventKindRequestFailure, ReasonPanic)
		o.finished = true
		panic(rec)
	}
	if o.finished {
		return
	}
	o.finished = true
	if o.failed {
		o.emit(EventKindRequestFailure, o.reason)
		return
	}
	o.emit(EventKindRequestFinish, ReasonOK)
}

// EmitStreamOpen reports the opening of a streaming response within the
// current request. Generated SSE and WebSocket servers call EmitStreamOpen
// once the stream headers have been negotiated.
func (o *RequestObserver) EmitStreamOpen() {
	if o == nil {
		return
	}
	o.emit(EventKindStreamOpen, ReasonOK)
}

// EmitStreamClose reports the natural close of a streaming response.
func (o *RequestObserver) EmitStreamClose() {
	if o == nil {
		return
	}
	o.emit(EventKindStreamClose, ReasonOK)
}

// EmitStreamFailure reports a failed write or flush during streaming. The
// terminal request event is unaffected; callers that want the failure to
// also classify the request must call Fail with the same reason.
func (o *RequestObserver) EmitStreamFailure(reason Reason) {
	if o == nil {
		return
	}
	o.emit(EventKindStreamFailure, reason)
}

func (o *RequestObserver) emit(kind EventKind, reason Reason) {
	e := Event{
		Kind:          kind,
		Reason:        reason,
		Transport:     o.transport,
		Service:       o.service,
		Method:        o.method,
		Route:         o.route,
		HTTPMethod:    o.httpVerb,
		Duration:      time.Since(o.start),
		JSONRPCMethod: o.jsonrpcMethod,
		JSONRPCID:     o.jsonrpcID,
		BatchCount:    o.batchCount,
		Notification:  o.notification,
		SessionID:     o.sessionID,
		SafeMessage:   o.emitWithMessage,
	}
	if o.capture != nil {
		e.StatusCode = o.capture.StatusCode()
		e.BytesWritten = o.capture.BytesWritten()
	}
	Observe(o.ctx, e)
	o.emitWithMessage = ""
}

type requestObserverKey struct{}

// WithRequestObserver returns a copy of ctx that carries obs so deeper
// generated layers (JSON-RPC method dispatch, MCP tool handlers) can
// resolve the lifecycle observer without threading a new parameter through
// every signature.
func WithRequestObserver(ctx context.Context, obs *RequestObserver) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestObserverKey{}, obs)
}

// RequestObserverFromContext returns the RequestObserver injected by
// [WithRequestObserver], or nil if none is attached. Generated code uses
// nil-safe methods on the returned observer so a missing observer remains
// a cheap no-op.
func RequestObserverFromContext(ctx context.Context) *RequestObserver {
	if ctx == nil {
		return nil
	}
	obs, _ := ctx.Value(requestObserverKey{}).(*RequestObserver)
	return obs
}

func httpRoute(r *http.Request) string {
	if r == nil {
		return ""
	}
	return r.Pattern
}

func httpVerb(r *http.Request) string {
	if r == nil {
		return ""
	}
	return r.Method
}
