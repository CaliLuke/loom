package jsonrpc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

type (
	// SSEDispatch calls a generated typed SSE method adapter. unary reports
	// whether an ID-less successful request must finish with HTTP 204.
	SSEDispatch func(
		context.Context,
		*http.Request,
		*RawRequest,
		http.ResponseWriter,
	) (matched bool, unary bool, err error)

	// SSEErrorSender writes one JSON-RPC error as an SSE event.
	SSEErrorSender func(
		context.Context,
		*http.Request,
		http.ResponseWriter,
		any,
		Code,
		string,
		any,
	) error

	// SSEHandlerSpec defines the generated adapters used by the JSON-RPC SSE
	// protocol runtime.
	SSEHandlerSpec struct {
		// Service is the designed service name.
		Service string
		// Decoder returns the decoder for one HTTP request.
		Decoder func(*http.Request) loomhttp.Decoder
		// Dispatch calls a generated typed SSE method adapter.
		Dispatch SSEDispatch
		// SendError writes one protocol error event.
		SendError SSEErrorSender
		// HandleFailure receives transport failures.
		HandleFailure func(context.Context, http.ResponseWriter, error)
	}

	// MixedHandlerSpec defines HTTP and SSE adapters for a service that
	// negotiates both JSON-RPC transports on one route.
	MixedHandlerSpec struct {
		// HTTP defines the unary and batch HTTP runtime.
		HTTP HTTPHandlerSpec
		// SSE defines the SSE runtime.
		SSE SSEHandlerSpec
		// SupportsGET reports whether events/stream is designed.
		SupportsGET bool
	}

	// WebSocketHandlerSpec defines the generated adapters used to establish a
	// JSON-RPC WebSocket service stream.
	WebSocketHandlerSpec struct {
		// Upgrader upgrades the HTTP connection.
		Upgrader loomhttp.Upgrader
		// Configure optionally configures the upgraded connection.
		Configure loomhttp.ConnConfigureFunc
		// WritePolicy bounds WebSocket writes.
		WritePolicy loomhttp.StreamWritePolicy
		// Run constructs and runs the generated typed service stream.
		Run func(context.Context, context.CancelFunc, *http.Request, http.ResponseWriter, *loomhttp.WebSocketStream) error
		// HandleFailure receives upgrade, stream, and close failures.
		HandleFailure func(context.Context, http.ResponseWriter, error)
	}

	// WebSocketDispatch calls a generated typed WebSocket method adapter.
	WebSocketDispatch func(context.Context, *RawRequest) error

	// WebSocketMethodMatcher reports whether a generated WebSocket service
	// defines a method.
	WebSocketMethodMatcher func(string) bool

	// WebSocketErrorSender writes one JSON-RPC error frame.
	WebSocketErrorSender func(context.Context, any, Code, string, any) error
)

// ServeSSE executes one JSON-RPC SSE request with generated typed adapters.
func ServeSSE(w http.ResponseWriter, r *http.Request, spec SSEHandlerSpec) {
	observer, observedWriter := loomtransport.BeginJSONRPCRequest(r.Context(), w, spec.Service, r)
	defer observer.End()
	ctx := loomtransport.WithRequestObserver(r.Context(), observer)
	r = r.WithContext(ctx)

	var request RawRequest
	if r.Method == http.MethodGet {
		request = RawRequest{JSONRPC: "2.0", Method: "events/stream"}
	} else if err := spec.Decoder(r).Decode(&request); err != nil {
		observer.Fail(loomtransport.ReasonInvalidJSONRPCEnvelope)
		code, message, data := envelopeDecodeError(err)
		if sendErr := spec.SendError(ctx, r, observedWriter, nil, code, message, data); sendErr != nil {
			handleHTTPFailure(ctx, observedWriter, spec.HandleFailure, sendErr)
		}
		return
	}

	observer.SetJSONRPC(request.Method, IDToString(request.ID), 0, !request.HasID)
	if reason, message := invalidStreamRequest(&request); reason != loomtransport.ReasonOK {
		observer.Fail(reason)
		writeSSEProtocolError(ctx, observedWriter, r, &request, InvalidRequest, message, spec)
		return
	}

	matched, unary, err := spec.Dispatch(ctx, r, &request, observedWriter)
	if err != nil {
		observer.Fail(loomtransport.ReasonHandlerError)
		handleHTTPFailure(ctx, observedWriter, spec.HandleFailure, fmt.Errorf("handler error for %s: %w", request.Method, err))
		return
	}
	if !matched {
		observer.Fail(loomtransport.ReasonUnsupportedMethod)
		writeSSEProtocolError(ctx, observedWriter, r, &request, MethodNotFound, "Method not found", spec)
		return
	}
	if unary && !request.HasID {
		observedWriter.WriteHeader(http.StatusNoContent)
	}
}

