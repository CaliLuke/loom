package transport

import "time"

type (
	// EventKind identifies the lifecycle position a transport event reports.
	EventKind uint8

	// Reason is a stable, low-cardinality classification of why an event was
	// emitted. Reason values are part of the package's public contract and are
	// safe to use as metric labels.
	Reason string

	// TransportKind identifies the generated transport that produced an event.
	TransportKind string

	// Event is the payload delivered to an Observer. All fields are optional;
	// generators populate the fields that are well defined at the point where
	// the event is emitted and leave the rest zero. Generated code never
	// populates Event with raw bodies, JSON-RPC params, MCP tool arguments,
	// credentials, or result payloads.
	Event struct {
		// Kind is the lifecycle position of the event.
		Kind EventKind
		// Transport identifies the generated transport (HTTP, JSON-RPC, MCP).
		Transport TransportKind
		// Reason is the stable classification for this event.
		Reason Reason

		// Service is the Loom service name owning the endpoint.
		Service string
		// Method is the operation name (HTTP handler, JSON-RPC method, or MCP
		// tool) as known to the generator.
		Method string
		// Route is the HTTP route pattern when available.
		Route string
		// HTTPMethod is the HTTP verb when the underlying transport is HTTP.
		HTTPMethod string
		// StatusCode is the final HTTP status code when known.
		StatusCode int
		// Duration is the wall-clock duration between request start and the
		// emission of the terminal event.
		Duration time.Duration
		// BytesWritten counts bytes flushed to the response writer when known.
		BytesWritten int64

		// JSONRPCMethod is the JSON-RPC method name decoded from the request
		// envelope.
		JSONRPCMethod string
		// JSONRPCID is the decoded JSON-RPC request id rendered as a string.
		JSONRPCID string
		// BatchCount is the number of envelopes in a JSON-RPC batch request,
		// zero for non-batch requests.
		BatchCount int
		// Notification reports whether the JSON-RPC request had no id and is a
		// notification.
		Notification bool

		// SessionID is the transport session identifier when known. For MCP
		// this is the Mcp-Session-Id; for WebSocket and SSE streams it is the
		// stream session id.
		SessionID string
		// RequestID is the generator-assigned request correlation id when
		// known.
		RequestID string
		// TraceID is the propagated trace id when known.
		TraceID string

		// ErrorClass categorizes the error using a short, redaction-safe
		// label.
		ErrorClass string
		// SafeMessage is a redaction-safe error message. Generated code never
		// places raw error messages from untrusted sources in this field.
		SafeMessage string

		// Attrs carries additional redaction-safe key/value attributes. Keys
		// and values are intended to be safe for log enrichment and metric
		// labeling; raw payloads must not be placed here.
		Attrs map[string]string
	}
)

const (
	// EventKindRequestStart marks the beginning of a request lifecycle.
	EventKindRequestStart EventKind = iota + 1
	// EventKindRequestFinish marks a successful terminal request event.
	EventKindRequestFinish
	// EventKindRequestFailure marks a failed terminal request event.
	EventKindRequestFailure
	// EventKindStreamOpen marks the opening of a streaming response.
	EventKindStreamOpen
	// EventKindStreamClose marks the closing of a streaming response.
	EventKindStreamClose
	// EventKindStreamFailure marks a failed stream write or flush.
	EventKindStreamFailure
)

const (
	// TransportHTTP identifies generic HTTP-generated transport events.
	TransportHTTP TransportKind = "http"
	// TransportJSONRPC identifies JSON-RPC-generated transport events.
	TransportJSONRPC TransportKind = "jsonrpc"
	// TransportMCP identifies Loom-MCP-generated transport events.
	TransportMCP TransportKind = "mcp"
)

