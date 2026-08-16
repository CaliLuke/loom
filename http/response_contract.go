package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/gorilla/websocket"
)

type (
	// ResponseContractCaseKind identifies whether a generated response contract
	// case describes a successful result or a service error.
	ResponseContractCaseKind string

	// ResponseContractTransport identifies the wire protocol validated by a
	// generated response contract case.
	ResponseContractTransport string

	// ResponseContractCase describes the HTTP wire invariants for one declared
	// response branch. It does not describe how application code reaches that
	// branch.
	ResponseContractCase struct {
		// ID is stable while the service response contract is unchanged.
		ID string
		// Kind identifies a successful result or service error response.
		Kind ResponseContractCaseKind
		// Transport identifies the response protocol.
		Transport ResponseContractTransport
		// StatusCode is the exact declared HTTP status code.
		StatusCode int
		// ErrorName is the expected Loom-Error header value for an error case.
		ErrorName string
		// ContentTypes lists the allowed response media types.
		ContentTypes []string
		// RequiredHeaders lists declared response headers that must be present.
		RequiredHeaders []string
		// RequiredCookies lists declared response cookies that must be present.
		RequiredCookies []string
		// Multipart describes the designed multipart request, if present.
		Multipart *MultipartRequestContract
		// SSE describes stream assertions for an SSE success case.
		SSE *SSEResponseContract
		// WebSocket describes stream assertions for a WebSocket success case.
		WebSocket *WebSocketResponseContract
	}

	// MultipartRequestContract describes the stable request shape supplied to a
	// consumer-owned multipart response scenario.
	MultipartRequestContract struct {
		// ContentType is the request media type.
		ContentType string
		// Parts lists the designed multipart fields in body order.
		Parts []MultipartPartContract
	}

	// MultipartPartContract describes one designed multipart request field.
	MultipartPartContract struct {
		// Name is the multipart form field name.
		Name string
		// MediaType is the default media type for the part value.
		MediaType string
		// Required reports whether the request body requires the part.
		Required bool
	}

	// SSEResponseContract describes the observable wire contract of an SSE stream.
	SSEResponseContract struct {
		// Direction is the designed stream direction.
		Direction string
		// MessageType is the designed streaming result type name.
		MessageType string
		// DataField is the result field encoded into SSE data, if any.
		DataField string
		// DataEncoding identifies whether SSE data is JSON or plain text.
		DataEncoding string
		// IDField is the result field encoded into SSE id, if any.
		IDField string
		// EventField is the result field encoded into SSE event, if any.
		EventField string
		// RetryField is the result field encoded into SSE retry, if any.
		RetryField string
		// IDRequired reports whether every event must include an ID.
		IDRequired bool
		// EventTypeRequired reports whether every event must include a type.
		EventTypeRequired bool
		// EventTypes lists allowed projection discriminator values, if constrained.
		EventTypes []string
		// Terminal identifies the expected stream completion behavior.
		Terminal string
	}

	// SSEResponseContractObservation contains the response and frames produced by
	// one consumer-owned SSE scenario.
	SSEResponseContractObservation struct {
		// Response is the HTTP handshake response.
		Response *http.Response
		// Events contains parsed SSE frames observed before completion.
		Events []SSEEvent
		// TerminalError is the final error returned by the stream reader.
		TerminalError error
	}

	// WebSocketResponseContract describes the observable wire contract of a
	// WebSocket stream.
	WebSocketResponseContract struct {
		// Direction is the designed stream direction.
		Direction string
		// InboundMessageType is the designed client-to-server message type name.
		InboundMessageType string
		// OutboundMessageType is the designed server-to-client message type name.
		OutboundMessageType string
		// HandshakeHeaders lists required WebSocket upgrade response headers.
		HandshakeHeaders []string
		// Terminal identifies the expected stream completion behavior.
		Terminal string
	}

	// WebSocketResponseContractObservation contains the handshake, JSON
	// messages, and terminal read result produced by one consumer-owned
	// WebSocket scenario.
	WebSocketResponseContractObservation struct {
		// Response is the HTTP upgrade response.
		Response *http.Response
		// Messages contains server-to-client JSON messages in wire order.
		Messages []json.RawMessage
		// TerminalError is the final error returned by the WebSocket reader. It is
		// nil when the contract terminates with one final response message.
		TerminalError error
	}
)