// ServeMixed negotiates JSON-RPC HTTP or SSE handling for one route.
func ServeMixed(w http.ResponseWriter, r *http.Request, spec MixedHandlerSpec) {
	switch r.Method {
	case http.MethodGet:
		if !spec.SupportsGET {
			http.NotFound(w, r)
			return
		}
		ServeSSE(w, r, spec.SSE)
	case http.MethodPost:
		if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			ServeHTTP(w, r, spec.HTTP)
			return
		}
		usesHTTP, err := mixedRequestUsesHTTP(r)
		if err != nil {
			handleHTTPFailure(r.Context(), w, spec.HTTP.HandleFailure, err)
			return
		}
		if usesHTTP {
			ServeHTTP(w, r, spec.HTTP)
			return
		}
		ServeSSE(w, r, spec.SSE)
	default:
		http.NotFound(w, r)
	}
}

// ServeWebSocket upgrades one HTTP request and runs a generated typed stream.
func ServeWebSocket(w http.ResponseWriter, r *http.Request, spec WebSocketHandlerSpec) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	conn, err := spec.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		handleHTTPFailure(r.Context(), w, spec.HandleFailure, fmt.Errorf("failed to upgrade to WebSocket: %w", err))
		return
	}
	if spec.Configure != nil {
		conn = spec.Configure(conn, cancel)
	}
	stream := loomhttp.NewWebSocketStream(conn, spec.WritePolicy)
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			handleHTTPFailure(ctx, w, spec.HandleFailure, fmt.Errorf("failed to close WebSocket: %w", closeErr))
		}
	}()
	if err := spec.Run(ctx, cancel, r, w, stream); err != nil {
		handleHTTPFailure(ctx, w, spec.HandleFailure, err)
	}
}

// ReceiveWebSocketRequest reads, validates, and dispatches one JSON-RPC
// WebSocket request frame.
func ReceiveWebSocketRequest(
	ctx context.Context,
	stream *loomhttp.WebSocketStream,
	matches WebSocketMethodMatcher,
	dispatch WebSocketDispatch,
	sendError WebSocketErrorSender,
) error {
	var request RawRequest
	if err := stream.ReadJSON(ctx, &request); err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			return err
		}
		if sendErr := sendError(ctx, nil, ParseError, "Parse error", nil); sendErr != nil {
			return fmt.Errorf("failed to send parse error: %w", sendErr)
		}
		return nil
	}
	if _, message := invalidStreamRequest(&request); message != "" {
		if !request.HasID {
			return nil
		}
		return sendError(ctx, request.ID, InvalidRequest, message, nil)
	}
	if !matches(request.Method) {
		if !request.HasID {
			return nil
		}
		return sendError(ctx, request.ID, MethodNotFound, "Method not found", nil)
	}
	return dispatch(ctx, &request)
}

// CompleteStream sends a final JSON-RPC success response when the initiating
// request has an ID. Notification and GET-listener completions are suppressed.
func CompleteStream(
	ctx context.Context,
	hasID bool,
	id any,
	result any,
	send func(*Response) error,
) error {
	if !hasID {
		observeSuppressedStreamResponse(ctx)
		return nil
	}
	return send(MakeSuccessResponse(id, result))
}

// CompleteStreamError sends a final JSON-RPC error response when the initiating
// request has an ID. Notification errors are suppressed.
func CompleteStreamError(ctx context.Context, hasID bool, send func() error) error {
	if !hasID {
		observeSuppressedStreamResponse(ctx)
		return nil
	}
	return send()
}

func observeSuppressedStreamResponse(ctx context.Context) {
	loomtransport.Observe(ctx, loomtransport.Event{
		Kind:      loomtransport.EventKindStreamClose,
		Reason:    loomtransport.ReasonStreamFinalResponseSuppressed,
		Transport: loomtransport.TransportJSONRPC,
	})
}

func invalidStreamRequest(request *RawRequest) (loomtransport.Reason, string) {
	reason, message := invalidRequest(request)
	if reason == loomtransport.ReasonInvalidJSONRPCMethod {
		return reason, "Invalid request"
	}
	return reason, message
}

func writeSSEProtocolError(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	request *RawRequest,
	code Code,
	message string,
	spec SSEHandlerSpec,
) {
	if !request.HasID {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := spec.SendError(ctx, r, w, request.ID, code, message, nil); err != nil {
		handleHTTPFailure(ctx, w, spec.HandleFailure, err)
	}
}

func mixedRequestUsesHTTP(r *http.Request) (bool, error) {
	reader := bufio.NewReader(r.Body)
	var first byte
	sniffed := 0
	for sniffed < maxJSONPrefixWhitespace {
		peek, err := reader.Peek(1)
		if err != nil && err != io.EOF {
			return false, fmt.Errorf("failed to read request body: %w", err)
		}
		if len(peek) == 0 {
			break
		}
		first = peek[0]
		if first != ' ' && first != '\t' && first != '\r' && first != '\n' {
			break
		}
		if _, err := reader.Discard(1); err != nil {
			return false, fmt.Errorf("failed to read request body: %w", err)
		}
		sniffed++
	}
	r.Body = &requestBody{Reader: reader, Closer: r.Body}
	return first == 0 || first == '[' || sniffed >= maxJSONPrefixWhitespace, nil
}