const (
	// ReasonOK marks a terminal event with no error classification.
	ReasonOK Reason = "ok"
	// ReasonRequestDecodeFailed marks failure to decode the HTTP request body
	// or query parameters into the generated request type.
	ReasonRequestDecodeFailed Reason = "request_decode_failed"
	// ReasonInvalidJSONRPCEnvelope marks failure to parse the JSON-RPC
	// envelope.
	ReasonInvalidJSONRPCEnvelope Reason = "invalid_jsonrpc_envelope"
	// ReasonInvalidJSONRPCBatch marks rejection of a malformed JSON-RPC batch
	// request.
	ReasonInvalidJSONRPCBatch Reason = "invalid_jsonrpc_batch"
	// ReasonInvalidJSONRPCMethod marks rejection of a JSON-RPC envelope whose
	// method field is missing or malformed.
	ReasonInvalidJSONRPCMethod Reason = "invalid_jsonrpc_method"
	// ReasonInvalidJSONRPCParams marks failure to decode JSON-RPC params into
	// the generated parameter type.
	ReasonInvalidJSONRPCParams Reason = "invalid_jsonrpc_params"
	// ReasonUnsupportedMethod marks rejection of a JSON-RPC method the
	// generated handler does not implement.
	ReasonUnsupportedMethod Reason = "unsupported_method"
	// ReasonMissingCredentials marks rejection because required credentials
	// were not presented.
	ReasonMissingCredentials Reason = "missing_credentials"
	// ReasonInvalidCredentials marks rejection because credentials were
	// presented but could not be validated.
	ReasonInvalidCredentials Reason = "invalid_credentials"
	// ReasonPermissionRejected marks rejection because authorization checks
	// denied the principal.
	ReasonPermissionRejected Reason = "permission_rejected"
	// ReasonPrincipalMismatch marks rejection because the principal did not
	// match the session owner.
	ReasonPrincipalMismatch Reason = "principal_mismatch"
	// ReasonHandlerError marks a handler returning a non-nil error.
	ReasonHandlerError Reason = "handler_error"
	// ReasonPanic marks a recovered panic in the generated handler.
	ReasonPanic Reason = "panic"
	// ReasonResponseWriteFailed marks a failure while writing the HTTP
	// response body.
	ReasonResponseWriteFailed Reason = "response_write_failed"
	// ReasonStreamWriteFailed marks a failed write to a streaming response.
	ReasonStreamWriteFailed Reason = "stream_write_failed"
	// ReasonStreamFlushFailed marks a failed flush of a streaming response.
	ReasonStreamFlushFailed Reason = "stream_flush_failed"
	// ReasonStreamWriteTimeout marks a timed-out streaming response write.
	ReasonStreamWriteTimeout Reason = "stream_write_timeout"
	// ReasonStreamFlushTimeout marks a timed-out streaming response flush.
	ReasonStreamFlushTimeout Reason = "stream_flush_timeout"
	// ReasonStreamFinalResponseSuppressed marks a SendAndClose final value
	// discarded because the stream carries no JSON-RPC request ID (a
	// notification or a raw GET events/stream listener), so protocol rules
	// forbid sending a final response.
	ReasonStreamFinalResponseSuppressed Reason = "stream_final_response_suppressed"
	// ReasonMCPSessionMissing marks an MCP events-stream request that omitted
	// the Mcp-Session-Id header.
	ReasonMCPSessionMissing Reason = "mcp_session_missing"
	// ReasonMCPSessionNotFound marks an MCP events-stream request whose
	// session id is unknown to the server.
	ReasonMCPSessionNotFound Reason = "mcp_session_not_found"
	// ReasonMCPSessionPrincipalMismatch marks an MCP events-stream request
	// whose authenticated principal differs from the session owner.
	ReasonMCPSessionPrincipalMismatch Reason = "mcp_session_principal_mismatch"
	// ReasonMCPEventsStreamWriteFailed marks a failure writing an MCP
	// notification frame to the events-stream response.
	ReasonMCPEventsStreamWriteFailed Reason = "mcp_events_stream_write_failed"
)
