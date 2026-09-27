package codegen

type (
	// ResponseContractCaseData contains the transport metadata required to
	// render a generated HTTP response contract case.
	ResponseContractCaseData struct {
		// ID is the stable response contract case identifier.
		ID string
		// IsError reports whether the case describes a service error.
		IsError bool
		// Transport identifies the response protocol.
		Transport string
		// StatusCode is the exact declared HTTP status code.
		StatusCode int
		// ErrorName is the declared service error name.
		ErrorName string
		// ContentTypes lists the declared response media types.
		ContentTypes []string
		// RequiredHeaders lists the required response header names.
		RequiredHeaders []string
		// RequiredCookies lists the required response cookie names.
		RequiredCookies []string
		// Multipart describes the designed multipart request, if present.
		Multipart *MultipartRequestContractData
		// SSE describes stream assertions for an SSE success case.
		SSE *SSEResponseContractData
		// WebSocket describes stream assertions for a WebSocket success case.
		WebSocket *WebSocketResponseContractData
	}

	// MultipartRequestContractData contains generated multipart request metadata.
	MultipartRequestContractData struct {
		// ContentType is the request media type.
		ContentType string
		// Parts lists the designed multipart fields in body order.
		Parts []MultipartPartContractData
	}

	// MultipartPartContractData contains generated metadata for one request part.
	MultipartPartContractData struct {
		// Name is the multipart form field name.
		Name string
		// MediaType is the default media type for the part value.
		MediaType string
		// Required reports whether the request body requires the part.
		Required bool
	}

	// SSEResponseContractData contains generated SSE manifest metadata.
	SSEResponseContractData struct {
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
		// EventTypes lists allowed projection discriminator values.
		EventTypes []string
		// Terminal identifies expected stream completion behavior.
		Terminal string
	}

	// WebSocketResponseContractData contains generated WebSocket manifest metadata.
	WebSocketResponseContractData struct {
		// Direction is the designed stream direction.
		Direction string
		// InboundMessageType is the designed client-to-server message type name.
		InboundMessageType string
		// OutboundMessageType is the designed server-to-client message type name.
		OutboundMessageType string
		// HandshakeHeaders lists required WebSocket upgrade response headers.
		HandshakeHeaders []string
		// Terminal identifies expected stream completion behavior.
		Terminal string
	}
)