const (
	// ResponseContractSuccess identifies a successful response contract case.
	ResponseContractSuccess ResponseContractCaseKind = "success"
	// ResponseContractError identifies an error response contract case.
	ResponseContractError ResponseContractCaseKind = "error"
	// ResponseContractHTTP identifies an ordinary HTTP response contract.
	ResponseContractHTTP ResponseContractTransport = "http"
	// ResponseContractSSE identifies a Server-Sent Events response contract.
	ResponseContractSSE ResponseContractTransport = "sse"
	// ResponseContractWebSocket identifies a WebSocket response contract.
	ResponseContractWebSocket ResponseContractTransport = "websocket"
)

// ValidateResponseContract validates the transport-owned wire invariants in
// contract against resp. Applications remain responsible for arranging the
// service state and request that produce the response.
func ValidateResponseContract(resp *http.Response, contract ResponseContractCase) error {
	prefix := fmt.Sprintf("response contract %q", contract.ID)
	if resp == nil {
		return fmt.Errorf("%s: response is nil", prefix)
	}
	if resp.StatusCode != contract.StatusCode {
		return fmt.Errorf("%s: status is %d, want %d", prefix, resp.StatusCode, contract.StatusCode)
	}
	if err := validateResponseContractContentType(resp.Header, contract.ContentTypes); err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	actualErrorName := resp.Header.Get("Loom-Error")
	if contract.Kind == ResponseContractError {
		if actualErrorName != contract.ErrorName {
			return fmt.Errorf("%s: Loom-Error is %q, want %q", prefix, actualErrorName, contract.ErrorName)
		}
	} else if actualErrorName != "" {
		return fmt.Errorf("%s: Loom-Error is %q, want empty", prefix, actualErrorName)
	}
	for _, name := range contract.RequiredHeaders {
		if !responseHeaderPresent(resp.Header, name) {
			return fmt.Errorf("%s: required header %q is missing", prefix, name)
		}
	}
	for _, name := range contract.RequiredCookies {
		if !responseCookiePresent(resp, name) {
			return fmt.Errorf("%s: required cookie %q is missing", prefix, name)
		}
	}
	return nil
}

// ValidateSSEResponseContract validates an SSE handshake, observed frames, and
// clean end-of-stream behavior against contract.
func ValidateSSEResponseContract(observation *SSEResponseContractObservation, contract ResponseContractCase) error {
	prefix := fmt.Sprintf("response contract %q", contract.ID)
	if observation == nil {
		return fmt.Errorf("%s: SSE observation is nil", prefix)
	}
	if contract.Transport != ResponseContractSSE || contract.SSE == nil {
		return fmt.Errorf("%s: contract is not an SSE response", prefix)
	}
	if err := ValidateResponseContract(observation.Response, contract); err != nil {
		return err
	}
	if len(observation.Events) == 0 {
		return fmt.Errorf("%s: no SSE events were observed", prefix)
	}
	for index, event := range observation.Events {
		if event.Data == "" {
			return fmt.Errorf("%s: SSE event %d has empty data", prefix, index)
		}
		if contract.SSE.DataEncoding == "json" && !json.Valid([]byte(event.Data)) {
			return fmt.Errorf("%s: SSE event %d data is not valid JSON", prefix, index)
		}
		if contract.SSE.IDRequired && event.ID == "" {
			return fmt.Errorf("%s: SSE event %d is missing an id", prefix, index)
		}
		if contract.SSE.EventTypeRequired && event.Type == "" {
			return fmt.Errorf("%s: SSE event %d is missing an event type", prefix, index)
		}
		if len(contract.SSE.EventTypes) > 0 && !responseContractEventTypeListed(contract.SSE.EventTypes, event.Type) {
			return fmt.Errorf("%s: SSE event %d type is %q, want one of %v", prefix, index, event.Type, contract.SSE.EventTypes)
		}
	}
	if contract.SSE.Terminal == "eof" && observation.TerminalError != nil && !errors.Is(observation.TerminalError, io.EOF) {
		return fmt.Errorf("%s: SSE terminal error, want clean EOF: %w", prefix, observation.TerminalError)
	}
	return nil
}

// ValidateWebSocketResponseContract validates a WebSocket handshake, observed
// server messages, and declared terminal behavior against contract.
func ValidateWebSocketResponseContract(observation *WebSocketResponseContractObservation, contract ResponseContractCase) error {
	prefix := fmt.Sprintf("response contract %q", contract.ID)
	if observation == nil {
		return fmt.Errorf("%s: WebSocket observation is nil", prefix)
	}
	if contract.Transport != ResponseContractWebSocket || contract.WebSocket == nil {
		return fmt.Errorf("%s: contract is not a WebSocket response", prefix)
	}
	if err := ValidateResponseContract(observation.Response, contract); err != nil {
		return err
	}
	for _, name := range contract.WebSocket.HandshakeHeaders {
		if err := validateWebSocketHandshakeHeader(observation.Response.Header, name); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
	}
	if contract.WebSocket.OutboundMessageType != "" && len(observation.Messages) == 0 {
		return fmt.Errorf("%s: no WebSocket messages were observed", prefix)
	}
	for index, message := range observation.Messages {
		if !json.Valid(message) {
			return fmt.Errorf("%s: WebSocket message %d is not valid JSON", prefix, index)
		}
	}
	switch contract.WebSocket.Terminal {
	case "normal_close":
		if !websocket.IsCloseError(observation.TerminalError, websocket.CloseNormalClosure) {
			if observation.TerminalError == nil {
				return fmt.Errorf("%s: WebSocket terminal error is nil, want normal close", prefix)
			}
			return fmt.Errorf("%s: WebSocket terminal error, want normal close: %w", prefix, observation.TerminalError)
		}
	case "final_message":
		if len(observation.Messages) != 1 {
			return fmt.Errorf(
				"%s: observed %d WebSocket messages, want exactly one final message",
				prefix,
				len(observation.Messages),
			)
		}
		if observation.TerminalError != nil {
			return fmt.Errorf("%s: WebSocket final-message terminal error: %w", prefix, observation.TerminalError)
		}
	default:
		return fmt.Errorf("%s: unsupported WebSocket terminal behavior %q", prefix, contract.WebSocket.Terminal)
	}
	return nil
}

func validateWebSocketHandshakeHeader(header http.Header, name string) error {
	if !responseHeaderPresent(header, name) {
		return fmt.Errorf("WebSocket handshake header %q is missing", name)
	}
	value := responseHeaderValue(header, name)
	switch textproto.CanonicalMIMEHeaderKey(name) {
	case "Connection":
		if !responseHeaderTokenListed(header, name, "upgrade") {
			return fmt.Errorf("WebSocket handshake header %q does not contain %q", name, "Upgrade")
		}
	case "Upgrade":
		if !strings.EqualFold(strings.TrimSpace(value), "websocket") {
			return fmt.Errorf("WebSocket handshake header %q is %q, want %q", name, value, "websocket")
		}
	case "Sec-Websocket-Accept":
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("WebSocket handshake header %q is empty", name)
		}
	}
	return nil
}

func responseHeaderTokenListed(header http.Header, name, target string) bool {
	for key, values := range header {
		if !strings.EqualFold(key, name) {
			continue
		}
		for _, value := range values {
			for token := range strings.SplitSeq(value, ",") {
				if strings.EqualFold(strings.TrimSpace(token), target) {
					return true
				}
			}
		}
	}
	return false
}

func responseHeaderValue(header http.Header, target string) string {
	for name, values := range header {
		if strings.EqualFold(name, target) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func responseContractEventTypeListed(eventTypes []string, target string) bool {
	for _, eventType := range eventTypes {
		if eventType == target {
			return true
		}
	}
	return false
}

func validateResponseContractContentType(header http.Header, declared []string) error {
	if len(declared) == 0 {
		return nil
	}
	actual, _, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("parse Content-Type %q: %w", header.Get("Content-Type"), err)
	}
	expected := make([]string, 0, len(declared))
	for _, contentType := range declared {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			return fmt.Errorf("parse declared Content-Type %q: %w", contentType, err)
		}
		if responseContractMediaTypeMatches(actual, mediaType) {
			return nil
		}
		if !responseContractMediaTypeListed(expected, mediaType) {
			expected = append(expected, mediaType)
		}
	}
	return fmt.Errorf("Content-Type is %q, want one of %v", actual, expected)
}

func responseContractMediaTypeMatches(actual, expected string) bool {
	actualType, actualSubtype, ok := strings.Cut(actual, "/")
	if !ok {
		return false
	}
	expectedType, expectedSubtype, ok := strings.Cut(expected, "/")
	if !ok {
		return false
	}
	return (expectedType == "*" || expectedType == actualType) &&
		(expectedSubtype == "*" || expectedSubtype == actualSubtype)
}

func responseContractMediaTypeListed(mediaTypes []string, target string) bool {
	for _, mediaType := range mediaTypes {
		if mediaType == target {
			return true
		}
	}
	return false
}

func responseHeaderPresent(header http.Header, target string) bool {
	target = textproto.CanonicalMIMEHeaderKey(target)
	for name := range header {
		if strings.EqualFold(textproto.CanonicalMIMEHeaderKey(name), target) {
			return true
		}
	}
	return false
}

func responseCookiePresent(resp *http.Response, target string) bool {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == target {
			return true
		}
	}
	return false
}
